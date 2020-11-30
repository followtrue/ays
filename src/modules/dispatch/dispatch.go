package dispatch

import (
	"encoding/json"
	"errors"
	"github.com/mikemintang/go-curl"
	"ays/src/models"
	"ays/src/modules/app"
	"ays/src/modules/ays_run"
	"ays/src/modules/graceful"
	"ays/src/modules/job_log"
	"ays/src/modules/logger"
	"ays/src/modules/mq"
	"ays/src/modules/register"
	"ays/src/modules/tools"
	"math/rand"
	"strconv"
	"strings"
)

// Mq调度
func MqDispatch(jobDetail *models.Job, jobParams string, jobLogMap models.JobLog) (bool, error) {
	result := true
	mqCli := mq.NewMqClient()
	node := handleRouter(jobDetail)
	if node == nil || node.Ip == "" {
		logger.Error("No Register nodes", jobDetail.JobAlias)
		jobLogMap.DispatchResult = "No Register nodes"
		jobLogMap.JobStatus = job_log.JOB_LOG_FAILED
		job_log.AddJobLog(jobLogMap)
		return false, errors.New("No Register nodes")
	}
	topic := tools.GetQueueName(node.Ip, node.Port)

	// 调度结束日志
	jobLogMap = job_log.JobLogStep(jobLogMap, job_log.DispatchEnd, *jobDetail, jobParams, node)
	jobLogId := job_log.AddJobLog(jobLogMap)
	jobLogMap.Id = jobLogId

	// 处理 jobParams 透传jobLog
	body := models.MqBody{}
	if jobParams != "" {
		json.Unmarshal([]byte(jobParams), &body)
	}
	body.JobLogId  = jobLogId
	body.JobDetail = jobDetail
	body.Params = jobParams
	jsonParams , _ := json.Marshal(body)

	message := mq.Msg {
		Topic: topic,
		Body: string(jsonParams),
		Tags: jobDetail.JobTag,
		Keys: node.Ip + strconv.Itoa(node.Port),
	}

	// 任务发送开始
	jobLogMap = job_log.JobLogStep(jobLogMap, job_log.JobSend, *jobDetail, jobParams, node)
	// 下发mq
	err := mqCli.SendMessageOrderly(message)
	if err != nil {
		logger.Error("Send Mq Job Failed", jobDetail, err, node)
		jobLogMap.JobResult = "Send Mq Job Failed!"
		jobLogMap.JobSendResult += tools.Substr(err.Error(), 0, 500)
		result = false
	}
	jobLogMap.Update()
	return result, err
}

