package node_group

import (
	"github.com/gin-gonic/gin"
	"gitlab.keda-digital.com/kedadigital/ays/src/models"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/logger"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/register"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/tools"
	"strconv"
)

type NodeGroupForm struct {
	NodeGroupAlias string    `form:"node_group_alias" json:"node_group_alias" binding:"required"`
	NodeGroupName  string    `form:"node_group_name" json:"node_group_name" binding:"required"`
	NodeGroupRank  int       `form:"node_group_rank" json:"node_group_rank"`
	NodeRegType    int       `form:"node_reg_type" json:"node_reg_type"`
}

func NodeGroupRouter(router *gin.RouterGroup)  {
	nodeRouter := router.Group("/node_group")

	// node group list
	nodeRouter.GET("/", func(context *gin.Context) {
		prePage, err := strconv.Atoi(context.DefaultQuery("pre_page", "20"))
		logger.IfError(err)
		page, err := strconv.Atoi(context.DefaultQuery("page", "1"))
		logger.IfError(err)
		groupAlias := context.DefaultQuery("node_group_alias", "")
		groupName := context.DefaultQuery("node_group_name", "")

		total, groupList := register.NodeGroupViewListGet(prePage, page, groupAlias, groupName)
		tools.Success(context, map[string]interface{}{
			"total_num": total,
			"list": groupList,
		})
	})

	// 添加node group
	nodeRouter.POST("/form", func(context *gin.Context) {
		var form NodeGroupForm
		if err := context.ShouldBind(&form); err != nil {
			logger.IfError(err)
			tools.Error(context, err.Error())
		}

		addNodeGroup(form, context)
	})
	nodeRouter.POST("/json", func(context *gin.Context) {
		var form NodeGroupForm
		if err := context.ShouldBindJSON(&form); err != nil {
			logger.IfError(err)
			tools.Error(context, err.Error())
		}

		addNodeGroup(form, context)
	})

	// 更新node group
	nodeRouter.PUT("/form", func(context *gin.Context) {
		var form NodeGroupForm
		if err := context.ShouldBind(&form); err != nil {
			logger.IfError(err)
			tools.Error(context, err.Error())
		}

		updateNodeGroup(form, context)
	})
	nodeRouter.PUT("/json", func(context *gin.Context) {
		var form NodeGroupForm
		if err := context.ShouldBindJSON(&form); err != nil {
			logger.IfError(err)
			tools.Error(context, err.Error())
		}

		updateNodeGroup(form, context)
	})

	// 删除node
	nodeRouter.DELETE("/", func(context *gin.Context) {
		alias := context.Query("node_group_alias")
		deleteNodeGroup(alias, context)
	})
}

func addNodeGroup(form NodeGroupForm, context *gin.Context) {
	if !tools.CheckAlias(form.NodeGroupAlias) {
		tools.Error(context, "任务代称只允许数字、字母、下划线的组合")
		return
	}

	res, _ := register.NodeGroupAdd(models.NodeGroup{
		NodeGroupAlias: form.NodeGroupAlias,
		NodeGroupName: form.NodeGroupName,
		NodeGroupRank: form.NodeGroupRank,
		NodeRegType: form.NodeRegType,
	})

	if res {
		tools.Success(context, map[string]string{})
	} else {
		tools.Error(context, "执行器添加失败")
	}
}

func updateNodeGroup(form NodeGroupForm, context *gin.Context)  {
	nodeGroup := register.NodeGroupGetByAlias(form.NodeGroupAlias)
	if nodeGroup == nil {
		tools.Error(context, "未找到相应执行器:"+form.NodeGroupAlias)
	}
	nodeGroup.NodeGroupName = form.NodeGroupName
	nodeGroup.NodeGroupRank = form.NodeGroupRank
	nodeGroup.NodeRegType = form.NodeRegType

	res, _ := register.NodeGroupUpdate(*nodeGroup)

	if res {
		tools.Success(context, map[string]string{})
	} else {
		tools.Error(context, "执行器更新失败")
	}
}

func deleteNodeGroup(alias string, context *gin.Context)  {
	nodeGroup := register.NodeGroupGetByAlias(alias)
	if nodeGroup == nil {
		tools.Error(context, "未找到相应执行器:"+alias)
	}
	res, _ := register.NodeGroupDel(*nodeGroup)

	if res {
		tools.Success(context, map[string]string{})
	} else {
		tools.Error(context, "执行器删除失败")
	}
}
