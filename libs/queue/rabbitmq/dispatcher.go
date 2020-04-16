package rabbitmq

import (
	"fmt"
	"queueman/libs/queue/types"
	"queueman/libs/request"
	"queueman/libs/statistic"
	"strings"
	"time"

	amqpReconnect "github.com/isayme/go-amqp-reconnect/rabbitmq"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// Dispatcher for Queue
func (queue *Queue) Dispatcher(combineConfig interface{}) {
	if cc, ok := combineConfig.(CombineConfig); ok {
		for _, queueConfig := range cc.Queues {
			if queueConfig.IsEnabled {
				queueConfig.SourceType = "RabbitMQ"
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
		}).Warn("Not correct rabbitmq config")
		return
	}
}

// GetConnection get a connect to source
func (qi *QueueInstance) GetConnection() (*amqpReconnect.Connection, error) {
	conn, err := qi.Source.GetConnection()
	if nil != err {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn("Can not connect to rabbitmq")
		return nil, err
	}

	return conn, nil
}

// QueueHandle handle Normal or Delay queue
func (qi *QueueInstance) QueueHandle() {
	if qi.Queue.Concurency < 1 {
		log.WithFields(log.Fields{
			"queueName": qi.Queue.QueueName,
		}).Warn("configure file of queue must have Concurency configure !!!")
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

	// wait ProcessDelay finished (to finished delay queue bind exchange queue)
	time.Sleep(1 * time.Second)

	if false == qi.Queue.IsDelayQueue {
		go qi.ProcessNormal(delayTime)
	} else {
		go qi.ProcessDelay("first")
	}

}

// ProcessNormal get a data from queue and dispatch
func (qi *QueueInstance) ProcessNormal(delayTime int) {
	conn, err := qi.GetConnection()
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	// ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args Table) error
	err = ch.ExchangeDeclare(qi.Queue.ExchangeName, qi.Queue.ExchangeType, qi.Queue.IsDurable, false, false, false, nil)
	failOnError(err, "Failed to Declare a exchange")

	// QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args Table) (Queue, error)
	q, err := ch.QueueDeclare(
		qi.Queue.QueueName, // name
		qi.Queue.IsDurable, // durable
		false,              // delete when unused
		false,              // exclusive
		false,              // no-wait
		nil,                // arguments
	)
	failOnError(err, "Failed to declare a queue"+q.Name)

	// QueueBind(name, key, exchange string, noWait bool, args Table) error
	err = ch.QueueBind(qi.Queue.QueueName, qi.Queue.RoutingKey, qi.Queue.ExchangeName, false, nil)
	failOnError(err, "Failed to bind a queue")

	// auto-ack exclusive
	// Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args Table) (<-chan Delivery, error)
	msgs, err := ch.Consume(
		qi.Queue.QueueName,   // queue
		qi.Queue.ConsumerTag, // consumer
		false,                // auto-ack
		false,                // exclusive
		false,                // no-local
		false,                // no-wait
		nil,                  // args
	)
	failOnError(err, "Failed to register a consumer")

	log.Printf("Start consume ( %s %s) queue (%s) from Exchange (%s) Vhost (%s)", qi.Queue.SourceType, qi.Source.Type, qi.Queue.QueueName, qi.Queue.ExchangeType, conn.Config.Vhost)

	// control the concurency
	concurency := make(chan bool, qi.Queue.Concurency)

	for delivery := range msgs {
		// control the concurency
		concurency <- true

		// dispatch to URLs
		go func(d amqp.Delivery) {
			queueData := string(d.Body)

			queueRequest := &request.QueueRequest{
				QueueName:       qi.Queue.QueueName,
				DispatchURL:     qi.Queue.DispatchURL,
				DispatchTimeout: qi.Queue.DispatchTimeout,
				QueueData:       queueData,
			}

			result, err := queueRequest.Post()

			status := ""
			if qi.Queue.IsAutoAck {
				d.Ack(false)
				status = "Auto acked"
				statistic.IncrSuccessCounter(qi.Queue.QueueName)
			} else {
				// failure
				if 1 != result.Code {
					log.WithFields(log.Fields{
						"err":       err,
						"Message":   result.Message,
						"queueName": qi.Queue.QueueName,
						"delayTime": delayTime,
					}).Warn("Request data result")

					nextDelayTime := delayTime
					if nextDelayTime > 0 {
						serializeDelayQueueData, err := types.SerializeDelayQueueData(queueData, nextDelayTime)

						if nil != err {
							log.WithFields(log.Fields{
								"error":   err,
								"Message": result.Message,
							}).Warn("Serialize DelayQueueData error")

							d.Ack(false) // when a error occur acked
							statistic.IncrFailureCounter(qi.Queue.QueueName)
							<-concurency // remove control
							return
						}

						queueNameDelay := fmt.Sprintf("%s.delayed", qi.Queue.QueueName)
						routingKeyDelay := fmt.Sprintf("%s.delayed", qi.Queue.RoutingKey)
						exchangeNameDelay := fmt.Sprintf("%s.delayed", qi.Queue.ExchangeName)

						// when fail push queue data to first delay queue
						header := amqp.Table{"x-delay": nextDelayTime * 1000}
						if "aliyun" == strings.ToLower(qi.Source.Type) {
							header = amqp.Table{"delay": nextDelayTime * 1000}
						}

						var deliveryMode uint8 = 1 // Transient (0 or 1) or Persistent (2)
						if qi.Queue.IsDurable {
							deliveryMode = 2
						}
						err = ch.Publish(
							exchangeNameDelay, // exchange
							routingKeyDelay,   // routing key
							false,             // mandatory
							false,             // immediate
							amqp.Publishing{
								Headers:      header,
								ContentType:  "text/plain",
								Body:         serializeDelayQueueData,
								DeliveryMode: deliveryMode,
							})

						if nil != err {
							log.WithFields(log.Fields{
								"error":        err,
								"queueName":    queueNameDelay,
								"exchangeName": exchangeNameDelay,
							}).Warn("Publish to rabbitmq failure")
						}
						status = "Normal Delayed"
						log.WithFields(log.Fields{
							"status":          status,
							"queueName":       queueNameDelay,
							"exchangeName":    exchangeNameDelay,
							"routingKeyDelay": routingKeyDelay,
							"trigglerTime":    time.Now().Add(time.Duration(nextDelayTime) * time.Second),
						}).Info("Delayed to queue")
					} else {
						status = "Normal Failure"
						statistic.IncrFailureCounter(qi.Queue.QueueName)
					}

					d.Ack(false)
				} else {
					status = "Normal Acked"
					statistic.IncrSuccessCounter(qi.Queue.QueueName)
					d.Ack(false)
				}
			}

			log.WithFields(log.Fields{
				"status":    status,
				"queueName": qi.Queue.QueueName,
				"queueData": queueData,
			}).Info("Messages from queue")

			<-concurency // remove control
		}(delivery)
	}
}

// ProcessDelay to deal with delay queue
func (qi *QueueInstance) ProcessDelay(runMode string) {
	queueName := ""
	queueNameDelay := ""
	routingKey := ""
	routingKeyDelay := ""
	exchangeName := ""
	exchangeNameDelay := ""
	if "retry" == runMode {
		queueName = fmt.Sprintf("%s.delayed", qi.Queue.QueueName)
		routingKey = fmt.Sprintf("%s.delayed", qi.Queue.RoutingKey)
		exchangeName = fmt.Sprintf("%s.delayed", qi.Queue.ExchangeName)
		queueNameDelay = queueName
		routingKeyDelay = routingKey
		exchangeNameDelay = exchangeName

	} else {
		// if runMode is first means it is the main delay queue
		queueName = qi.Queue.QueueName
		routingKey = qi.Queue.RoutingKey
		exchangeName = qi.Queue.ExchangeName
		queueNameDelay = fmt.Sprintf("%s.delayed", qi.Queue.QueueName)
		routingKeyDelay = fmt.Sprintf("%s.delayed", qi.Queue.RoutingKey)
		exchangeNameDelay = fmt.Sprintf("%s.delayed", qi.Queue.ExchangeName)
	}

	conn, err := qi.GetConnection()
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	// all queue is delay below ProcessDelay function
	// ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args Table) error
	if "aliyun" != strings.ToLower(qi.Source.Type) {
		err = ch.ExchangeDeclare(exchangeName, "x-delayed-message", qi.Queue.IsDurable, false, false, false, amqp.Table{"x-delayed-type": "direct"})
	} else {
		err = ch.ExchangeDeclare(exchangeName, qi.Queue.ExchangeType, qi.Queue.IsDurable, false, false, false, nil)
	}
	failOnError(err, "Failed to Declare a exchange")

	// QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args Table) (Queue, error)
	q, err := ch.QueueDeclare(
		queueName,          // name
		qi.Queue.IsDurable, // durable
		false,              // delete when unused
		false,              // exclusive
		false,              // no-wait
		nil,                // arguments
	)
	failOnError(err, "Failed to declare a queue"+q.Name)

	// QueueBind(name, key, exchange string, noWait bool, args Table) error
	err = ch.QueueBind(queueName, routingKey, exchangeName, false, nil)
	failOnError(err, "Failed to bind a queue")

	// auto-ack exclusive
	// Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args Table) (<-chan Delivery, error)
	msgs, err := ch.Consume(
		queueName,            // queue
		qi.Queue.ConsumerTag, // consumer
		false,                // auto-ack
		false,                // exclusive
		false,                // no-local
		false,                // no-wait
		nil,                  // args
	)
	failOnError(err, "Failed to register a consumer")

	log.Printf("Start consume ( %s %s) queue (%s) from Exchange (%s) Vhost (%s)", qi.Queue.SourceType, qi.Source.Type, queueName, qi.Queue.ExchangeType, conn.Config.Vhost)

	// control the concurency
	concurency := make(chan bool, qi.Queue.Concurency)

	for delivery := range msgs {
		// control the concurency
		concurency <- true

		// dispatch to URLs
		go func(d amqp.Delivery) {
			queueData, nextDelayTime, delayTime, err := types.UnserializeDelayQueueData(runMode, string(d.Body), qi.Queue.DelayOnFailure)
			if nil != err {
				log.WithFields(log.Fields{
					"error":        err,
					"queueName":    queueName,
					"exchangeName": exchangeName,
				}).Warn("DelayPop Unmarshal error")

				<-concurency // remove control
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
			if qi.Queue.IsAutoAck {
				d.Ack(false)
				status = "Auto acked"
				statistic.IncrSuccessCounter(qi.Queue.QueueName)
			} else {
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

							d.Ack(false) // when a error occur acked
							statistic.IncrFailureCounter(qi.Queue.QueueName)
							<-concurency // remove control
							return
						}

						// when fail push queue data to first delay queue
						header := amqp.Table{"x-delay": (nextDelayTime - delayTime) * 1000}
						if "aliyun" == strings.ToLower(qi.Source.Type) {
							header = amqp.Table{"delay": (nextDelayTime - delayTime) * 1000}
						}

						var deliveryMode uint8 = 1 // Transient (0 or 1) or Persistent (2)
						if qi.Queue.IsDurable {
							deliveryMode = 2
						}
						err = ch.Publish(
							exchangeNameDelay, // exchange
							routingKeyDelay,   // routing key
							false,             // mandatory
							false,             // immediate
							amqp.Publishing{
								Headers:      header,
								ContentType:  "text/plain",
								Body:         serializeDelayQueueData,
								DeliveryMode: deliveryMode,
							})

						if nil != err {
							log.WithFields(log.Fields{
								"error":        err,
								"queueName":    queueNameDelay,
								"exchangeName": exchangeNameDelay,
							}).Warn("Publish to rabbitmq failure")
						}

						status = "Delayed"

						log.WithFields(log.Fields{
							"status":          status,
							"queueName":       queueNameDelay,
							"exchangeName":    exchangeNameDelay,
							"routingKeyDelay": routingKeyDelay,
							"trigglerTime":    time.Now().Add(time.Duration(nextDelayTime-delayTime) * time.Second),
						}).Info("Delayed to queue")
					} else {
						status = "Failure"
						statistic.IncrFailureCounter(qi.Queue.QueueName)
					}

					d.Ack(false) // we alrealy pushed failure message to delay queue, so mark as acked
				} else {
					status = "Success"
					statistic.IncrSuccessCounter(qi.Queue.QueueName)
					d.Ack(false)
				}
			}

			log.WithFields(log.Fields{
				"status":    status,
				"queueName": queueName,
				"queueData": queueData,
			}).Info("Messages from queue")
			<-concurency // remove control
		}(delivery)
	}
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}
