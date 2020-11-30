package index

import (
	"github.com/gin-gonic/gin"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/job_log"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/register"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/tools"
	"time"
)

func IndexRouter(router *gin.RouterGroup) {
	indexRouter := router.Group("/index")

	//index
	indexRouter.GET("/", func(context *gin.Context) {
		//任务数量
		jobNum := register.JobCount()
		//fmt.Println(jobNum)
		//调度数量
		dispatchNum := job_log.JobLogCount(map[string]string{})
		//fmt.Println(dispatchNum)
		//执行器数量
		nodeGroupNum := register.NodeGroupCount()
		//fmt.Println(nodeGroupNum)
		//调度趋势
		//当前年、月
		year := int(time.Now().Year())
		month := int(time.Now().Month())
		dispatchDateList := tools.GetMonthDays(year, month)
		//当月成功趋势
		dispatchListSucess := job_log.JobLogCountList(year, month, map[string]string{"job_status" : "2"})
		//当月失败趋势
		dispatchListFailed := job_log.JobLogCountList(year, month, map[string]string{"job_status" : "3"})
		//当月失败趋势
		//调度成功率占比
		dispatchSuccessNum := job_log.JobLogCount(map[string]string{"job_status" : "2"})
		tools.Success(context, map[string]interface{}{
			"job_total": jobNum,
			"dispatch_total": dispatchNum,
			"node_group_total" : nodeGroupNum,
			"dispatch_success_total" : dispatchSuccessNum,
			"dispatch_success_list" : dispatchListSucess,
			"dispatch_dailed_list" : dispatchListFailed,
			"dispatch_date_list" : dispatchDateList,
		})
	})
}