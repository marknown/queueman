package statistic

import (
	"errors"
	"fmt"
	"strings"

	"github.com/marknown/oredis"
)

var packageConfig Config

// Config configure for statistic
type Config struct {
	HTTPPort    int
	SourceType  string
	RedisSource oredis.Config
}

// QueueStatistic queue statistic
type QueueStatistic struct {
	QueueName  string // 队列名称
	SourceType string // 队列类型
	IsEnabled  bool   // 队列是否启用
	Normal     int64  // 主队列未处理的总数
	Delayed    int64  // 延时队列未处理的总数
	Success    int64  // 成功数量
	Failure    int64  // 失败数量
	Total      int64  // 以上总数合计
}

// InitStatistic init statistic configure
func InitStatistic(c Config) {
	packageConfig = c
}

// IncrCounter Incr the special counter number
func IncrCounter(queueName string) (int64, error) {
	if "redis" != strings.ToLower(packageConfig.SourceType) {
		return 0, nil
	}

	if "redis" == strings.ToLower(packageConfig.SourceType) {
		obj := &redisConfig{
			RedisSource: packageConfig.RedisSource,
		}

		return obj.IncrCounter(queueName)
	}

	return 0, errors.New("Unsupport statistic source type")
}

// IncrSuccessCounter Incr the special counter number
func IncrSuccessCounter(queueName string) (int64, error) {
	return IncrCounter(fmt.Sprintf("%s:success", queueName))
}

// IncrFailureCounter Incr the special counter number
func IncrFailureCounter(queueName string) (int64, error) {
	return IncrCounter(fmt.Sprintf("%s:failure", queueName))
}

// GetCounter get the special counter number
func GetCounter(queueName string) (int64, error) {
	if "redis" == strings.ToLower(packageConfig.SourceType) {
		obj := &redisConfig{
			RedisSource: packageConfig.RedisSource,
		}

		return obj.GetCounter(queueName)
	}

	return 0, errors.New("Unsupport statistic source type")
}

// GetSuccessCounter get the special counter number
func GetSuccessCounter(queueName string) (int64, error) {
	return GetCounter(fmt.Sprintf("%s:success", queueName))
}

// GetFailureCounter get the special counter number
func GetFailureCounter(queueName string) (int64, error) {
	return GetCounter(fmt.Sprintf("%s:failure", queueName))
}
