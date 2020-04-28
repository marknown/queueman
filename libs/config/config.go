package config

import (
	"queueman/libs/queue/rabbitmq"
	"queueman/libs/queue/redis"
	"queueman/libs/statistic"
	"sort"
	"sync"

	"github.com/marknown/oconfig"
)

// App configure for the app
type App struct {
	IsDebug bool   // is debug mode
	PIDFile string // PIDFile path
	LogDir  string // Log directory
}

// Config for the file
type Config struct {
	App       App
	Statistic statistic.Config
	Redis     []redis.CombineConfig
	RabbitMQ  []rabbitmq.CombineConfig
}

var once = &sync.Once{}
var lock = &sync.Mutex{}
var packageConfigInstance *Config

// GetConfig only init one time
func GetConfig(configPath string) *Config {
	lock.Lock()
	defer lock.Unlock()

	if nil != packageConfigInstance {
		return packageConfigInstance
	}

	once.Do(func() {
		packageConfigInstance = &Config{}
		oconfig.GetConfig(configPath, packageConfigInstance)

		// sort the DelayOnFailure array
		for _, qc := range packageConfigInstance.Redis {
			for _, qcc := range qc.Queues {
				sort.Ints(qcc.DelayOnFailure)
			}
		}
	})

	return packageConfigInstance
}
