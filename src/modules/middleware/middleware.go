package middleware

import (
	"github.com/gin-gonic/gin"
	"gitlab.keda-digital.com/kedadigital/ays/src/modules/user"
	"net/http"
)

func LoginAuth() gin.HandlerFunc{
	return func(context *gin.Context) {
		token := context.GetHeader("Authorization")
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
		context.Next()
	}
}
