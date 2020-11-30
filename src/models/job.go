package models

import (
	"time"
)

const (
	STATUS_RUNING = 1
	STATUS_STOP = 0
)

type Job struct {
	Id                int       `xorm:"not null pk autoincr INT(11)"`
	NodeGroupAlias    string    `xorm:"not null comment('节点组alias') CHAR(20)"`
	JobAlias          string    `xorm:"not null comment('任务alias') CHAR(20)"`
	JobName           string    `xorm:"not null comment('任务名称') VARCHAR(255)"`
	JobTag            string    `xorm:"not null comment('任务标签') VARCHAR(50)"`
	RouteType         int       `xorm:"not null default 1 comment('路由策略') TINYINT(4)"`
	JobType           int       `xorm:"not null default 1 comment('任务类型：1.定时任务，2非定时任务') TINYINT(1)"`
	JobDepend         int       `xorm:"not null comment('依赖关系：1弱依赖，2强依赖') TINYINT(1)"`
	ChildList         []string  `xorm:"comment('子任务ID列表') JSON"`
	DispatchType      int       `xorm:"not null comment('调度方式：1.local，2.rpc，3.mq') TINYINT(1)"`
	RunType           int       `xorm:"not null default 1 comment('执行方式：1.http，2.command') TINYINT(1)"`
	HttpRequestType   int       `xorm:"not null default 1 comment('http请求方式：1get，2post，3put，4delete，5patch') TINYINT(1)"`
	HttpRequestUrl    string    `xorm:"not null default '' comment('http请求地址') VARCHAR(255)"`
	HttpRequestHeader map[string]string   `xorm:"comment('http请求头（非定时任务无需填写）') JSON"`
	HttpRequestBody   map[string]interface{}    `xorm:"comment('http请求体（非定时任务无需填写）') JSON"`
	Command           string    `xorm:"not null default '' comment('command命令') VARCHAR(255)"`
	Timing            string    `xorm:"not null default '' comment('定时时间') VARCHAR(50)"`
	TimeOut           int32      `xorm:"not null default 0 comment('超时时间（秒）') INT(11)"`
	Delay             int       `xorm:"not null default 0 comment('延时时间（秒）') INT(11)"`
	RepeatTimes       int       `xorm:"not null comment('重试次数') INT(11)"`
	RepeatDelay       int       `xorm:"not null comment('重试间隔（秒）') INT(11)"`
	NoticeId          int       `xorm:"not null comment('通知方式列表') INT(11)"`
	Remark            string    `xorm:"not null default '' comment('备注') VARCHAR(255)"`
	CreateUser        int       `xorm:"not null default 0 comment('创建者ID') INT(11)"`
	UpdateUser        int       `xorm:"not null default 0 comment('更新者ID') INT(11)"`
	CreatedAt         time.Time `xorm:"created DATETIME"`
	UpdatedAt         time.Time `xorm:"updated DATETIME"`
	Status            int       `xorm:"comment('状态') TINYINT(4)"`
}

type MqBody struct {
	JobDetail *Job
	Params string
	JobLogId uint64
}

// 根据alias判断是否存在
func (job *Job) Exists() (isExists bool, err error) {
	return DB.Table(job).Where("job_alias = ?", job.JobAlias).Exist()
}

// 新增
func (job *Job) Create() (insertId int, err error) {
	_, err = DB.Insert(job)
	if err == nil {
		insertId = job.Id
	}

	return insertId, err
}

// 更新所有指定字段，未填的强制设置为空
func (job *Job) UpdateBean(id int) (int64, error) {
	return DB.ID(id).Cols("node_group_id, job_alias, job_name").Update(job)
}

// 更新有值字段
func (job *Job) UpdateMap(id int, data CommonMap) (int64, error) {
	return DB.Table(job).ID(id).Update(data)
}

// 更新
func (job *Job) Update() (int64, error) {
	return DB.Id(job.Id).
		Cols("node_group_alias,job_alias,job_name,job_tag,route_type,job_type,job_depend,child_list,dispatch_type,run_type,http_request_type,http_request_url,http_request_header,http_request_body,command,timing,time_out,delay,repeat_times,repeat_delay,notice_id,remark,create_user,update_user,updated_at,status").
		Update(job)
}

// 删除
func (job *Job) Delete() (int64, error) {
	return DB.Id(job.Id).Delete(new(Job))
}

// 获取job任务
func (job *Job) GetJobByAlias(jobAlias string) (bool, error) {
	return DB.Where("job_alias=?", jobAlias).Get(job)
}

// 是否是定时任务
func (job *Job) IsTiming() bool {
	if job.Timing == "" {
		return false
	} else {
		return true
	}
}

// 检测任务是否是队列下发方式
func (job *Job) IsMq() bool {
	if job.DispatchType == 3 {
		return true
	} else {
		return false
	}
}

// 查询总数量
func (job *Job) TotalNum(nodeGroupAlias, jobAlias, jobName string) (int64, error) {
	engine := DB.Table(job).AllCols()
	if nodeGroupAlias != "" {
		//j.NodeGroupAlias = nodeGroupAlias
		engine = engine.Where("node_group_alias like ?", "%"+nodeGroupAlias+"%")
	}

	if jobAlias != "" {
		//j.JobAlias = jobAlias
		engine = engine.Where("job_alias like ?", "%"+jobAlias+"%")
	}

	if jobName != "" {
		//j.JobName = jobName
		engine = engine.Where("job_name like ?", "%"+jobName+"%")
	}

	return engine.Count()
}

// 获取job列表
func (job *Job) GetList(pre_page int, page int, nodeGroupAlias, jobAlias, jobName string) []Job {
	jobs := make([]Job, 0)
	engine := DB.AllCols()

	if nodeGroupAlias != "" {
		engine = engine.Where("node_group_alias like ?", "%"+nodeGroupAlias+"%")
	}

	if jobAlias != "" {
		engine = engine.Where("job_alias like ?", "%"+jobAlias+"%")
	}

	if jobName != "" {
		engine = engine.Where("job_name like ?", "%"+jobName+"%")
	}

	startIndex := pre_page * (page - 1)
	engine.OrderBy("id DESC").Limit(pre_page, startIndex).Find(&jobs)

	return jobs
}

func (job *Job) CountJob() (total int64, err error){
	return DB.Count(job)
}
