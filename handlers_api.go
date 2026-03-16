package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// API处理函数

func handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleAPICheckLoginStatus 检查登录状态
func (s *AppServer) handleAPICheckLoginStatus(c *gin.Context) {
	status, err := s.zhipinService.CheckLoginStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}

// handleAPIGetLoginQrcode 获取登录二维码
func (s *AppServer) handleAPIGetLoginQrcode(c *gin.Context) {
	result, err := s.zhipinService.GetLoginQrcode(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// handleAPIDeleteCookies 删除cookies
func (s *AppServer) handleAPIDeleteCookies(c *gin.Context) {
	err := s.zhipinService.DeleteCookies(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "cookies deleted"})
}

// handleAPISearchJobs 搜索职位
func (s *AppServer) handleAPISearchJobs(c *gin.Context) {
	var req SearchJobsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logrus.Infof("API: 搜索职位 - keyword=%s, city=%s", req.Keyword, req.City)

	result, err := s.zhipinService.SearchJobs(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// handleAPIGetJobDetail 获取职位详情
func (s *AppServer) handleAPIGetJobDetail(c *gin.Context) {
	jobID := c.Param("job_id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job_id is required"})
		return
	}

	detail, err := s.zhipinService.GetJobDetail(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, detail)
}

// handleAPIDeliverJob 投递简历
func (s *AppServer) handleAPIDeliverJob(c *gin.Context) {
	var req DeliverJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logrus.Infof("API: 投递职位 - job_id=%s", req.JobID)

	result, err := s.zhipinService.DeliverJob(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// handleAPIDeliveredList 已投递列表
func (s *AppServer) handleAPIDeliveredList(c *gin.Context) {
	limit := 20
	offset := 0

	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	result, err := s.zhipinService.DeliveredList(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// handleAPIBatchDeliver 批量投递
func (s *AppServer) handleAPIBatchDeliver(c *gin.Context) {
	var req BatchDeliverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logrus.Infof("API: 批量投递 - 共 %d 个职位", len(req.JobIDs))

	result, err := s.zhipinService.BatchDeliver(c.Request.Context(), req.JobIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// handleAPIGetStats 获取统计
func (s *AppServer) handleAPIGetStats(c *gin.Context) {
	stats, err := s.zhipinService.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// handleAPIGetConfig 获取配置
func (s *AppServer) handleAPIGetConfig(c *gin.Context) {
	cfg, err := s.zhipinService.GetConfig(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

// handleAPIUpdateConfig 更新配置
func (s *AppServer) handleAPIUpdateConfig(c *gin.Context) {
	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := s.zhipinService.UpdateConfig(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "config updated"})
}

// handleAPIStartCron 启动定时任务
func (s *AppServer) handleAPIStartCron(c *gin.Context) {
	var req CronTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task := &CronTask{
		TaskName: req.TaskName,
		CronExpr: req.CronExpr,
		Keyword:  req.Keyword,
		City:     req.City,
		IsActive: true,
	}

	err := s.zhipinService.StartCron(c.Request.Context(), task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "cron started"})
}

// handleAPIStopCron 停止定时任务
func (s *AppServer) handleAPIStopCron(c *gin.Context) {
	var req struct {
		TaskID int `json:"task_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := s.zhipinService.StopCron(c.Request.Context(), req.TaskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "cron stopped"})
}
