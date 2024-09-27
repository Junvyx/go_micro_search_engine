package handler

import (
	"net/url"

	"github.com/gin-gonic/gin"
)

//中间件的作用，一个可以获取一些信息
//一个可以阻断请求，中断handler链条（认证）

func GetUserInfo(ctx *gin.Context) {
	username, err := url.QueryUnescape(ctx.Request.Header.Get("UserName")) //从request header里获得UserName
	if err == nil {
		ctx.Set("user_name", username)
	}
}
