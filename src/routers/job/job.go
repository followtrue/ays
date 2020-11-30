package job

import (
	"github.com/gin-gonic/gin"
	"gitlab.keda-digital.com/kedadigital/ays/src/models"
	jobModules "gitlab.keda-digital.com/kedadigital/ays/src/modules/job"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/logger"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/tools"
	"strings"
)

// JobForm 任务表单
type JobForm struct {
	JobAlias        string `form:"JobAlias" json:"job_alias" binding:"required"` // 任务别名
	Params          string `form:"Params" json:"params"`// 业务参数
}

func Handler(c *gin.Context) {
	jobParams := c.PostForm("params")
	jobAlias := c.PostForm("job_alias")

	// json格式
	isExist := strings.Index(c.GetHeader("Content-Type"), "application/json")
	if isExist >= 0  {
		var jobForm JobForm
		if err := c.ShouldBindJSON(&jobForm); err != nil {
			logger.Error("参数错误", err.Error())
			tools.Error(c, "参数错误")
			return
		}
		jobAlias  = jobForm.JobAlias
		jobParams = jobForm.Params
	}

	// 检查任务
	jobDetail := new(models.Job)
	res, err := jobDetail.GetJobByAlias(jobAlias)

	// 任务不存在
	if err != nil {
		logger.Error("任务不存在", err.Error())
		tools.Error(c, "任务不存在")
		return
	}

	// 任务不存在
	if !res {
		logger.Error("任务不存在", res)
		tools.Error(c, "任务不存在")
		return
	}

	if jobDetail.Status != 1 {
		logger.Error("任务已停用", jobAlias)
		tools.Error(c, "任务已停用")
		return
	}

	// 开go协程异步处理
	go jobModules.Dispatch(jobParams, jobDetail)

	// 处理子任务
	childList := jobDetail.ChildList
	if len(childList) > 0 && jobDetail.JobDepend == 1{
		for _, v := range childList {
			jobDetail = new(models.Job)
			res, err := jobDetail.GetJobByAlias(v)
			if err != nil || !res || jobDetail.Status != 1 {
				logger.Error("子任务不存在或已停止", "[JobAlias:]"+v)
				continue
			}
			go jobModules.Dispatch(jobParams, jobDetail)
		}
	}
	tools.Success(c, []string{})
}
