package user_job

import (
	"ays/src/models"
)

// 创建负责人与任务关联关系
func AddUserJob(user_job models.UserJob) int {
	insertId, err := user_job.Create()
	if err == nil {
		return insertId
	} else {
		return 0
	}
}

// 删除关联
func DelUserJob(user_job models.UserJob, job_alias string) (bool, error) {
	_, err := user_job.Delete(job_alias)
	if err != nil {
		return false, err
	} else {
		return true, nil
	}
}
