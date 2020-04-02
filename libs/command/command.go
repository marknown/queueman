package command

import (
	"flag"
	"fmt"
	"os"
	"queueman/libs/config"
	"queueman/libs/constant"

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
		printStats(args)
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

	queueCount := 0
	// sort the DelayOnFailure array
	for _, r := range cfg.Redis {
		oredis.GetInstancePanic(r.Config)
		queueCount += len(r.Queues)
	}

	for _, r := range cfg.RabbitMQ {
		r.Config.GetConnectionPanic()
		queueCount += len(r.Queues)
	}

	if queueCount < 1 {
		panic("Please configure the Queue content")
	}

	// if -t
	if isJustTest {
		fmt.Printf("%s ok\n", message)
		os.Exit(0)
	}
}

// printStats print stats information
// tododeal 把反射改掉
func printStats(args *Args) {
	// todo 代码要改成最新的版本
	// c := config.GetConfig(args.ConfigFile)

	fmt.Printf("%s %s statistics information\n\n", appname, appversion)

	// todo 代码要改成最新的版本
	/*
		for _, q := range c.Redis {
			for _, qc := range q.Queues {
				// go Dispatcher(config, queueConfig)
				queueConnectionConfig := reflect.ValueOf(*c).FieldByName(qc.Source)
				if reflect.Invalid == queueConnectionConfig.Kind() {
					fmt.Printf("%s stats skipped reason is %s", qc.QueueName, fmt.Sprintf("configure file must have %s configure !!!\n", qc.Source))
					continue
				}

				q := &queue.Queue{
					Source: qc.Source,
					Config: queueConnectionConfig,
				}

				message := "         Details:\n"

				var queueItemsTotal int64 = 0
				if !qc.IsDelayQueue {
					queueItemsTotal, _ = q.Length(qc.QueueName)
				} else {
					queueItemsTotal, _ = q.DelayLength(qc.QueueName)
				}
				message += fmt.Sprintf("%10d item(s) in %s\n", queueItemsTotal, qc.QueueName)

				var delayItemsTotal int64 = 0
				if len(qc.DelayOnFailure) > 0 {
					for _, delayTime := range qc.DelayOnFailure {
						queueName := fmt.Sprintf("%s:delay:%d", qc.QueueName, delayTime)
						len1, _ := q.DelayLength(queueName)
						delayItemsTotal = delayItemsTotal + len1
						message += fmt.Sprintf("%10d item(s) in %s\n", len1, queueName)
					}
				}

				message = fmt.Sprintf("%10d remain\n\n", queueItemsTotal+delayItemsTotal) + message

				if v, ok := queue.QueueInstance[q.Source]; ok {
					cb := "IncrCounter"
					cbi := "GetCounter"
					fv := reflect.ValueOf(v)
					fc := fv.MethodByName(cb)
					fi := fv.MethodByName(cbi)
					fck := fc.Kind()
					fik := fi.Kind()

					if reflect.Func == fck && reflect.Func == fik {
						successTotal, _ := q.GetCounter(fmt.Sprintf("%s:success", qc.QueueName))
						failureTotal, _ := q.GetCounter(fmt.Sprintf("%s:failure", qc.QueueName))
						pushedTotal := successTotal + failureTotal + queueItemsTotal + delayItemsTotal
						message = fmt.Sprintf("%10d total\n%10d success\n%10d failure\n", pushedTotal, successTotal, failureTotal) + message
					}
				}

				fmt.Printf("   > %s stats:\n%s\n\n", qc.QueueName, message)
			}
		}
	*/

	os.Exit(0)
}
