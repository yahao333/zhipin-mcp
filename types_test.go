package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xpzouying/zhipin-mcp/zhipin"
)

// TestConvertJob 测试职位类型转换
func TestConvertJob(t *testing.T) {
	now := time.Now()
	zhipinJob := zhipin.Job{
		ID:          "job-001",
		Title:       "高级工程师",
		CompanyName: "字节跳动",
		SalaryRange: "30k-50k",
		City:        "北京",
		District:    "海淀区",
		Experience:  "3-5年",
		Education:   "本科",
		JobType:     "全职",
		CompanySize: "1000人以上",
		HRName:      "张HR",
		HRActive:    "今日活跃",
		Description: "负责后端开发",
		Tags:        []string{"五险一金", "弹性工作"},
		UpdatedAt:   now,
	}

	result := convertJob(&zhipinJob)

	assert.Equal(t, zhipinJob.ID, result.ID)
	assert.Equal(t, zhipinJob.Title, result.Title)
	assert.Equal(t, zhipinJob.CompanyName, result.CompanyName)
	assert.Equal(t, zhipinJob.SalaryRange, result.SalaryRange)
	assert.Equal(t, zhipinJob.City, result.City)
	assert.Equal(t, zhipinJob.District, result.District)
	assert.Equal(t, zhipinJob.Experience, result.Experience)
	assert.Equal(t, zhipinJob.Education, result.Education)
	assert.Equal(t, zhipinJob.JobType, result.JobType)
	assert.Equal(t, zhipinJob.CompanySize, result.CompanySize)
	assert.Equal(t, zhipinJob.HRName, result.HRName)
	assert.Equal(t, zhipinJob.HRActive, result.HRActive)
	assert.Equal(t, zhipinJob.Description, result.Description)
	assert.Equal(t, zhipinJob.Tags, result.Tags)
	assert.Equal(t, zhipinJob.UpdatedAt, result.UpdatedAt)
}

// TestConvertJobs 测试批量职位类型转换
func TestConvertJobs(t *testing.T) {
	zhipinJobs := []zhipin.Job{
		{
			ID:    "job-001",
			Title: "工程师A",
			City:  "北京",
		},
		{
			ID:    "job-002",
			Title: "工程师B",
			City:  "上海",
		},
		{
			ID:    "job-003",
			Title: "工程师C",
			City:  "深圳",
		},
	}

	result := convertJobs(zhipinJobs)

	assert.Len(t, result, 3, "应转换3个职位")
	assert.Equal(t, "工程师A", result[0].Title)
	assert.Equal(t, "工程师B", result[1].Title)
	assert.Equal(t, "工程师C", result[2].Title)
}

// TestConvertJobsEmpty 测试空列表转换
func TestConvertJobsEmpty(t *testing.T) {
	result := convertJobs([]zhipin.Job{})
	assert.Len(t, result, 0, "空列表应返回空切片")
}

// TestAppliedJobFields 测试 AppliedJob 字段
func TestAppliedJobFields(t *testing.T) {
	now := time.Now()
	job := AppliedJob{
		ID:          1,
		JobID:       "job-001",
		JobTitle:    "高级工程师",
		CompanyName: "字节跳动",
		SalaryRange: "30k-50k",
		City:        "北京",
		AppliedAt:   now,
		Status:      "success",
		ErrorMsg:    "",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, 1, job.ID)
	assert.Equal(t, "job-001", job.JobID)
	assert.Equal(t, "高级工程师", job.JobTitle)
	assert.Equal(t, "字节跳动", job.CompanyName)
	assert.Equal(t, "30k-50k", job.SalaryRange)
	assert.Equal(t, "北京", job.City)
	assert.Equal(t, "success", job.Status)
}

// TestDeliveryStatsFields 测试 DeliveryStats 字段
func TestDeliveryStatsFields(t *testing.T) {
	now := time.Now()
	stats := DeliveryStats{
		ID:              1,
		Date:            "2024-01-01",
		TotalDelivered:  10,
		SuccessCount:    8,
		FailedCount:     2,
		LastDeliveredAt: &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	assert.Equal(t, 1, stats.ID)
	assert.Equal(t, "2024-01-01", stats.Date)
	assert.Equal(t, 10, stats.TotalDelivered)
	assert.Equal(t, 8, stats.SuccessCount)
	assert.Equal(t, 2, stats.FailedCount)
}

// TestCronTaskFields 测试 CronTask 字段
func TestCronTaskFields(t *testing.T) {
	now := time.Now()
	task := CronTask{
		ID:        1,
		TaskName:  "每日投递",
		CronExpr:  "0 9 * * *",
		Keyword:   "工程师",
		City:      "北京",
		IsActive:  true,
		LastRunAt: &now,
		NextRunAt: &now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, 1, task.ID)
	assert.Equal(t, "每日投递", task.TaskName)
	assert.Equal(t, "0 9 * * *", task.CronExpr)
	assert.Equal(t, "工程师", task.Keyword)
	assert.Equal(t, "北京", task.City)
	assert.True(t, task.IsActive)
}

// TestSearchJobsRequestFields 测试 SearchJobsRequest 字段
func TestSearchJobsRequestFields(t *testing.T) {
	req := SearchJobsRequest{
		Keyword:    "工程师",
		City:       "北京",
		District:   "海淀区",
		Experience: "3-5年",
		Education:  "本科",
		JobType:    "全职",
		Salary:     "20k-40k",
		Page:       1,
		PageSize:   20,
	}

	assert.Equal(t, "工程师", req.Keyword)
	assert.Equal(t, "北京", req.City)
	assert.Equal(t, "海淀区", req.District)
	assert.Equal(t, "3-5年", req.Experience)
	assert.Equal(t, "本科", req.Education)
	assert.Equal(t, "全职", req.JobType)
	assert.Equal(t, "20k-40k", req.Salary)
	assert.Equal(t, 1, req.Page)
	assert.Equal(t, 20, req.PageSize)
}

// TestSearchJobsResponseFields 测试 SearchJobsResponse 字段
func TestSearchJobsResponseFields(t *testing.T) {
	jobs := []Job{
		{ID: "job-001", Title: "工程师A"},
		{ID: "job-002", Title: "工程师B"},
	}
	resp := SearchJobsResponse{
		Jobs:     jobs,
		Total:    100,
		Page:     1,
		PageSize: 20,
	}

	assert.Len(t, resp.Jobs, 2)
	assert.Equal(t, 100, resp.Total)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 20, resp.PageSize)
}

