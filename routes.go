package main

import (
	"github.com/gin-gonic/gin"
)

// setupRoutes 设置路由
func setupRoutes(app *AppServer) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// 中间件
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middlewareCORS())

	// API路由
	api := router.Group("/api")
	{
		// 健康检查
		api.GET("/health", handleHealth)

		// 登录相关
		api.GET("/login/status", app.handleAPICheckLoginStatus)
		api.GET("/login/qrcode", app.handleAPIGetLoginQrcode)
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
	}

	return router
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
