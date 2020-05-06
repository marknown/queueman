package request

import (
	"encoding/json"
	"queueman/libs/constant"
	"queueman/libs/ohttp"
	"time"

	log "github.com/sirupsen/logrus"
)

// QueueResponse for queue request
type QueueResponse struct {
	Code     int    // return 1 success, and 0 failure
	Message  string // when  code is 1 then message is "ok" otherwise message can be other info
	HTTPCode int    // the HTTPCode for request
}

// QueueRequest for queue dispather
type QueueRequest struct {
	QueueName       string // name of the queue
	DelayQueueName  string // name of the delay queue
	DispatchURL     string // URL to dispatch
	DispatchTimeout int    // timeout for dispatch. 0 is unlimited
	QueueData       string // data for one element form queue
	UserAgent       string // user agent for request
}

// Post for QueueRequest struct
func (req *QueueRequest) Post() (*QueueResponse, error) {
	settings := ohttp.InitSetttings()
	settings.Timeout = time.Duration(req.DispatchTimeout) * time.Second
	settings.UserAgent = constant.APPNAME + " " + constant.APPVERSION

	params := map[string]string{
		"queueName": req.QueueName,
		"delayName": req.DelayQueueName,
		"queueData": req.QueueData,
	}

	content, response, err := settings.Post(req.DispatchURL, params)

	result := &QueueResponse{
		HTTPCode: 0,
	}

	// fixed a bug. when err is occour the response is nil, response.StatusCode will panic
	if err != nil {
		log.WithFields(log.Fields{
			"err":       err,
			"queueData": req.QueueData,
			"Response":  content,
		}).Warn("Request post error")

		result.Code = 0
		result.Message = content
		return result, err
	}

	// init HTTPCode
	result.HTTPCode = response.StatusCode

	err = json.Unmarshal([]byte(content), &result)

	if nil != err {
		log.WithFields(log.Fields{
			"err":       err,
			"queueData": req.QueueData,
			"Response":  content,
		}).Warn("Request unmarshal error")

		result.Code = 0
		result.Message = content
		return result, err
	}

	return result, nil
}
