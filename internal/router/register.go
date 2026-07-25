package router

import "github.com/gin-gonic/gin"

// RouteRegister 路由注册接口
// 每个模块实现这个接口，自己注册自己的路由
type RouteRegister interface {
    Register(r *gin.RouterGroup)
}