// TestDeliverJobRequestFields 测试 DeliverJobRequest 字段
func TestDeliverJobRequestFields(t *testing.T) {
	req := DeliverJobRequest{
		JobID: "job-123",
	}

	assert.Equal(t, "job-123", req.JobID)
}

// TestDeliverJobResponseFields 测试 DeliverJobResponse 字段
func TestDeliverJobResponseFields(t *testing.T) {
	resp := DeliverJobResponse{
		JobID:   "job-123",
		Success: true,
		Message: "投递成功",
	}

	assert.Equal(t, "job-123", resp.JobID)
	assert.True(t, resp.Success)
	assert.Equal(t, "投递成功", resp.Message)
}

// TestBatchDeliverRequestFields 测试 BatchDeliverRequest 字段
func TestBatchDeliverRequestFields(t *testing.T) {
	req := BatchDeliverRequest{
		JobIDs: []string{"job-1", "job-2", "job-3"},
	}

	assert.Len(t, req.JobIDs, 3)
	assert.Equal(t, "job-1", req.JobIDs[0])
}

// TestBatchDeliverResponseFields 测试 BatchDeliverResponse 字段
func TestBatchDeliverResponseFields(t *testing.T) {
	resp := BatchDeliverResponse{
		Total:   10,
		Success: 8,
		Failed:  2,
		Results: []DeliverJobResponse{
			{JobID: "job-1", Success: true},
			{JobID: "job-2", Success: false},
		},
	}

	assert.Equal(t, 10, resp.Total)
	assert.Equal(t, 8, resp.Success)
	assert.Equal(t, 2, resp.Failed)
	assert.Len(t, resp.Results, 2)
}

// TestLoginStatusResponseFields 测试 LoginStatusResponse 字段
func TestLoginStatusResponseFields(t *testing.T) {
	resp := LoginStatusResponse{
		IsLoggedIn: true,
		Username:   "testuser",
	}

	assert.True(t, resp.IsLoggedIn)
	assert.Equal(t, "testuser", resp.Username)
}

// TestLoginQrcodeResponseFields 测试 LoginQrcodeResponse 字段
func TestLoginQrcodeResponseFields(t *testing.T) {
	resp := LoginQrcodeResponse{
		Timeout:    "5m0s",
		IsLoggedIn: false,
		Img:        "base64-image-data",
	}

	assert.Equal(t, "5m0s", resp.Timeout)
	assert.False(t, resp.IsLoggedIn)
	assert.Equal(t, "base64-image-data", resp.Img)
}

// TestConfigResponseFields 测试 ConfigResponse 字段
func TestConfigResponseFields(t *testing.T) {
	resp := ConfigResponse{
		Username:   "testuser",
		MaxDaily:   50,
		Headless:   true,
		CronActive: false,
	}

	assert.Equal(t, "testuser", resp.Username)
	assert.Equal(t, 50, resp.MaxDaily)
	assert.True(t, resp.Headless)
	assert.False(t, resp.CronActive)
}

// TestUpdateConfigRequestFields 测试 UpdateConfigRequest 字段
func TestUpdateConfigRequestFields(t *testing.T) {
	req := UpdateConfigRequest{
		Username: "newuser",
		Password: "encrypted-password",
		MaxDaily: 100,
	}

	assert.Equal(t, "newuser", req.Username)
	assert.Equal(t, "encrypted-password", req.Password)
	assert.Equal(t, 100, req.MaxDaily)
}

// TestMCPToolResultFieldsTypes 测试 MCPToolResult 字段
func TestMCPToolResultFieldsTypes(t *testing.T) {
	result := MCPToolResult{
		Content: []MCPContent{
			{Type: "text", Text: "Hello"},
			{Type: "image", MimeType: "image/png", Data: "base64-data"},
		},
		IsError: false,
	}

	assert.Len(t, result.Content, 2)
	assert.Equal(t, "text", result.Content[0].Type)
	assert.Equal(t, "Hello", result.Content[0].Text)
	assert.Equal(t, "image", result.Content[1].Type)
	assert.Equal(t, "image/png", result.Content[1].MimeType)
	assert.False(t, result.IsError)
}

// TestError 测试自定义错误
func TestError(t *testing.T) {
	err := &Error{Msg: "测试错误消息"}
	assert.Equal(t, "测试错误消息", err.Error())
	assert.Equal(t, "测试错误消息", err.Msg)
}
