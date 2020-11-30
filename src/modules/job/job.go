package job

import (
	"encoding/json"
	"errors"
	"gitlab.keda-digital.com/kedadigital/ays/src/models"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/dispatch"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/job_log"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/logger"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/tools"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/user"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/app"
)

const(
	LOCAL_DISPATCH int = 1		// 本地http方式
	RPC_DISPATCH   int = 2		// rpc方式
	MQ_DISPATCH    int = 3		// mq方式

	HTTP_RUN int = 1  			// http执行方式
	HTTP_COMMAND int = 2  		// command执行方式

	JOB_LOG_SUCCESS int = 2
	JOB_LOG_FAILED int  = 3
)

// 调度
func Dispatch(jobParams string, jobDetail *models.Job) (result bool, err error){
	if jobDetail.Id == 0 {
		logger.Error("Job Info Lost", jobDetail.JobAlias)
		return
	}
	logger.Info("start Dispatch", jobDetail.JobAlias)
	str := ""
	// 调度开始日志
	jobLogMap := models.JobLog{}
	jobLogMap = job_log.JobLogStep(jobLogMap, job_log.DispatchStart, *jobDetail,jobParams, nil)

	// 解析参数
	var params map[string]map[string]interface {}
	json.Unmarshal([]byte(jobParams), &params)
	result = true
	// 执行策略  1=>http  2=>rpc  3=>mq
	dispatchType := jobDetail.DispatchType
	switch dispatchType {
	case LOCAL_DISPATCH:
		jobLogMap.JobType = job_log.JobTypeHttp
		result, err = dispatch.HttpDispatch(jobDetail, params, jobLogMap)
	case RPC_DISPATCH:
		jobLogMap.JobType = job_log.JobTypeRpc
		str, err = dispatch.RpcDispatch(jobDetail, params, jobLogMap)
		if err != nil {
			result = false
		}
	case MQ_DISPATCH:
		jobLogMap.JobType = job_log.JobTypeMq
		result, err = dispatch.MqDispatch(jobDetail, jobParams, jobLogMap)
	default:
		jobLogMap.DispatchResult = "调度失败,任务执行策略不在调度范围之内"
		jobLogMap.JobStatus = job_log.JOB_LOG_FAILED
		job_log.AddJobLog(jobLogMap)
		result, err = false, errors.New("Dispatch Type Error")
		logger.Error("Dispatch Type Error", jobDetail.JobType)
		return
	}
	// 失败发送邮件、短信
	if jobDetail.JobAlias != "ays_email_send" && (!result || err != nil) {
		// 负责人
		userJobModel := new(models.UserJob)
		users, _ := userJobModel.GetUserByAlias(jobDetail.JobAlias)
		callEmails := []string{
			"lizhitao@kedabeijing.com",
			"zhangtianjiao@kedabeijing.com",
			"liupeng@kedabeijing.com",
			"qina@kedabeijing.com",
		}
		userEmails := []string{
			"wangwen@kedabeijing.com",
		}
		for _, v := range users {
			userInfo := user.GetUserInfo(v.UserId)
			userEmails = append(userEmails, userInfo.Email)
		}
		env := app.Config.ENV

		data := "任务Alias:" + jobDetail.JobAlias
		data += "<br/>任务名称:" + jobDetail.JobName
		data += "<br/>环境:" + env
		data += "<br/>状态:失败"
		data += "<br/>"+err.Error() + str
		emailParams := map[string]interface{}{
			"header": "",
			"body"	: map[string]interface{}{
				"type" 		: "email",
				"platform"	: "ays",
				"body"		: map[string]interface{}{
					"to"		: userEmails,
					"subject" 	: "π智-AYS任务系统-任务执行结果通知",
					"data" 		: data,
					"sign"		: "派智系统",
					"call" 		: callEmails,
				},
			},
		}
		host := app.Config.AYS.Host

		tools.SendEmail(emailParams, host)
	}
	return
}