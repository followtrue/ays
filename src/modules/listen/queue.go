package listen

import (
	"encoding/json"
	"github.com/apache/rocketmq-client-go/core"
	"ays/src/models"
	"ays/src/modules/dispatch"
	"ays/src/modules/logger"
	"ays/src/modules/mq"
)


// 监听队列消息并执行
func ListenQueue(queueName string) {
	client := mq.NewMqClient()
	client.LoopPull(queueName, func(message *rocketmq.Message) {
		body:=  models.MqBody{}
		json.Unmarshal([]byte(message.Body), &body)
		jobDetail := body.JobDetail
		params := body.Params

		bodyParams := make(map[string]map[string]interface {})
		if params != "" {
			json.Unmarshal([]byte(params), &bodyParams)
		}
		jobLogId := body.JobLogId

		jobLogDetail := new(models.JobLog)
		jobLogDetail.Find(jobLogId)


		_, err :=dispatch.LocalDispatch(jobDetail, bodyParams, *jobLogDetail)
		if err != nil {
			logger.Error("执行失败", jobDetail, params)
		}

	})
}