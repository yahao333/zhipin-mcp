package main

import (
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// setupRoutes 设置路由
func setupRoutes(app *AppServer) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// 中间件
	router.Use(gin.Logger())
	router.Use(middlewareRecovery())
	router.Use(middlewareCORS())
	router.Use(middlewareAPILogger())

	// API路由
	api := router.Group("/api")
	{
		// 健康检查
		api.GET("/health", handleHealth)

		// 登录相关
		api.GET("/login/status", app.handleAPICheckLoginStatus)
		api.GET("/login/qrcode", app.handleAPIGetLoginQrcode)
		api.GET("/login/qrcode/browser", app.handleAPIGetLoginQrcodeWithBrowser) // 扫码登录（显示浏览器窗口）
		api.DELETE("/login/cookies", app.handleAPIDeleteCookies)

		// 职位相关
		api.POST("/jobs/search", app.handleAPISearchJobs)
		api.GET("/jobs/:job_id", app.handleAPIGetJobDetail)

		// 投递相关
		api.POST("/deliver", app.handleAPIDeliverJob)
		api.GET("/delivered", app.handleAPIDeliveredList)
		api.POST("/batch/deliver", app.handleAPIBatchDeliver)

		// 统计
		api.GET("/stats", app.handleAPIGetStats)

		// 配置
		api.GET("/config", app.handleAPIGetConfig)
		api.PUT("/config", app.handleAPIUpdateConfig)

		// 定时任务
		api.POST("/cron/start", app.handleAPIStartCron)
		api.POST("/cron/stop", app.handleAPIStopCron)

		// 消息
		api.GET("/messages", app.handleAPIListMessages)
	}

	return router
}

// middlewareAPILogger API日志中间件
func middlewareAPILogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录请求开始
		logrus.Debugf("API调用: %s %s", c.Request.Method, c.Request.URL.Path)

		// 记录请求参数
		if c.Request.Method == "POST" || c.Request.Method == "PUT" {
			if err := c.Request.ParseForm(); err == nil {
				if formData := c.Request.Form.Encode(); formData != "" {
					logrus.Debugf("API参数: %s", formData)
				}
			}
		}

		// 记录查询参数
		if query := c.Request.URL.Query().Encode(); query != "" {
			logrus.Debugf("API查询参数: %s", query)
		}

		c.Next()

		// 记录响应状态
		logrus.Debugf("API响应: %s %s - %d", c.Request.Method, c.Request.URL.Path, c.Writer.Status())
	}
}

// middlewareRecovery 自定义recovery中间件
func middlewareRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logrus.Errorf("Panic recovered: %v\n%s", err, debug.Stack())
				c.JSON(500, gin.H{"error": "Internal server error"})
				c.Abort()
			}
		}()
		c.Next()
	}
}

// middlewareCORS CORS中间件
func middlewareCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
