package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// middleware 日志中间件
func middlewareLogging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method

		logrus.Infof("[%s] %s %s %d %v",
			method,
			path,
			c.ClientIP(),
			status,
			latency,
		)
	}
}
