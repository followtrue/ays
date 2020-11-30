package job_log

import (
	"github.com/gin-gonic/gin"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/job_log"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/tools"
	"strconv"
)

type JobLogForm struct {
	DispatchResult 	  	string
	DispatchStartTime	string
	DispatchTime   		string
	Id					uint64
	JobAlias			string
	JobEndTime			string
	JobResult			string
	JobSendResult		string
	JobSendTime			string
	JobStartTime		string
	JobStatus			int
	JobTime				string
	RunType 			int
	NodeGroupAlias 		string
	NodeIp				string
}


func JobLogRouter(router *gin.RouterGroup) {
	nodeRouter := router.Group("/job_log")

	// log list
	nodeRouter.GET("/", func(context *gin.Context) {
		prePage, _ := strconv.Atoi(context.DefaultQuery("pre_page", "10"))
		page, _ := strconv.Atoi(context.DefaultQuery("page", "1"))

		nodeGroupAlias := context.Query("node_group_alias")
		jobAlias := context.Query("job_alias")

		condition := map[string]string{}
		if nodeGroupAlias != "" {
			condition["node_group_alias"] = nodeGroupAlias
		}

		if jobAlias != "" {
			condition["job_alias"] = jobAlias
		}

		offset 	:= (page - 1) * prePage

		total 	:= job_log.JobLogCount(condition)
		jobList := job_log.GetLogList(condition, offset, prePage)
		var list []JobLogForm
		for _, v := range jobList{
			tmp := JobLogForm {
				DispatchResult 	: v.DispatchResult	,
				DispatchTime 	: v.DispatchTime	,
				Id 				: v.Id				,
				JobAlias 		: v.JobAlias		,
				JobResult 		: v.JobResult		,
				JobSendResult 	: v.JobSendResult	,
				JobStatus 		: v.JobStatus		,
				RunType 		: v.JobType			,
				JobTime 		: v.JobTime			,
				NodeGroupAlias 	: v.NodeGroupAlias	,
				NodeIp 			: v.NodeIp			,
			}
			if v.DispatchStartTime.Unix() > 0 {
				tmp.DispatchStartTime = v.DispatchStartTime.Format("2006-01-02 15:04:05")
			}

			if v.JobEndTime.Unix() > 0 {
				tmp.JobEndTime = v.JobEndTime.Format("2006-01-02 15:04:05")
			}

			if v.JobSendTime.Unix() > 0 {
				tmp.JobSendTime = v.JobSendTime.Format("2006-01-02 15:04:05")
			}

			if v.JobStartTime.Unix() > 0 {
				tmp.JobStartTime = v.JobStartTime.Format("2006-01-02 15:04:05")
			}

			list = append(list, tmp)
		}

		tools.Success(context, map[string]interface{}{
			"total_num": total,
			"list":      list,
		})
	})

}