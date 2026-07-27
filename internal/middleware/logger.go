package middleware

import (
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"
	"time"
	"my-bbs/internal/logger"
	"github.com/gin-gonic/gin"
)

// RequestLogger 自定义请求日志中间件：记录方法、路径、状态码、耗时、客户端 IP
// 日志通过 logger 的 channel 异步落盘，不阻塞请求主流程
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		if query != "" {
			path = path + "?" + query
		}

		logger.Info("HTTP status=%d | latency=%v | ip=%s | method=%-7s | path=%s %s",
			status,
			latency,
			clientIP,
			method,
			path,
			errorMessage,
		)
	}
}

// Recovery 自定义 panic 恢复中间件：记录堆栈并返回 500
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 断开的连接不需要当作服务器错误反复刷日志
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						errMsg := strings.ToLower(se.Error())
						if strings.Contains(errMsg, "broken pipe") ||
							strings.Contains(errMsg, "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				if brokenPipe {
					logger.Warn("Recovery broken pipe | error=%v | request=%s", err, string(httpRequest))
					if e, ok := err.(error); ok {
						_ = c.Error(e)
					}
					c.Abort()
					return
				}

				logger.Error("Recovery panic recovered | error=%v\nrequest=%s\nstack=%s",
					err,
					string(httpRequest),
					debug.Stack(),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "服务器内部错误",
				})
			}
		}()
		c.Next()
	}
}
