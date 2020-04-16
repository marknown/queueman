package redis

import (
	"fmt"
	"queueman/libs/queue/types"
	"queueman/libs/request"
	"queueman/libs/statistic"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/marknown/oredis"
	log "github.com/sirupsen/logrus"
)

// Dispatcher for Queue
func (queue *Queue) Dispatcher(combineConfig interface{}) {
	if cc, ok := combineConfig.(CombineConfig); ok {
		for _, queueConfig := range cc.Queues {
			if queueConfig.IsEnabled {
				queueConfig.SourceType = "Redis"
				// enabledTotal++ todo
				qi := &QueueInstance{
					Source: cc.Config,
					Queue:  queueConfig,
				}
				go qi.QueueHandle()
			}
		}
	} else {
		log.WithFields(log.Fields{
			"queueConfig": combineConfig,
		}).Warn("Not correct redis config")

		return
	}
}

// GetConnection get a connect to source
func (qi *QueueInstance) GetConnection() (redis.Conn, error) {
	conn := oredis.GetInstance(qi.Source)

	return conn, nil
}

// QueueHandle handle Normal or Delay queue
func (qi *QueueInstance) QueueHandle() {
	if qi.Queue.Concurency < 1 {
		log.WithFields(log.Fields{
			"queueName": qi.Queue.QueueName,
		}).Warn("Configure file of queue must have Concurency configure !!!")
		return
	}

	// process main queue
	// get the delay time
	delayTime := 0
	if len(qi.Queue.DelayOnFailure) > 0 {
		delayTime = qi.Queue.DelayOnFailure[0]
	}

	// first process delay queue
	totalDelays := len(qi.Queue.DelayOnFailure)
	if totalDelays > 0 {
		go qi.ProcessDelay("retry")
	}

	// wait ProcessDelay finished
	time.Sleep(1 * time.Second)

	if false == qi.Queue.IsDelayQueue {
		go qi.ProcessNormal(delayTime)
	} else {
		go qi.ProcessDelay("first")
	}
}

// ProcessNormal get a data from queue and dispatch
func (qi *QueueInstance) ProcessNormal(delayTime int) {
	// log.WithFields(log.Fields{
	// 	"queueName": qi.Queue.QueueName,
	// }).Info("Queue process begin")
	log.Printf("Start consume ( %s ) queue (%s)", qi.Queue.SourceType, qi.Queue.QueueName)

	// control the concurency
	concurency := make(chan bool, qi.Queue.Concurency)

	for {
		// control the concurency
		concurency <- true

		queueData, err := qi.Pop(qi.Queue.QueueName)
		if nil != err {
			<-concurency // remove control

			if nil != err && "redigo: nil returned" != err.Error() {
				log.WithFields(log.Fields{
					"queueName":                   qi.Queue.QueueName,
					"Process handler has a error": err.Error(),
				}).Warn("!!! ProcessDelay handler has a error")
			}

			time.Sleep(1 * time.Second)
			continue
		}

		log.WithFields(log.Fields{
			"queueData": queueData,
		}).Info("Get data from queue")

		// dispatch to URLs
		go func(queueData string) {
			queueRequest := &request.QueueRequest{
				QueueName:       qi.Queue.QueueName,
				DispatchURL:     qi.Queue.DispatchURL,
				DispatchTimeout: qi.Queue.DispatchTimeout,
				QueueData:       queueData,
			}

			result, err := queueRequest.Post()

			status := ""
			// failure
			if 1 != result.Code {
				log.WithFields(log.Fields{
					"err":       err,
					"Message":   result.Message,
					"queueName": qi.Queue.QueueName,
					"delayTime": delayTime,
				}).Warn("Request data result")

				if delayTime > 0 {
					// when fail push queue data to first delay queue
					serializeDelayQueueData, err := types.SerializeDelayQueueData(queueData, delayTime)

					if nil != err {
						log.WithFields(log.Fields{
							"error":   err,
							"Message": result.Message,
						}).Warn("Serialize DelayQueueData error")

						statistic.IncrFailureCounter(qi.Queue.QueueName)
						<-concurency // remove control
						return
					}

					qi.DelayPush(fmt.Sprintf("%s:delayed", qi.Queue.QueueName), string(serializeDelayQueueData), time.Now().Unix()+int64(delayTime))

					status = "Normal Delayed"
					log.WithFields(log.Fields{
						"status":       status,
						"queueName":    qi.Queue.QueueName,
						"trigglerTime": time.Now().Add(time.Duration(delayTime) * time.Second),
					}).Info("Delayed to queue")
				} else {
					// finally also failure
					status = "Normal Failure"
					statistic.IncrFailureCounter(qi.Queue.QueueName)
				}
			} else {
				status = "Normal Acked"
				statistic.IncrSuccessCounter(qi.Queue.QueueName)
			}

			log.WithFields(log.Fields{
				"status":    status,
				"queueName": qi.Queue.QueueName,
				"queueData": queueData,
			}).Info("Messages from queue")

			<-concurency // remove control
		}(queueData)
	}
}

