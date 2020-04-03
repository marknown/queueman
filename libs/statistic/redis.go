package statistic

import (
	"github.com/gomodule/redigo/redis"
	"github.com/marknown/oredis"
)

type redisConfig struct {
	RedisSource oredis.Config
}

// IncrCounter increase counter number
func (c *redisConfig) IncrCounter(keyName string) (int64, error) {
	conn := oredis.GetInstance(c.RedisSource)
	defer conn.Close()

	reply, err := conn.Do("INCR", keyName)

	if nil != err {
		return 0, err
	}
	replyInt, err := redis.Int64(reply, err)
	if nil != err {
		return 0, err
	}

	return replyInt, nil
}

// GetCounter get counter number
func (c *redisConfig) GetCounter(keyName string) (int64, error) {
	conn := oredis.GetInstance(c.RedisSource)
	defer conn.Close()

	reply, err := conn.Do("GET", keyName)

	if nil != err {
		return 0, err
	}
	replyInt, err := redis.Int64(reply, err)
	if nil != err {
		return 0, err
	}

	return replyInt, nil
}
