package types

import (
	"encoding/json"
	"time"
)

// DelayQueueData struct
type DelayQueueData struct {
	Data        string    `json:"data"`        // the queue origin data
	DelayTime   int       `json:"delaytime"`   // the delay time, unit is second
	TriggerTime time.Time `json:"triggertime"` // the unix timestamp to trigger
}

// SerializeDelayQueueData serialize delay queue data for a auto run delay queue
func SerializeDelayQueueData(data string, delayTime int) ([]byte, error) {
	// Json encode
	delayQueueData := &DelayQueueData{
		Data:        data,
		DelayTime:   delayTime,
		TriggerTime: time.Now().Add(time.Duration(delayTime) * time.Second),
	}

	jsonStr, err := json.Marshal(delayQueueData)

	if nil != err {
		return []byte{}, err
	}

	return jsonStr, nil
}

// UnserializeDelayQueueData unserialize delay queue data for a auto run delay queue
func UnserializeDelayQueueData(runMode string, data string, delayOnFailure []int) (string, int, int, error) {
	queueData := ""
	nextDelayTime := 0
	delayTime := 0

	// log.Printf("runMode %s ", runMode)
	// log.Printf("data %s ", data)
	// log.Printf("delayOnFailure %v ", delayOnFailure)

	if "retry" == runMode {
		delayQueueData := &DelayQueueData{}
		err := json.Unmarshal([]byte(data), delayQueueData)
		if nil != err {
			return "", 0, 0, err
		}

		queueData = delayQueueData.Data

		if len(delayOnFailure) > 0 {
			for _, dt := range delayOnFailure {
				if delayQueueData.DelayTime < dt {
					nextDelayTime = dt
					break
				}
			}
		}
		delayTime = delayQueueData.DelayTime
	} else {
		queueData = data
		if len(delayOnFailure) > 0 {
			nextDelayTime = delayOnFailure[0]
		}
		delayTime = 0
	}
	// log.Printf("delayTime %d ", delayTime)
	// log.Printf("nextDelayTime %d \n", nextDelayTime)

	return queueData, nextDelayTime, delayTime, nil
}