// rpc调度
func RpcDispatch(jobDetail *models.Job, params map[string]map[string]interface {}, jobLogMap models.JobLog) (string, error) {
	graceful.AddJob() // 记录执行中的任务，进程退出前等待任务执行
	defer graceful.FinishJob() // 标记任务执行完成

	// 根据路由策略，确定执行器
	node := handleRouter(jobDetail)
	if node == nil || node.Ip == "" {
		logger.Error("No Register nodes", jobDetail, node)
		jobLogMap.DispatchResult = "No Register nodes"
		jobLogMap.JobStatus = job_log.JOB_LOG_FAILED
		job_log.AddJobLog(jobLogMap)
		return "", errors.New("No Register nodes")
	}

	command := jobDetail.Command
	if command == "" || strings.Contains(command, "rm ") {
		logger.Error("Job Command Invalid", jobDetail, node)
		jobLogMap.DispatchResult = "命令行不存在"
		jobLogMap.JobStatus = job_log.JOB_LOG_FAILED
		job_log.AddJobLog(jobLogMap)
		return "", errors.New("命令行不存在")
	}

	// 非定时任务拼接参数  1.定时任务，2非定时任务
	argv, argvHas := params["body"]["argv"]
	if jobDetail.JobType != 1 && argvHas && len(argv.([]interface{})) > 0{
		for _, v := range argv.([]interface{}) {
			command += " " + v.(string)
		}
		jobDetail.Command = command
	}

	// 调度结束日志
	jobLogMap = job_log.JobLogStep(jobLogMap, job_log.DispatchEnd, *jobDetail, "", node)

	// 执行开始日志
	jobLogMap = job_log.JobLogStep(jobLogMap, job_log.JobStart, *jobDetail, "", node)

	logger.Info("jobCommand:", jobDetail.Command)
	// 执行
	result, err := ays_run.CommandJob(*jobDetail, node.Ip, node.Port)

	// 执行结束日志
	jobLogMap = job_log.JobLogStep(jobLogMap, job_log.JobEnd, *jobDetail, "", node)
	if err != nil{
		logger.Error("Run Command Job Failed", jobDetail.JobAlias, node.Name)
		logger.IfError(err)
		jobLogMap.JobResult = "Run Command Job Failed!"
		jobLogMap.JobResult += tools.Substr(result, 0, 500)
		jobLogMap.JobStatus = job_log.JOB_LOG_FAILED
	} else {
		// 成功后发送子任务
		childList := jobDetail.ChildList
		if len(childList) > 0 && jobDetail.JobDepend == 2{
			for _, v := range childList {
				req := curl.NewRequest()
				url := app.Config.AYS.Host
				url += "/ays/v1/job/dispatch"

				req.SetHeaders(map[string]string{
					"Content-Type":"application/json",
				})
				childParams ,_ := json.Marshal(params)
				child  := string(childParams)
				req.SetPostData(map[string]interface{}{
					"job_alias" : v,
					"params" : child,
				})
				req.Send(url, "POST")
			}
		}
	}
	if jobLogMap.Id == 0 {
		job_log.AddJobLog(jobLogMap)
	} else {
		jobLogMap.Update()
	}

	return result, err
}

// local调度
func LocalDispatch(jobDetail *models.Job, params map[string]map[string]interface {}, jobLogMap models.JobLog) (string, error) {
	command := jobDetail.Command
	if command == "" || strings.Contains(command, "rm ") {
		logger.Error("Job Command Invalid", jobDetail)
		jobLogMap.DispatchResult = "命令行不存在"
		jobLogMap.JobStatus = job_log.JOB_LOG_FAILED
		job_log.AddJobLog(jobLogMap)
		return "", errors.New("命令行不存在")
	}

	// 非定时任务拼接参数  1.定时任务，2非定时任务
	argv, argvHas := params["body"]["argv"]
	if jobDetail.JobType != 1 && argvHas && len(argv.([]interface{})) > 0{
		for _, v := range argv.([]interface{}) {
			switch v.(type) {
			case int:
				command += " " + strconv.Itoa(v.(int))
			default:
				command += " " + v.(string)
			}
		}
		jobDetail.Command = command
	}

	// 调度结束日志
	jobLogMap = job_log.JobLogStep(jobLogMap, job_log.DispatchEnd, *jobDetail, "", nil)

	// 执行开始日志
	jobLogMap = job_log.JobLogStep(jobLogMap, job_log.JobStart, *jobDetail, "", nil)

	// 执行
	result, err := ays_run.LocalJob(*jobDetail)

	// 执行结束日志
	jobLogMap = job_log.JobLogStep(jobLogMap, job_log.JobEnd, *jobDetail, "", nil)
	if err != nil {
		logger.Error("Run Local Job Failed", jobDetail, err)
		jobLogMap.JobResult = "Run Local Job Failed!"
		jobLogMap.JobResult += tools.Substr(result, 0, 500)
		jobLogMap.JobStatus = job_log.JOB_LOG_FAILED
	} else {
		// 成功后发送子任务
		childList := jobDetail.ChildList
		if len(childList) > 0 && jobDetail.JobDepend == 2{
			for _, v := range childList {
				req := curl.NewRequest()
				url := app.Config.AYS.Host
				url += "/ays/v1/job/dispatch"

				req.SetHeaders(map[string]string{
					"Content-Type":"application/json",
				})
				childParams ,_ := json.Marshal(params)
				child  := string(childParams)
				req.SetPostData(map[string]interface{}{
					"job_alias" : v,
					"params" : child,
				})
				req.Send(url, "POST")
			}
		}
	}
	if jobLogMap.Id == 0 {
		job_log.AddJobLog(jobLogMap)
	} else {
		jobLogMap.Update()
	}

	return result, err
}

