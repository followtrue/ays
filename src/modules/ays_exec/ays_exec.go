package ays_exec

import (
	"ays/src/models"
	"ays/src/modules/ays_run"
	"ays/src/modules/job_log"
	"ays/src/modules/logger"
	"strconv"
	"time"
)

//依赖
const (
	JobDependStrong  int = 1	//强依赖
	JobDependWeak int = 2		//弱依赖
)
//执行方式
const (
	RunTypeHttp int = 1			//HTTP
	RunTypeCommand int = 2		//Command
)

//执行参数
type ExecParams struct {
	ip string
	port int
	jobLogId uint64
}

func ExecJob(jobDetail models.Job, params ExecParams) (result bool, err error) {
	//任务执行
	//logger.Info("job_detail", jobDetail)
	job_log_map := models.CommonMap{}
	job_log_map["JobStartTime"] = time.Now()
	job_log_map["JobBody"] = jobDetail
	result = true
	switch jobDetail.RunType {
		case RunTypeHttp:
			logger.Info("HTTP")
			result, err = ays_run.HttpJob(jobDetail)
			job_log_map["JobResult"] = err.Error()
			//记录err
		case RunTypeCommand:
			logger.Info("Command")
			_, err = ays_run.CommandJob(jobDetail, params.ip, params.port)
			err != nil && result = false
			job_log_map["JobResult"] = err.Error()
	}

	//记录日志
	//执行结束
	job_log_map["JobEndTime"] = time.Now()
	job_log_map["JobTime"] = strconv.FormatInt(job_log_map["JobEndTime"].(time.Time).Unix() - job_log_map["JobStartTime"].(time.Time).Unix(), 10)
	job_log.UpdateJobLog(params.jobLogId, job_log_map)

	//是否弱依赖
	if result == true {
		//弱依赖下发子任务
		/*
		##################   ##################
		###############   # #   ###############
		###############   ###  ################
		################   #  #################
		#################    ##################
		##################  ###################
		#######################################
		 */
		logger.Info("true!!!!")
	}
	return
}