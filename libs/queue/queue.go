package queue

import (
	"queueman/libs/queue/rabbitmq"
	"queueman/libs/queue/redis"
)

// QInterface queue interface
type QInterface interface {
	Dispatcher(queueConfig interface{})
}

// QFactory queue factory
func QFactory(queueType string) QInterface {
	switch queueType {
	case "RabbitMQ":
		return &rabbitmq.Queue{}
	case "Redis":
		return &redis.Queue{}
	}

	return nil
}