// http调度
func HttpDispatch(jobDetail *models.Job, params map[string]map[string]interface {}, jobLogMap models.JobLog) (bool, error){
	paramsHeader, headOk := params["header"]
	// 处理header
	headers := make(map[string]string)
	if headOk {
		for k, v := range paramsHeader {
			headers[k] = v.(string)
		}
	}
	paramsBody, bodyOk := params["body"]

	if jobDetail.HttpRequestUrl == ""{
		logger.Error("请求url不存在", jobDetail, params)
		jobLogMap.DispatchResult = "请求url不存在"
		jobLogMap.JobStatus = job_log.JOB_LOG_FAILED
		job_log.AddJobLog(jobLogMap)
		return false, errors.New("请求url不存在")
	}
	jobDetail.HttpRequestHeader = headers
	// http请求方式：1get，2post，3put，4delete，5patch
	httpMethod := jobDetail.HttpRequestType
	switch httpMethod {
	case 1:
		url := jobDetail.HttpRequestUrl
		if bodyOk {
			for k, v := range paramsBody {
				var separator string
				if strings.Contains(url, "?") {
					separator = "&"
				} else {
					separator = "?"
				}
				url = url + separator + k + "=" + v.(string)
			}
		}
		jobDetail.HttpRequestUrl = url
	//case 2:
	//case 3:
	//case 4:
	//case 5:
	default:
		jobDetail.HttpRequestBody = paramsBody
	}

	// 调度结束日志
	jobLogMap = job_log.JobLogStep(jobLogMap, job_log.DispatchEnd, *jobDetail, "", nil)
	// 执行开始日志
	jobLogMap = job_log.JobLogStep(jobLogMap, job_log.JobStart, *jobDetail, "", nil)

	// 执行
	result, err := ays_run.HttpJob(*jobDetail)

	// 执行结束日志
	jobLogMap = job_log.JobLogStep(jobLogMap, job_log.JobEnd, *jobDetail, "", nil)

	if !result || err != nil{
		logger.Error("Run Http Job Failed", jobDetail, err)
		jobLogMap.JobResult = "Run Http Job Failed!"
		if err != nil  {
			jobLogMap.JobResult += tools.Substr(err.Error(), 0, 500)
		}
		jobLogMap.JobStatus = job_log.JOB_LOG_FAILED
	} else {
		// 成功后发送子任务
		childList := jobDetail.ChildList
		if len(childList) > 0 && jobDetail.JobDepend == 2{
			for _, v := range childList {
				req := curl.NewRequest()
				url := app.Config.AYS.Host
				url += "/ays/v1/job/dispatch"

				req.SetHeaders(map[string]string{
					"Content-Type":"application/json",
				})
				childParams ,_ := json.Marshal(params)
				child  := string(childParams)
				req.SetPostData(map[string]interface{}{
					"job_alias" : v,
					"params" : child,
				})
				req.Send(url, "POST")
			}
		}
	}

	job_log.AddJobLog(jobLogMap)

	return result, err
}

// 路由策略
func handleRouter(jobDetail *models.Job) (*register.NodeView) {
	// 检查是否存在已注册执行服务器  http直接下发，不用验证注册器及路由策略
	node := register.NodeView{}
	nodeGroupAlias := jobDetail.NodeGroupAlias
	nodeList := register.NodeFindByGroup(nodeGroupAlias)
	nodeLength := len(nodeList)
	if nodeLength == 0 {
		logger.Error("No Register nodes", jobDetail, nodeGroupAlias)
		return &node
	}

	routeType := jobDetail.RouteType
	// 1=>第一个 2=>最后一个 3=>随机 4=>轮询
	switch routeType {
	case 1 :
		node = nodeList[0]
	case 2:
		node = nodeList[nodeLength -1]
	case 3:
		key := rand.Intn(nodeLength)
		if key > 0 {
			key = key -1
		} else {
			key =0
		}
		node = nodeList[key]
	default:
		key := rand.Intn(nodeLength)
		if key > 0 {
			key = key -1
		} else {
			key =0
		}
		node = nodeList[key]
	}
	return &node
}