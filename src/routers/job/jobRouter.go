package job

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/mikemintang/go-curl"
	"ays/src/models"
	"ays/src/modules/logger"
	"ays/src/modules/queue"
	"ays/src/modules/register"
	"ays/src/modules/tools"
	"ays/src/modules/user"
	"ays/src/modules/user_job"
	"strconv"
	"ays/src/modules/app"

)

type JsonResult struct {
	Code 	int 		`json:"code"`
	Message string 		`json:"message"`
	Data 	DataStruct `json:"data"`
}
type DataStruct struct {
	List []ListStruct `json:"list"`
}

type ListStruct struct {
	UserId 	string `json:"user_id"`
	Name 	string `json:"name"`
	Phone 	string `json:"phone"`
	Email 	string `json:"email"`
}

type JobInfoForm struct {
	NodeGroupAlias    string    `form:"NodeGroupAlias" json:"NodeGroupAlias" binding:"required"`
	JobAlias          string    `form:"JobAlias" json:"JobAlias" binding:"required"`
	JobName           string    `form:"JobName" json:"JobName" binding:"required"`
	JobTag            []string  `form:"JobTag"`
	RouteType         int       `form:"RouteType" json:"RouteType" binding:"required"`
	JobType           int       `form:"JobType" json:"JobType" binding:"required"`
	JobDepend         int       `form:"JobDepend" json:"JobDepend"`
	ChildList         []string  `form:"ChildList" json:"ChildList"`
	DispatchType      int       `form:"DispatchType" json:"DispatchType"`
	RunType           int       `form:"RunType" json:"RunType" binding:"required"`
	HttpRequestType   int       `form:"HttpRequestType" json:"HttpRequestType"`
	HttpRequestUrl    string    `form:"HttpRequestUrl" json:"HttpRequestUrl"`
	HttpRequestHeader map[string]string   `form:"HttpRequestHeader" json:"HttpRequestHeader"`
	HttpRequestBody   map[string]interface{}    `form:"HttpRequestBody" json:"HttpRequestBody"`
	Command           string    `form:"Command" json:"Command"`
	Timing            string    `form:"Timing" json:"Timing"`
	TimeOut           int32      `form:"TimeOut" json:"TimeOut"`
	Delay             int       `form:"Delay" json:"Delay"`
	RepeatTimes       int       `form:"RepeatTimes" json:"RepeatTimes"`
	RepeatDelay       int       `form:"RepeatDelay" json:"RepeatDelay"`
	NoticeId          int       `form:"NoticeId" json:"NoticeId"`
	Remark            string    `form:"Remark" json:"Remark"`
	Status            int       `form:"Status" json:"Status"`
	ManageList        []int     `form:"ManageList" json:"ManageList"`
	Id        		  int     	`form:"Id" json:"Id"`
	ManageName        string    `form:"ManageName" json:"ManageName"`
}

func JobRouter(router *gin.RouterGroup)  {
	jobRouter := router.Group("/job")

	// 任务列表
	jobRouter.GET("/", func(context *gin.Context) {
		prePage, err := strconv.Atoi(context.DefaultQuery("pre_page", "20"))
		logger.IfError(err)
		page, err := strconv.Atoi(context.DefaultQuery("page", "1"))
		logger.IfError(err)
		nodeGroupAlias := context.DefaultQuery("node_group_alias", "")
		jobAlias := context.DefaultQuery("job_alias", "")
		jobName := context.DefaultQuery("job_name", "")

		total, jobList := register.JobViewList(prePage, page, nodeGroupAlias, jobAlias, jobName)
		list := make([]JobInfoForm, 0)
		for _, v := range jobList {
			// 标签
			tag := make([]string, 0)
			json.Unmarshal([]byte(v.JobTag), &tag)

			// 负责人
			userJobModel := new(models.UserJob)
			users, _ := userJobModel.GetUserByAlias(v.JobAlias)
			userIds := make([]int, 0)
			for _, user := range users {
				userIds = append(userIds, user.UserId)
			}

			manegeName := getUsersName(userIds)

			tmp := JobInfoForm {
				NodeGroupAlias    : v.NodeGroupAlias,
				JobAlias          : v.JobAlias,
				JobName           : v.JobName,
				JobTag            :	tag,
				RouteType         : v.RouteType,
				JobType           :	v.JobType,
				JobDepend         :	v.JobDepend,
				ChildList         : v.ChildList,
				DispatchType      : v.DispatchType,
				RunType           : v.RunType,
				HttpRequestType   : v.HttpRequestType,
				HttpRequestUrl    : v.HttpRequestUrl,
				HttpRequestHeader : v.HttpRequestHeader,
				HttpRequestBody   : v.HttpRequestBody,
				Command           : v.Command,
				Timing            : v.Timing,
				TimeOut           : v.TimeOut,
				Delay             : v.Delay,
				RepeatTimes       : v.RepeatTimes,
				RepeatDelay       : v.RepeatDelay,
				NoticeId          : v.NoticeId,
				Remark            : v.Remark,
				Status            : v.Status,
				ManageList        : userIds,
				Id 				  : v.Id,
				ManageName 	      : manegeName,
			}
			list = append(list, tmp)
		}
		tools.Success(context, map[string]interface{}{
			"total_num": total,
			"list": list,
		})
	})

	// 添加node group
	jobRouter.POST("/form", func(context *gin.Context) {
		var form JobInfoForm
		if err := context.ShouldBind(&form); err != nil {
			logger.IfError(err)
			tools.Error(context, err.Error())
			return
		}

		addJob(form, context)
	})
	jobRouter.POST("/json", func(context *gin.Context) {
		var form JobInfoForm
		if err := context.ShouldBindJSON(&form); err != nil {
			logger.IfError(err)
			tools.Error(context, err.Error())
			return
		}

		addJob(form, context)
	})

	// 更新node group
	jobRouter.PUT("/form", func(context *gin.Context) {
		var form JobInfoForm
		if err := context.ShouldBind(&form); err != nil {
			logger.IfError(err)
			tools.Error(context, err.Error())
			return
		}

		updateJob(form, context)
	})
	jobRouter.PUT("/json", func(context *gin.Context) {
		var form JobInfoForm
		if err := context.ShouldBindJSON(&form); err != nil {
			logger.IfError(err)
			tools.Error(context, err.Error())
			return
		}

		updateJob(form, context)
	})

	// 删除node
	jobRouter.DELETE("/", func(context *gin.Context) {
		alias := context.Query("alias")
		deleteJob(alias, context)
	})
}

