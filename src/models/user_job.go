package models

import (
	"time"
)

type UserJob struct {
	Id                int       `xorm:"not null pk autoincr INT(10)"`
	UserId    		  int    	`xorm:"not null comment('用户Id') INT(10)"`
	JobAlias          string    `xorm:"not null comment('任务alias') CHAR(50)"`
	CreatedAt         time.Time `xorm:"created DATETIME"`
	UpdatedAt         time.Time `xorm:"updated DATETIME"`
}


// 根据alias判断是否存在
func (userJob *UserJob) Exists() (isExists bool, err error) {
	return DB.Table(userJob).Where("job_alias = ?", userJob.JobAlias).Exist()
}

// 新增
func (userJob *UserJob) Create() (insertId int, err error) {
	_, err = DB.Insert(userJob)
	if err == nil {
		insertId = userJob.Id
	}

	return insertId, err
}

// 更新所有指定字段，未填的强制设置为空
func (userJob *UserJob) UpdateBean(id int) (int64, error) {
	return DB.ID(id).Cols("node_group_id, job_alias, job_name").Update(userJob)
}

// 更新有值字段
func (userJob *UserJob) UpdateMap(id int, data CommonMap) (int64, error) {
	return DB.Table(userJob).ID(id).Update(data)
}

// 更新
func (userJob *UserJob) Update() (int64, error) {
	return DB.Where("id = ?", userJob.Id).Update(userJob)
}

// 删除
func (userJob *UserJob) Delete(job_alias string) (int64, error) {
	return DB.Where("job_alias=?", job_alias).Delete(new(UserJob))
}

// 获取job任务
func (userJob UserJob) GetUserByAlias(jobAlias string) ([]UserJob, error) {
	userJobs := make([]UserJob, 0)
	engine := DB.Where("job_alias=?", jobAlias)
	err := engine.Find(&userJobs)
	return userJobs, err
}

