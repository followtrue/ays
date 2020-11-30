package models

import (
	"fmt"
	_ "github.com/go-xorm/xorm"
	"github.com/sony/sonyflake"
	"ays/src/modules/logger"
	"time"
)

type JobLog struct {
	Id                uint64	`xorm:"not null pk char(64)"`
	JobType           int       `xorm:"not null default 1 comment('任务执行类型，1http 2rpc 3 mq') TINYINT(2)"`
	JobAlias          string    `xorm:"not null comment('任务alias') CHAR(20)"`
	NodeGroupAlias    string    `xorm:"not null comment('执行器组alias') CHAR(20)"`
	NodeIp            string    `xorm:"not null comment('执行Ip+port') VARCHAR(60)"`
	DispatchStartTime time.Time `xorm:"comment('调度开始时间') DATETIME"`
	DispatchTime      string    `xorm:"comment('调度花费时间') VARCHAR(50)"`
	DispatchResult    string    `xorm:"comment('调度结果') VARCHAR(1000)"`
	JobSendTime       time.Time `xorm:"comment('任务下发时间') DATETIME"`
	JobSendResult     string    `xorm:"comment('下发结果') VARCHAR(1000)"`
	JobStartTime      time.Time `xorm:"comment('任务开始执行时间') DATETIME"`
	JobBody           map[string]string    `xorm:"comment('任务请求体') JSON"`
	JobEndTime        time.Time `xorm:"comment('任务执行结束时间') DATETIME"`
	JobTime           string    `xorm:"comment('任务执行时间') VARCHAR(50)"`
	JobResult         string    `xorm:"comment('任务执行结果') VARCHAR(1000)"`
	JobStatus         int    	`xorm:"not null default 1 comment('任务状态') TINYINT(2)"`
}

func (job_log *JobLog) Find (id uint64) bool {
	res, _ := DB.Where("id = ?", id).Get(job_log)
	return res
}

func (job_log *JobLog) Store () (bool) {

	//新建任务日志
	job_log.Id = createJobLogId()

	_, err := DB.InsertOne(job_log)
	if err != nil {
		//错误
		//logger.Info("insert job_log failed:", fmt.Sprintf("job_log_id: %v", job_log.Id), err)
		logger.Info("aaaa", err)
		return false
	}
	logger.Info("insert job_log success:", fmt.Sprintf("job_log_id: %v", job_log.Id))
	return true
}

func (job_log *JobLog) Update () (int)  {
	if job_log.Id == 0 {
		//记录日志
		logger.Info("update job_log failed: ", fmt.Sprintf("job_log_id: %v", job_log.Id))
		return 0
	}

	update_rows, err := DB.Table(job_log).Where("id = ?", job_log.Id).Update(job_log)
	if err != nil {
		logger.Info("update job_log row 0:", fmt.Sprintf("job_log_id: %v", job_log.Id), err)
		return 0
	}
	return int(update_rows)
}


func (job_log *JobLog) UpdateData(id uint64, data CommonMap) (int64, error) {
	return DB.Table(job_log).Where("id = ?", id).Update(data)
}

func createJobLogId()(jobId uint64){
	rd := time.Now().UnixNano() % int64(100000000)
	flake := sonyflake.NewSonyflake(sonyflake.Settings{})
	jobId, err := flake.NextID()
	logger.Info("flake.NextID", jobId)
	if err != nil {
		logger.Info("JobLog.ID failed", err)
	}
	return jobId + uint64(rd)
}

func (job_log *JobLog) CountJobLog(where map[string]string) (total int64, err error){
	engine := DB.Where("id>?", 0)
	if len(where) > 0 {
		for k, v := range where {
			engine = engine.Where(k + " like ?", "%"+v+"%")
		}
	}
	return engine.Count(job_log)
}

func (job_log *JobLog) GetCountJobLogList(year int, month int, where map[string]string) (list []map[string]string, err error) {
	sql := "select date(dispatch_start_time) as log_date ,count(id) as log_total " +
		" from ays_job_log where year(dispatch_start_time)=? " +
		" and month(dispatch_start_time)=? " //+ string(month)

	if len(where) > 0 {
		for k, v := range where {
			sql = sql + " and " + k + "=" + v
		}
	}
	sql = sql + " group by date(dispatch_start_time)"
	//var list []map[string]string
	err = DB.SQL(sql, year, month).Find(&list)
	return list, err
}

func (job_log *JobLog) GetLogList(condition map[string]string, offset int, limit int) ([]JobLog, error){
	logs := make([]JobLog, 0)
	engine := DB.Where("id > ?", 0)
	if len(condition) > 0 {
		for k, v := range condition {
			engine = engine.Where(k + " like ?", "%"+v+"%")
		}
	}
	err := engine.Desc("dispatch_start_time").Limit(limit, offset).Find(&logs)
	//println(err.Error())
	return logs, err
}