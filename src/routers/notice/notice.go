package notice

import (
	"github.com/gin-gonic/gin"
)

func NoticeRouter(router *gin.RouterGroup)  {
	nodeRouter := router.Group("/notice")

	// 节点有变更时，consul调用，启动节点监听，节点下线时才退出
	// 因改为定时扫描，接到通知也不在有操作
	nodeRouter.GET("/node_list_change", func(context *gin.Context) {
		//register.WatchNodeList()
	})
}