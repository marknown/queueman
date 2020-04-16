package redis

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/gomodule/redigo/redis"
	uuid "github.com/satori/go.uuid"
)

// DelayData struct
type DelayData struct {
	UUID string `json:"uuid"` // UUID for delay value
	Time int64  `json:"time"` // the unix timestamp to trigger
	Data string `json:"data"` // the queue origin data
}

// Pop a element
func (qi *QueueInstance) Pop(queueName string) (string, error) {
	conn, _ := qi.GetConnection()
	defer conn.Close()

	reply, err := conn.Do("RPOP", queueName)

	if nil != err {
		return "", err
	}
	replyStr, err := redis.String(reply, err)
	if nil != err {
		return "", err
	}

	return replyStr, nil
}

// Push a element
func (qi *QueueInstance) Push(queueName string, value string) (bool, error) {
	conn, _ := qi.GetConnection()
	defer conn.Close()

	reply, err := conn.Do("LPUSH", queueName, value)

	if nil != err {
		return false, err
	}
	replyInt, err := redis.Int(reply, err)
	if nil != err {
		return false, err
	}

	if replyInt < 1 {
		return false, errors.New("Push 0 element to queue")
	}

	return true, nil
}

// Length a queue's length
func (qi *QueueInstance) Length(queueName string) (int64, error) {
	conn, _ := qi.GetConnection()
	defer conn.Close()

	reply, err := conn.Do("LLen", queueName)

	if nil != err {
		return 0, err
	}
	replyInt, err := redis.Int64(reply, err)
	if nil != err {
		return 0, err
	}

	return replyInt, nil
}

// DelayPop pop data from delay queue
func (qi *QueueInstance) DelayPop(queueName string, isReturnRaw bool) ([]string, error) {
	conn, _ := qi.GetConnection()
	defer conn.Close()

	min := 0
	max := time.Now().Unix()
	// todo when number is to large, how to deal with
	// zrangebyscore test 0 2 LIMIT 2 4
	conn.Send("MULTI")
	conn.Send("ZRANGEBYSCORE", queueName, min, max)
	conn.Send("ZREMRANGEBYSCORE", queueName, min, max)
	reply, err := conn.Do("EXEC")

	result := []string{}
	replys, err := redis.MultiBulk(reply, err)
	if nil != err {
		return result, err
	}

	// not resut, return empty result
	if len(replys) <= 0 || replys[1].(int64) <= 0 {
		return result, nil
	}

	// has result but length less than 0
	replys2, err := redis.MultiBulk(replys[0], err)
	if len(replys2) <= 0 {
		return result, nil
	}

	for _, v := range replys2 {
		if isReturnRaw {
			result = append(result, string(v.([]byte)))
		} else {
			vv := &DelayData{}
			err := json.Unmarshal(v.([]byte), vv)
			if nil != err {
				continue
			}

			result = append(result, vv.Data)
		}
	}

	return result, err
}

// DelayPush push a element to a delayqueue
func (qi *QueueInstance) DelayPush(queueName string, value string, delayUnixTime int64) (bool, error) {
	conn, _ := qi.GetConnection()
	defer conn.Close()

	uniqueID := uuid.NewV4().String()

	// Json encode
	delayData := &DelayData{
		UUID: uniqueID,
		Time: delayUnixTime,
		Data: value,
	}

	jsonStr, err := json.Marshal(delayData)

	if nil != err {
		return false, err
	}

	reply, err := conn.Do("ZADD", queueName, "NX", delayUnixTime, jsonStr)

	if nil != err {
		return false, err
	}
	replyInt, err := redis.Int(reply, err)
	if nil != err {
		return false, err
	}

	if replyInt < 1 {
		return false, errors.New("Push 0 element to delay queue")
	}

	return true, nil
}

// DelayLength a delay queue's length
func (qi *QueueInstance) DelayLength(queueName string) (int64, error) {
	conn, _ := qi.GetConnection()
	defer conn.Close()

	reply, err := conn.Do("ZCOUNT", queueName, "-INF", "+INF")

	if nil != err {
		return 0, err
	}
	replyInt, err := redis.Int64(reply, err)
	if nil != err {
		return 0, err
	}

	return replyInt, nil
}
