package routers

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/app"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/logger"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/tools"
	"gitlab.keda-digital.com/kedadigital/ays/src/routers/index"
	"gitlab.keda-digital.com/kedadigital/ays/src/routers/job"
	"gitlab.keda-digital.com/kedadigital/ays/src/routers/job_log"
	"gitlab.keda-digital.com/kedadigital/ays/src/routers/node"
	"gitlab.keda-digital.com/kedadigital/ays/src/routers/node_group"
	"gitlab.keda-digital.com/kedadigital/ays/src/routers/notice"
	"io"
	"net/http"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/middleware"
	"os"
	"time"
)

var (
	tcpServer *http.Server
)

// 开启tcp端口监听
func Serve(file *os.File) {
	// 重定向日志
	accessFile, _ := os.OpenFile(app.Config.ACCESS_LOG, os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0755)
	gin.DefaultWriter = io.MultiWriter(accessFile, os.Stdout)
	gin.DefaultErrorWriter = io.MultiWriter(accessFile, os.Stdout)
	router := gin.Default()
	// v1 版本
	v1 := router.Group("/ays/v1") 
	{
		// 任务
		jobRouter := v1.Group("/job")
		{
			// 调度接口
			jobRouter.POST("/dispatch", job.Handler)
		}
		notice.NoticeRouter(v1)

		v1.Use(middleware.LoginAuth())
		node.NodeRouter(v1)
		node_group.NodeGroupRouter(v1)
		index.IndexRouter(v1)
		job.JobRouter(v1)
		job_log.JobLogRouter(v1)
	}

	router.GET("/pid", func(i *gin.Context) {
		tools.Success(i, os.Getpid())
	})

	router.GET("/sleep", func(i *gin.Context) {
		time.Sleep(time.Duration(20)*time.Second)
		tools.Success(i, os.Getpid())
	})

	runServe(router, file)
}

// 对sockFD进行监听，实现tcp监听
func runServe(engine *gin.Engine, file *os.File) {
	logger.Error(fmt.Sprintf("------sock fd file: %v----------", file))

	// 监听sockfd
	listener, err := tools.GetFileListener(file)
	if err != nil {
		logger.IfError(err)
		return
	}
	logger.Error(fmt.Sprintf("------manager listener: %v----------", listener))

	// 启动http服务
	tcpServer = &http.Server{Handler: engine}
	err = tcpServer.Serve(listener)
	logger.IfError(err)
}

// 手动停止tcp服务
func StopServe(ctx context.Context) error {
	if tcpServer == nil {
		logger.Error("------manager tcp server is nil----------")
	}

	return tcpServer.Shutdown(ctx)
}

/*func LoginAuth() gin.HandlerFunc{
	return func(context *gin.Context) {
		token := context.GetHeader("Authorization")
		println(token)

		userId := user.GetUserId(token)
		if userId == 0 {
			var data [] string
			code := 4000
			context.JSON(http.StatusOK, gin.H{
				"code": code,
				"message": "请登录",
				"data": data,
			})
			context.Abort()
		}
		println(userId)
		context.Next()
	}
}*/
