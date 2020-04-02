package rabbitmq

// Queue to bind functions
type Queue struct{}

// CombineConfig for the RabbitMQ
type CombineConfig struct {
	Config Config
	Queues []QueueConfig
}

// QueueConfig for RabbitMQ queue
type QueueConfig struct {
	SourceType      string // queue type example "RabbitMQ"
	IsEnabled       bool   // this config is enabled
	IsDelayQueue    bool   // the queue is a delay queue
	IsDurable       bool   // true Durable false not
	ExchangeName    string // name of the exchange
	ExchangeType    string // type of the exchange
	QueueName       string // name of the queue
	RoutingKey      string // name of the routing key
	ConsumerTag     string // name of the consumer tag
	IsAutoAck       bool   // true for auto ack
	DispatchURL     string // URL to dispatch
	DispatchTimeout int    // timeout for dispatch. 0 is unlimited
	Concurency      int    // dispatch concurency number
	DelayConcurency int    // delay queue dispatch concurency number
	DelayOnFailure  []int  // when failed delay seconds to retry
}

// QueueInstance a sigle queue configure with
type QueueInstance struct {
	Source Config
	Queue  QueueConfig
}
