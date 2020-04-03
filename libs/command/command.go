package command

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"queueman/libs/config"
	"queueman/libs/constant"
	"queueman/libs/queue/redis"
	"queueman/libs/statistic"
	"strings"

	"github.com/marknown/oredis"
)

// Args is command argv
type Args struct {
	ConfigFile string
	Help       bool
	Test       bool
	Stats      bool // show stats info
}

var appname string
var appversion string

// GetArgs Get args form command line
func GetArgs() *Args {
	appname = constant.APPNAME
	appversion = constant.APPVERSION

	args := &Args{}
	flag.BoolVar(&args.Help, "h", false, "show help information")
	flag.StringVar(&args.ConfigFile, "c", "./queueman.json", "the configure file path")
	flag.BoolVar(&args.Test, "t", false, `test configure in "queueman.json" file`)
	flag.BoolVar(&args.Stats, "s", false, "show statistics information")
	flag.Usage = printUsage
	flag.Parse()

	if args.Help {
		printUsage()
		return nil
	}

	if args.Test {
		printTest(args, true)
		return nil
	}

	printTest(args, false)

	// when configure file is right
	if args.Stats {
		info := GetStats(args, "")
		fmt.Println(info)
		os.Exit(0)
		return nil
	}

	return args
}

// printUsage print the useage
func printUsage() {
	message := `----------------------------------------------
	Welcome to Use %s %s
----------------------------------------------

usage:
`
	fmt.Printf(message, appname, appversion)
	flag.PrintDefaults()

	os.Exit(0)
}

// printTest print test configure results
func printTest(args *Args, isJustTest bool) {
	message := fmt.Sprintf(`%s %s configuration file %s test is `, appname, appversion, args.ConfigFile)

	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("%s wrong !!!\n> %s\n", message, err)
			os.Exit(0)
		}
	}()

	// Get config file content to struct
	cfg := config.GetConfig(args.ConfigFile)

	enabledQueueCount := 0
	// sort the DelayOnFailure array
	for _, r := range cfg.Redis {
		oredis.GetInstancePanic(r.Config)
		for _, n := range r.Queues {
			if n.IsEnabled {
				enabledQueueCount++
			}
		}
	}

	for _, r := range cfg.RabbitMQ {
		r.Config.GetConnectionPanic()
		for _, n := range r.Queues {
			if n.IsEnabled {
				enabledQueueCount++
			}
		}
	}

	if enabledQueueCount < 1 {
		panic(`There has no enabled queue, please check configure file "IsEnabled" fields for every queue`)
	}

	// if -t
	if isJustTest {
		fmt.Printf("%s ok\n", message)
		os.Exit(0)
	}
}

// GetStats get stats information format can be "text", "html", "json"
func GetStats(args *Args, format string) string {
	cfg := config.GetConfig(args.ConfigFile)

	allQueueStatistic := []*statistic.QueueStatistic{}

	for _, cc := range cfg.Redis {
		for _, queueConfig := range cc.Queues {
			s := &statistic.QueueStatistic{
				QueueName:  queueConfig.QueueName,
				SourceType: "Redis",
				IsEnabled:  queueConfig.IsEnabled,
			}

			qi := &redis.QueueInstance{
				Source: cc.Config,
				Queue:  queueConfig,
			}

			if queueConfig.IsDelayQueue {
				s.Normal, _ = qi.DelayLength(queueConfig.QueueName)
			} else {
				s.Normal, _ = qi.Length(queueConfig.QueueName)
			}

			if len(queueConfig.DelayOnFailure) > 0 {
				queueName := fmt.Sprintf("%s:delayed", queueConfig.QueueName)
				s.Delayed, _ = qi.DelayLength(queueName)
			}

			s.Success, _ = statistic.GetCounter(fmt.Sprintf("%s:success", queueConfig.QueueName))
			s.Failure, _ = statistic.GetCounter(fmt.Sprintf("%s:failure", queueConfig.QueueName))

			s.Total = s.Normal + s.Delayed + s.Success + s.Failure

			allQueueStatistic = append(allQueueStatistic, s)
		}
	}

	for _, cc := range cfg.RabbitMQ {
		for _, queueConfig := range cc.Queues {
			s := &statistic.QueueStatistic{
				QueueName:  queueConfig.QueueName,
				SourceType: "RabbitMQ",
				IsEnabled:  queueConfig.IsEnabled,
			}

			// qi := &rabbitmq.QueueInstance{
			// 	Source: cc.Config,
			// 	Queue:  queueConfig,
			// }
			// todo get queue length

			s.Normal = 0
			s.Delayed = 0

			s.Success, _ = statistic.GetCounter(fmt.Sprintf("%s:success", queueConfig.QueueName))
			s.Failure, _ = statistic.GetCounter(fmt.Sprintf("%s:failure", queueConfig.QueueName))

			s.Total = s.Normal + s.Delayed + s.Success + s.Failure

			allQueueStatistic = append(allQueueStatistic, s)
		}
	}

	if "json" == format {
		output, err := json.Marshal(allQueueStatistic)

		if nil != err {
			return ""
		}

		return string(output)
	}

	output := fmt.Sprintf("%s %s statistics information\n\n", constant.APPNAME, constant.APPVERSION)
	for _, s := range allQueueStatistic {
		status := "disable"
		if s.IsEnabled {
			status = "enable"
		}
		output += fmt.Sprintf("   > Type: %-8s Status: %-8s Name: %s\n%10d Total\n%10d Normal\n%10d Delayed\n%10d Success\n%10d Failure\n\n", s.SourceType, status, s.QueueName, s.Total, s.Normal, s.Delayed, s.Success, s.Failure)
	}

	if "html" == format {
		strings.Replace(output, "\n", "<br />", -1)
	}

	return output
}
