package job_log

import (
	"gitlab.keda-digital.com/kedadigital/ays/src/models"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/logger"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/tools"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/register"
	"strconv"
	"time"
)
/*
type JobLogModules struct {
	Id                uint64
	LogType           int       `xorm:"not null default 1 comment('日志类型') TINYINT(2)"`
	JobId             int       `xorm:"not null default 0 comment('任务ID') INT(11)"`
	DispatchStartTime time.Time `xorm:"comment('调度开始时间') DATETIME"`
	DispatchResult    string    `xorm:"comment('调度结果') STRING"`
	JobSendTime       time.Time `xorm:"comment('任务下发时间') DATETIME"`
	JobSendResult     string    `xorm:"comment('下发结果') STRING"`
	JobStartTime      time.Time `xorm:"comment('任务开始执行时间') DATETIME"`
	JobBody           map[string]string    `xorm:"comment('任务请求体') JSON"`
	JobEndTime        time.Time `xorm:"comment('任务执行结束时间') DATETIME"`
	JobTime           string    `xorm:"comment('任务执行时间') VARCHAR(50)"`
	JobResult         string    `xorm:"comment('任务执行结果') STRING"`
}
*/

const (
	DispatchStart int = 1   // 调度开始
	DispatchEnd   int = 2   // 调度结束
	JobSend       int = 3   // 任务下发
	JobStart 	  int = 4   // 任务执行
	JobEnd 		  int = 5   // 任务结束

)

const (
	JobTypeHttp int = 1
	JobTypeRpc  int = 2
	JobTypeMq   int = 3
)

const(
	JOB_LOG_DISPATH int = 1
	JOB_LOG_SUCCESS int = 2
	JOB_LOG_FAILED  int = 3
)

//创建一个任务，新增任务日志
func AddJobLog(job_log models.JobLog) uint64 {

	if (job_log.Store()) {
		return job_log.Id
	} else {
		return 0
	}
}

//更新任务日志
func UpdateJobLog(id uint64, data models.CommonMap) (int64) {
	job_log := models.JobLog{}
	if (job_log.Find(id) != true) {
		return 0
	}
	update_row, _ := job_log.UpdateData(id, data)
	return update_row
}

func GetLogList(condition map[string]string, offset int, limit int) ([]models.JobLog){
	jobLogModel := new(models.JobLog)
	logList, _ := jobLogModel.GetLogList(condition, offset, limit)
	return logList
}

//任务执行过程中不同步骤更新日志记录
func UpdateStep(id uint64, step int, data models.CommonMap) int{
	if (id == 0) {
		return 0
	}

	//查询该条日志
	job_log_model := models.JobLog{}
	if (job_log_model.Find(id) != true) {
		return 0
	}

	//当前步骤
	switch step {
		case DispatchStart:
			//调度开始
			job_log_model.DispatchStartTime = time.Now()
		case JobSend:
			//任务下发--调度花费时间
			job_log_model.JobSendTime = time.Now();
			job_dispatch_start_time := job_log_model.DispatchStartTime.Unix()
			job_send_time := job_log_model.JobSendTime.Unix()
			job_log_model.DispatchTime = strconv.FormatInt(job_send_time - job_dispatch_start_time, 10)
			//调度结果
			job_log_model.DispatchResult = data["DispatchResult"].(string)
		case JobStart:
			//任务执行--任务请求体
			job_log_model.JobStartTime = time.Now();
			job_log_model.JobBody = data["JobBody"].(map[string]string)
			//下发结果
			job_log_model.JobSendResult = data["JobSendResult"].(string)
		case JobEnd:
			//任务结束--任务执行时间
			job_log_model.JobEndTime = time.Now();
			//执行结果
			job_log_model.JobResult = data["JobResult"].(string)
			job_start_time := job_log_model.JobStartTime.Unix()
			job_end_time := job_log_model.JobEndTime.Unix()
			job_log_model.JobTime = strconv.FormatInt(job_end_time - job_start_time, 10)
	}
	logger.Info("switch", job_log_model)
	return job_log_model.Update()
}

func JobLogCount(where map[string]string) int64 {
	job_log_model := new(models.JobLog)
	total, _ := job_log_model.CountJobLog(where)
	return total
}

func JobLogCountList(year int, month int, where map[string]string) (list []string) {
	job_log_model := new(models.JobLog)
	listArr, err := job_log_model.GetCountJobLogList(year, month, where)
	if err != nil {
		logger.Info("JobLogCountList Error", err)
	}

	listMap := map[string]string{}
	for _, v := range listArr {
		listMap[v["log_date"]] = v["log_total"]
	}
	//获取当前日期
	dispatchDateList := tools.GetMonthDays(year, month)
	for _, v := range dispatchDateList {
		if _, ok := listMap[v]; ok {
			list = append(list, listMap[v])
		} else {
			list = append(list, "0")
		}
	}
	return list
}

func JobLogStep(jobLogDetail models.JobLog, step int, jobDetail models.Job, jobParams string, node *register.NodeView) (models.JobLog) {
	//当前步骤
	switch step {
	case DispatchStart:
		//调度开始
		jobLogDetail.JobAlias = jobDetail.JobAlias
		jobLogDetail.DispatchStartTime = time.Now()
		jobLogDetail.JobBody = map[string]string{
			"job_alias":jobDetail.JobAlias,
			"params":jobParams,
		}
		jobLogDetail.NodeGroupAlias = jobDetail.NodeGroupAlias
		jobLogDetail.JobStatus = JOB_LOG_DISPATH

	case DispatchEnd:
		jobLogDetail.DispatchResult = "success"
		dispatchEndTime := time.Now()
		jobLogDetail.DispatchTime = strconv.FormatInt(dispatchEndTime.UnixNano()/ 1e6 - jobLogDetail.DispatchStartTime.UnixNano()/ 1e6, 10)

	case JobSend:
		jobLogDetail.JobSendTime = time.Now()
		jobLogDetail.NodeIp = node.Ip + ":" + strconv.Itoa(node.Port)
		jobLogDetail.JobSendResult = "success"

	case JobStart:
		jobLogDetail.JobStartTime = time.Now()
		if jobDetail.DispatchType == 2 && node != nil {
			jobLogDetail.NodeIp =  node.Ip + ":" + strconv.Itoa(node.Port)
		}

	case JobEnd:
		jobLogDetail.JobEndTime = time.Now()
		jobLogDetail.JobTime = strconv.FormatInt(jobLogDetail.JobEndTime.UnixNano()/ 1e6 - jobLogDetail.JobStartTime.UnixNano()/ 1e6, 10)
		jobLogDetail.JobResult = "success"
		jobLogDetail.JobStatus = JOB_LOG_SUCCESS

	}
	return jobLogDetail
}