func addJob(form JobInfoForm, context *gin.Context) {
	jobTag, _ := json.Marshal(form.JobTag)
	userId := user.UserId
	if userId == 0 {
		userId = user.GetUserId(context.GetHeader("Authorization"))
	}
	if !tools.CheckAlias(form.JobAlias) {
		tools.Error(context, "任务代称只允许数字、字母、下划线的组合")
		return
	}
	job := models.Job{
		NodeGroupAlias    : form.NodeGroupAlias    ,
		JobAlias          : form.JobAlias          ,
		JobName           : form.JobName           ,
		JobTag        	  : string(jobTag)         ,
		RouteType         : form.RouteType         ,
		JobType           : form.JobType           ,
		JobDepend         : form.JobDepend         ,
		ChildList         : form.ChildList         ,
		DispatchType      : form.DispatchType      ,
		RunType           : form.RunType           ,
		HttpRequestType   : form.HttpRequestType   ,
		HttpRequestUrl    : form.HttpRequestUrl    ,
		HttpRequestHeader : form.HttpRequestHeader ,
		HttpRequestBody   : form.HttpRequestBody   ,
		Command           : form.Command           ,
		Timing            : form.Timing            ,
		TimeOut           : form.TimeOut           ,
		Delay             : form.Delay             ,
		RepeatTimes       : form.RepeatTimes       ,
		RepeatDelay       : form.RepeatDelay       ,
		NoticeId          : form.NoticeId          ,
		Remark            : form.Remark            ,
		Status            : form.Status            ,
		CreateUser        : userId           	   ,
		UpdateUser        : userId           	   ,
	}

	res, err := register.JobAdd(&job)

	logger.IfError(err)
	if res {
		// 插入负责人关系
		for _, v := range form.ManageList {
			userJob := models.UserJob{
				UserId : v,
				JobAlias:form.JobAlias,
			}
			user_job.AddUserJob(userJob)
		}
		//// 添加定时任务
		//if job.IsTiming() {
		//	_, err = timer.AddTimerJob(job)
		//	logger.IfError(err)
		//}

		// mq下发类型的通知node监听
		if job.IsMq() {
			nodeList := register.NodeFindByGroup(job.NodeGroupAlias)
			queueList := make(queue.QueueList, len(nodeList))
			for _, node := range nodeList {
				queueList[tools.GetQueueName(node.Ip, node.Port)] = queue.Queue{
					Ip: node.Ip,
					Port: node.Port,
				}
			}
			queue.AddJob(queueList, job.JobAlias, job.NodeGroupAlias)
		}

		tools.Success(context, map[string]string{})
	} else {
		if err != nil {
			tools.Error(context, err.Error())
		} else {
			tools.Error(context, "任务添加失败")
		}
	}
}

