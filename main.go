package main

import (
	"errors"
	"os"
	"queueman/libs/command"
	"queueman/libs/config"
	"queueman/libs/queue"
	"runtime/debug"
	"time"

	"github.com/docker/docker/pkg/pidfile"
	log "github.com/sirupsen/logrus"
)

func init() {
	// Log as JSON instead of the default ASCII formatter.
	// log.SetFormatter(&log.JSONFormatter{})
	// log.SetFormatter(&log.TextFormatter{})

	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)
	// var file, err = os.OpenFile("./log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	// if err != nil {
	// 	log.WithFields(log.Fields{
	// 		"error": err,
	// 	}).Fatal(`Could Not Open Log File`)
	// }
	// log.SetOutput(file)

	// Only log the warning severity or above.
	// log.SetLevel(log.WarnLevel)
	log.SetLevel(log.InfoLevel)
}

func main() {
	// Get command args
	args := command.GetArgs()

	// Get config file content to struct
	cfg := config.GetConfig(args.ConfigFile)

	if !cfg.App.IsDebug {
		log.SetLevel(log.WarnLevel)
	}

	// add the pidfile
	if "" == cfg.App.PIDFile {
		log.WithFields(log.Fields{
			"error": errors.New("PIDFile option not configure in the configure file"),
		}).Fatal(`Please check the "PIDFile" option in configure file`)
	}

	// check & put the pid to the pid file
	pidHandle, err := pidfile.New(cfg.App.PIDFile)
	if nil != err {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("A pidfile error has occurred")
	}

	// catch the error
	defer func() {
		if err := recover(); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("A error has occurred!")
			debug.PrintStack()

			if nil != pidHandle {
				err1 := pidHandle.Remove()
				if nil != err1 {
					log.WithFields(log.Fields{
						"error": err1,
					}).Error("The pid file can not be deleted!")
				}
			}

			os.Exit(1)
		}
	}()

	for _, config := range cfg.Redis {
		go queue.QFactory("Redis").Dispatcher(config)
	}

	for _, config := range cfg.RabbitMQ {
		go queue.QFactory("RabbitMQ").Dispatcher(config)
	}

	for {
		time.Sleep(5 * time.Second)
	}
}
