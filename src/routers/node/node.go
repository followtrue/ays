package node

import (
	"github.com/gin-gonic/gin"
	"ays/src/models"
	"ays/src/modules/logger"
	"ays/src/modules/register"
	"ays/src/modules/tools"
	"strconv"
)

func NodeRouter(router *gin.RouterGroup)  {
	nodeRouter := router.Group("/node")
	// 节点列表，某个执行器组下
	nodeRouter.GET("/", func(context *gin.Context) {
		groupAlias := context.Query("node_group_alias")
		nodeList := register.NodeFindByGroupMysql(groupAlias)

		tools.Success(context, nodeList)
	})

	// 添加node
	nodeRouter.POST("/", func(context *gin.Context) {
		ip := context.PostForm("ip")
		port, _ := strconv.Atoi(context.PostForm("port"))
		groupAlias := context.PostForm("node_group_alias")
		err := register.NodeAdd(ip, port, groupAlias)
		logger.IfError(err)

		if err != nil {
			tools.Error(context, err.Error())
		} else {
			tools.Success(context, map[string]string{})
		}
	})

	// 删除node
	nodeRouter.DELETE("/", func(context *gin.Context) {
		ip := context.Query("ip")
		port, err := strconv.Atoi(context.Query("port"))
		logger.IfError(err)
		var node models.Node
		exists, err := node.Find(ip, port)
		logger.IfError(err)
		if exists {
			res := register.NodeDel(&node)
			if !res {
				tools.Error(context, "删除失败")
			}
		}

		tools.Success(context, map[string]string{})
	})
}