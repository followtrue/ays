package register

import (
	"encoding/json"
	"errors"
	"ays/src/models"
	"ays/src/modules/consul"
	"ays/src/modules/logger"
)

const (
	JOB_INFO_PRE = "job_info_"
)

// 新增job
func JobAdd(job *models.Job) (bool, error) {
	exist, err := job.Exists()
	if exist || err != nil {
		return false, errors.New("任务已存在，请更换任务代称")
	}

	if job.JobType == 2 && job.Timing != "" {
		job.Timing = ""
	}
	if job.JobType == 1 && job.Timing == "" {
		return false, errors.New("定时任务，请填写执行频率")
	}

	_, err = job.Create()
	if err != nil {
		return false, err
	}
	job.GetJobByAlias(job.JobAlias)
	return JobAddConsul(*job)
}

// consul新增job
func JobAddConsul(job models.Job) (bool, error) {
	key := generateJobKey(job.JobAlias)
	return consul.KvSetObj(key, job)
}

// 更新job
func JobUpdate(job *models.Job) (bool, error) {
	if job.JobType == 2 && job.Timing != "" {
		job.Timing = ""
	}
	if job.JobType == 1 && job.Timing == "" {
		return false, errors.New("定时任务，请填写执行频率")
	}

	_, err := job.Update()
	if err != nil {
		return false, err
	} else {
		key := generateJobKey(job.JobAlias)
		return consul.KvSetObj(key, job)
	}
}

// job删除
func JobDel(job models.Job) (bool, error) {
	key := generateJobKey(job.JobAlias)
	res, err := consul.KvDel(key)
	logger.IfError(err)
	if !res {
		return res, err
	}

	_, err = job.Delete()
	if err != nil {
		return false, err
	} else {
		return true, nil
	}
}

// 获取job
func JobGetByAlias(alias string) *models.Job {
	var job models.Job
	key := generateJobKey(alias)
	jobJson := consul.KvGet(key)

	if jobJson == "" {
		return nil 
	}

	err := json.Unmarshal([]byte(jobJson), &job)
	logger.IfError(err)
	if err != nil {
		return nil
	}
	return &job
}

// job json 列表
func JobListGet() []string {
	return consul.PreSearch(JOB_INFO_PRE)
}

func JobViewList(pre_page int, page int, nodeGroupAlias, jobAlias, jobName string) (int64, []models.Job) {
	jobViewList := make([]models.Job, 0)
	//total, jobList := JobList(pre_page, page)
	job := models.Job{}
	total, err := job.TotalNum(nodeGroupAlias, jobAlias, jobName)
	logger.IfError(err)
	jobList := job.GetList(pre_page, page, nodeGroupAlias, jobAlias, jobName)

	for _, job := range jobList {
		jobViewList = append(jobViewList, job)
	}

	return total, jobViewList
}

// 生成job在kv的键值
func generateJobKey(alias string) string {
	return JOB_INFO_PRE + alias
}

//job总数
func JobCount() int64{
	job_model := new(models.Job)
	total, _ := job_model.CountJob()
	return total
}