func updateJob(form JobInfoForm, context *gin.Context)  {
	job := register.JobGetByAlias(form.JobAlias)
	if job == nil {
		tools.Error(context, "任务不存在")
		return
	}
	jobTag, _ := json.Marshal(form.JobTag)
	userId := user.UserId
	if userId == 0 {
		userId = user.GetUserId(context.GetHeader("Authorization"))
	}
	orgIsMq := 0
	if job.IsMq() {
		orgIsMq = 1
	}

	job.NodeGroupAlias    = form.NodeGroupAlias
	job.JobName           = form.JobName
	job.JobTag            = string(jobTag)
	job.RouteType         = form.RouteType
	job.JobType           = form.JobType
	job.JobDepend         = form.JobDepend
	job.ChildList         = form.ChildList
	job.DispatchType      = form.DispatchType
	job.RunType           = form.RunType
	job.HttpRequestType   = form.HttpRequestType
	job.HttpRequestUrl    = form.HttpRequestUrl
	job.HttpRequestHeader = form.HttpRequestHeader
	job.HttpRequestBody   = form.HttpRequestBody
	job.Command           = form.Command
	job.Timing            = form.Timing
	job.TimeOut           = form.TimeOut
	job.Delay             = form.Delay
	job.RepeatTimes       = form.RepeatTimes
	job.RepeatDelay       = form.RepeatDelay
	job.NoticeId          = form.NoticeId
	job.Remark            = form.Remark
	job.Status            = form.Status
	job.UpdateUser        = userId

	res, err := register.JobUpdate(job)

	logger.IfError(err)
	if res {
		// 删除负责人关联关系
		user_job.DelUserJob(models.UserJob{}, job.JobAlias)
		// 插入新的负责人关系
		for _, v := range form.ManageList {
			userJob := models.UserJob{
				UserId : v,
				JobAlias:form.JobAlias,
			}
			user_job.AddUserJob(userJob)
		}

		//// 修改定时任务
		//if job.IsTiming() {
		//	_, err = timer.UpdateTimerJob(*job)
		//	logger.IfError(err)
		//}

		// 原来是队列任务，现在不是，做删除操作
		if orgIsMq == 1 && !job.IsMq() {
			nodeList := register.NodeFindByGroup(job.NodeGroupAlias)
			queueList := make(queue.QueueList, len(nodeList))
			for _, node := range nodeList {
				queueList[tools.GetQueueName(node.Ip, node.Port)] = queue.Queue{
					Ip: node.Ip,
					Port: node.Port,
				}
			}
			queue.DelJob(queueList, job.JobAlias, job.NodeGroupAlias)
		}

		// 原来不是队列任务，更新后是队列任务,mq下发类型的通知node监听
		if orgIsMq == 0 && job.IsMq() {
			nodeList := register.NodeFindByGroup(job.NodeGroupAlias)
			queueList := make(queue.QueueList, len(nodeList))
			for _, node := range nodeList {
				queueList[tools.GetQueueName(node.Ip, node.Port)] = queue.Queue{
					Ip: node.Ip,
					Port: node.Port,
				}
			}
			queue.AddJob(queueList, job.JobAlias, job.NodeGroupAlias)
		}

		tools.Success(context, map[string]string{})
	} else {
		tools.Error(context, "任务更新失败")
	}
}

func deleteJob(alias string, context *gin.Context)  {
	job := register.JobGetByAlias(alias)

	if job == nil {
   		tools.Error(context, "找不到相关任务")
        return 
    }       

	res, err := register.JobDel(*job)

	logger.IfError(err)
	if res {
		// mq下发类型的操作删除队列
		if job.IsMq() {
			nodeList := register.NodeFindByGroup(job.NodeGroupAlias)
			queueList := make(queue.QueueList, len(nodeList))
			for _, node := range nodeList {
				queueList[tools.GetQueueName(node.Ip, node.Port)] = queue.Queue{
					Ip: node.Ip,
					Port: node.Port,
				}
			}
			queue.DelJob(queueList, job.JobAlias, job.NodeGroupAlias)
		}

		//// 删除定时任务
		//if job.IsTiming() {
		//	_, err = timer.DelTimerJob(*job)
		//	logger.IfError(err)
		//}

		tools.Success(context, map[string]string{})
	} else {
		tools.Error(context, "任务删除失败")
	}
}

func getUsersName(userIds []int) string{
	userName := ""
	if len(userIds) <=0 {
		return userName
	}
	req := curl.NewRequest()
	url := app.Config.OAUTH.Host
	url += "/v1/user/system_user"

	req.SetPostData(map[string]interface{}{
		"system_ids" : 11,
	})

	resp, err := req.Send(url, "POST")
	if err != nil {
		return userName
	}
	if !resp.IsOk() {
		return userName
	}
	println(resp.Body)
	var body JsonResult
	err = json.Unmarshal([]byte(resp.Body), &body)
	if err != nil {
		return userName
	}
	list := body.Data.List
	for _, userId := range userIds {
		for _, v := range list {
			tmpUserId , _ := strconv.Atoi(v.UserId)
			if userId == tmpUserId {
				userName += v.Name + " "
			}
		}
	}
	println(userName)

	return userName
}
