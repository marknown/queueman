package redis

import (
	"github.com/marknown/oredis"
)

// Queue to bind functions
type Queue struct{}

// CombineConfig for the redis
type CombineConfig struct {
	Config oredis.Config
	Queues []QueueConfig
}

// QueueConfig for redis queue
type QueueConfig struct {
	SourceType      string // queue type example "Redis"
	IsEnabled       bool   // this config is enabled
	IsDelayQueue    bool   // the queue is a delay queue
	IsDelayRaw      bool   // your format for redis zset delay queue, not the standard's DelayData
	QueueName       string // name of the queue
	DispatchURL     string // URL to dispatch
	DispatchTimeout int    // timeout for dispatch. 0 is unlimited
	Concurency      int    // dispatch concurency number
	DelayConcurency int    // delay queue dispatch concurency number
	DelayOnFailure  []int  // when failed delay seconds to retry
}

// QueueInstance a sigle queue configure with
type QueueInstance struct {
	Source oredis.Config
	Queue  QueueConfig
}
