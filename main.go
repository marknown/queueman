package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"queueman/libs/command"
	"queueman/libs/config"
	"queueman/libs/constant"
	"queueman/libs/queue"
	"queueman/libs/statistic"
	"queueman/libs/utils"
	"runtime/debug"
	"time"

	"github.com/docker/docker/pkg/pidfile"
	log "github.com/sirupsen/logrus"
)

func initLog() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func setLogFormatter(formatter string) {
	if "json" == formatter {
		customFormatter := new(log.JSONFormatter)
		customFormatter.TimestampFormat = "2006-01-02 15:04:05"
		log.SetFormatter(customFormatter)
	} else {
		customFormatter := new(log.TextFormatter)
		customFormatter.TimestampFormat = "2006-01-02 15:04:05"
		customFormatter.FullTimestamp = true
		log.SetFormatter(customFormatter)
	}
}

func setLogFile(logDir string) {
	fileName := fmt.Sprintf("%s/queueman.%s.log", logDir, utils.NowDateStringCN())
	var file, err = os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"fileName": fileName,
		}).Fatal(`Could Not Open Log File`)
	} else {
		log.SetOutput(file)
	}
}

func main() {
	// init logger
	initLog()

	// Get command args
	args := command.GetArgs()

	// Get config file content to struct
	cfg := config.GetConfig(args.ConfigFile)
	// init statistic for record
	statistic.InitStatistic(cfg.Statistic)

	if !cfg.App.IsDebug {
		log.SetLevel(log.WarnLevel)
	}

	if "" == cfg.App.LogFormatter {
		cfg.App.LogFormatter = "text"
	}
	setLogFormatter(cfg.App.LogFormatter)

	// write log to file
	if "" != cfg.App.LogDir {
		if !utils.IsDir(cfg.App.LogDir) {
			log.Warn("Log file dir is not exist or not a direcotry, please check it in your configure file.")
		} else {
			setLogFile(cfg.App.LogDir)
			go func() {
				for {
					select {
					case <-time.After(60 * time.Second):
						setLogFile(cfg.App.LogDir)
					}
				}
			}()
		}
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

	// out put every time started
	log.Warnf("%s %s started at %s.", constant.APPNAME, constant.APPVERSION, utils.NowTimeStringCN())

	for _, config := range cfg.Redis {
		go queue.QFactory("Redis").Dispatcher(config)
	}

	for _, config := range cfg.RabbitMQ {
		go queue.QFactory("RabbitMQ").Dispatcher(config)
	}

	if cfg.Statistic.HTTPPort > 0 {
		http.HandleFunc("/statistic", func(w http.ResponseWriter, r *http.Request) {
			format := r.FormValue("format")
			if "json" != format {
				format = "html"
			}

			fmt.Fprint(w, command.GetStats(args, format))
			return
		})

		err = http.ListenAndServe(fmt.Sprintf(":%d", cfg.Statistic.HTTPPort), nil)

		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("%s can not listening on port %d", constant.APPNAME, cfg.Statistic.HTTPPort)
		}
	}

	for {
		time.Sleep(5 * time.Second)
	}
}
