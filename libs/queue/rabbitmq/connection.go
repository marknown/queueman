package rabbitmq

import (
	"crypto/md5"
	"errors"
	"fmt"
	"queueman/libs/aliyun"
	"strings"
	"sync"

	amqpReconnect "github.com/isayme/go-amqp-reconnect/rabbitmq"
)

// Config configure for rabbitmq
type Config struct {
	Scheme       string // amqp or amqps
	Host         string
	Port         int32
	User         string
	Password     string
	Vhost        string
	Type         string // default is standard,other options is aliyun
	AliyunParams aliyun.Config
}

// 包内变量，存储实例相关对象
var packageOnce = map[string]*sync.Once{}
var packageInstance = map[string]*amqpReconnect.Connection{}
var packageMutex = &sync.Mutex{}

// URL get rabbitmq amqp url
func (config *Config) URL() string {
	if "aliyun" == strings.ToLower(config.Type) {
		config.User = config.AliyunParams.GetUserName()
		config.Password = config.AliyunParams.GetPassword()
	}

	return fmt.Sprintf("%s://%s:%s@%s:%d/%s", config.Scheme, config.User, config.Password, config.Host, config.Port, config.Vhost)
}

// GetConnection get a rabbitmq Connection
func (config *Config) GetConnection() (*amqpReconnect.Connection, error) {
	// TODODEL
	// amqpReconnect.Debug = true
	packageMutex.Lock()
	defer packageMutex.Unlock()

	md5byte := md5.Sum([]byte(fmt.Sprintf("%s%s%d%s%s%s", config.Scheme, config.Host, config.Port, config.User, config.Password, config.Vhost)))
	md5key := fmt.Sprintf("%x", md5byte)

	// 如果有值直接返回
	if v, ok := packageInstance[md5key]; ok {
		// fmt.Println("direct")
		return v, nil
	}

	// 如果once 不存在
	if _, ok := packageOnce[md5key]; !ok {
		var once = &sync.Once{}
		var conn *amqpReconnect.Connection
		var err error
		// var err error
		once.Do(func() {
			conn, err = amqpReconnect.Dial(config.URL())

			if nil == err {
				packageInstance[md5key] = conn
				packageOnce[md5key] = once
			}
		})

		if nil != err {
			return nil, err
		}

		return conn, nil
	}

	return nil, errors.New("RabbitMQ get connection error")
}

// GetConnectionPanic get a rabbitmq Connection and panic when error occurred
func (config *Config) GetConnectionPanic() *amqpReconnect.Connection {
	conn, err := config.GetConnection()

	if nil != err {
		panic("RabbitMQ " + err.Error())
	}

	return conn
}