// ProcessDelay to deal with delay queue
func (qi *QueueInstance) ProcessDelay(runMode string) {
	queueName := ""
	queueNameDelay := ""
	if "retry" == runMode {
		queueName = fmt.Sprintf("%s:delayed", qi.Queue.QueueName)
		queueNameDelay = queueName
	} else {
		// if runMode is first means it is the main delay queue
		queueName = qi.Queue.QueueName
		queueNameDelay = fmt.Sprintf("%s:delayed", qi.Queue.QueueName)
	}

	// log.WithFields(log.Fields{
	// 	"queueName": queueName,
	// }).Info("Queue process begin")
	log.Printf("Start consume ( %s ) queue (%s)", qi.Queue.SourceType, queueName)

	// control the concurency
	concurency := make(chan bool, qi.Queue.DelayConcurency)

	for {
		// control the concurency
		concurency <- true

		// runMode is "first" and is main queue and configure file's IsDelayRaw is true
		isReturnRaw := false
		if "first" == runMode && qi.Queue.IsDelayRaw {
			isReturnRaw = true
		}
		queueDatas, err := qi.DelayPop(queueName, isReturnRaw)

		if nil != err || len(queueDatas) < 1 {
			<-concurency // remove control

			if nil != err && "redigo: nil returned" != err.Error() {
				log.WithFields(log.Fields{
					"queueName":                   queueName,
					"Process handler has a error": err.Error(),
				}).Warn("!!! ProcessDelay handler has a error")
			}

			time.Sleep(1 * time.Second)
			continue
		}

		<-concurency // remove control

		log.WithFields(log.Fields{
			"queueName": queueName,
			"length":    len(queueDatas),
		}).Info("Get datas from delay queue")

		for _, queueData := range queueDatas {
			// control the concurency
			concurency <- true

			// dispatch to URLs
			go func(d string) {
				queueData, nextDelayTime, delayTime, err := types.UnserializeDelayQueueData(runMode, string(d), qi.Queue.DelayOnFailure)
				if nil != err {
					log.WithFields(log.Fields{
						"error":     err,
						"queueName": queueName,
					}).Warn("DelayPop Unmarshal error")
					return
				}

				log.WithFields(log.Fields{
					"queueName": queueName,
					"queueData": queueData,
				}).Info("Queue data")

				queueRequest := &request.QueueRequest{
					QueueName:       qi.Queue.QueueName,
					DelayQueueName:  queueName,
					DispatchURL:     qi.Queue.DispatchURL,
					DispatchTimeout: qi.Queue.DispatchTimeout,
					QueueData:       queueData,
				}

				result, err := queueRequest.Post()

				status := ""
				// failure
				if 1 != result.Code {
					log.WithFields(log.Fields{
						"err":           err,
						"Message":       result.Message,
						"queueName":     queueName,
						"nextDelayTime": nextDelayTime,
						"delayTime":     delayTime,
					}).Warn("Request data result")

					if nextDelayTime > 0 {
						serializeDelayQueueData, err := types.SerializeDelayQueueData(queueData, nextDelayTime)

						if nil != err {
							log.WithFields(log.Fields{
								"error":   err,
								"Message": result.Message,
							}).Warn("Serialize DelayQueueData error")

							statistic.IncrFailureCounter(qi.Queue.QueueName)
							<-concurency // remove control
							return
						}

						// when fail push queue data to first delay queue
						qi.DelayPush(queueNameDelay, string(serializeDelayQueueData), time.Now().Unix()+int64(nextDelayTime)-int64(delayTime))

						status = "Delayed"
						log.WithFields(log.Fields{
							"status":       status,
							"queueName":    queueNameDelay,
							"trigglerTime": time.Now().Add(time.Duration(nextDelayTime-delayTime) * time.Second),
						}).Info("Delayed to queue")
					} else {
						// finally also failure
						status = "Failure"
						statistic.IncrFailureCounter(qi.Queue.QueueName)
					}
				} else {
					status = "Success"
					statistic.IncrSuccessCounter(qi.Queue.QueueName)
				}

				log.WithFields(log.Fields{
					"status":    status,
					"queueName": queueName,
					"queueData": queueData,
				}).Info("Messages from queue")

				<-concurency // remove control
			}(queueData)
		}
	}
}
