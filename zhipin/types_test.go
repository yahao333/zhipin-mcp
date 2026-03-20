package zhipin

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestJobFields 测试 Job 字段
func TestJobFields(t *testing.T) {
	now := time.Now()
	job := Job{
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

	assert.Equal(t, "job-001", job.ID)
	assert.Equal(t, "高级工程师", job.Title)
	assert.Equal(t, "字节跳动", job.CompanyName)
	assert.Equal(t, "30k-50k", job.SalaryRange)
	assert.Equal(t, "北京", job.City)
	assert.Equal(t, "海淀区", job.District)
	assert.Equal(t, "3-5年", job.Experience)
	assert.Equal(t, "本科", job.Education)
	assert.Equal(t, "全职", job.JobType)
	assert.Equal(t, "1000人以上", job.CompanySize)
	assert.Equal(t, "张HR", job.HRName)
	assert.Equal(t, "今日活跃", job.HRActive)
	assert.Equal(t, "负责后端开发", job.Description)
	assert.Equal(t, []string{"五险一金", "弹性工作"}, job.Tags)
	assert.Equal(t, now, job.UpdatedAt)
}

// TestJobEmptyTags 测试空标签
func TestJobEmptyTags(t *testing.T) {
	job := Job{
		ID:    "job-002",
		Title: "初级工程师",
		Tags:  []string{},
	}

	assert.Empty(t, job.Tags)
}

// TestSearchResultFields 测试 SearchResult 字段
func TestSearchResultFields(t *testing.T) {
	jobs := []Job{
		{ID: "job-001", Title: "工程师A"},
		{ID: "job-002", Title: "工程师B"},
	}
	result := SearchResult{
		Jobs:     jobs,
		Total:    100,
		Page:     1,
		PageSize: 20,
	}

	assert.Len(t, result.Jobs, 2)
	assert.Equal(t, 100, result.Total)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 20, result.PageSize)
}

// TestSearchParamsFields 测试 SearchParams 字段
func TestSearchParamsFields(t *testing.T) {
	params := SearchParams{
		Keyword:    "工程师",
		District:   "海淀区",
		Experience: "3-5年",
		Education:  "本科",
		JobType:    "全职",
		Salary:     "20k-40k",
		Page:       1,
		PageSize:   20,
	}

	assert.Equal(t, "工程师", params.Keyword)
	assert.Equal(t, "海淀区", params.District)
	assert.Equal(t, "3-5年", params.Experience)
	assert.Equal(t, "本科", params.Education)
	assert.Equal(t, "全职", params.JobType)
	assert.Equal(t, "20k-40k", params.Salary)
	assert.Equal(t, 1, params.Page)
	assert.Equal(t, 20, params.PageSize)
}

// TestDeliverResultFields 测试 DeliverResult 字段
func TestDeliverResultFields(t *testing.T) {
	tests := []struct {
		name   string
		result DeliverResult
	}{
		{
			name: "成功投递",
			result: DeliverResult{
				JobID:   "job-001",
				Success: true,
				Message: "投递成功",
			},
		},
		{
			name: "失败投递",
			result: DeliverResult{
				JobID:   "job-002",
				Success: false,
				Message: "职位已过期",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "成功投递" {
				assert.True(t, tt.result.Success)
				assert.Equal(t, "投递成功", tt.result.Message)
			} else {
				assert.False(t, tt.result.Success)
				assert.Equal(t, "职位已过期", tt.result.Message)
			}
		})
	}
}

// TestLoginResultFields 测试 LoginResult 字段
func TestLoginResultFields(t *testing.T) {
	tests := []struct {
		name   string
		result LoginResult
	}{
		{
			name: "登录成功",
			result: LoginResult{
				Success:  true,
				Username: "testuser",
				Message:  "登录成功",
			},
		},
		{
			name: "登录失败",
			result: LoginResult{
				Success:  false,
				Username: "",
				Message:  "二维码已过期",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "登录成功" {
				assert.True(t, tt.result.Success)
				assert.Equal(t, "testuser", tt.result.Username)
			} else {
				assert.False(t, tt.result.Success)
				assert.Empty(t, tt.result.Username)
			}
		})
	}
}

// TestJobJSONSerialization 测试 JSON 序列化
func TestJobJSONSerialization(t *testing.T) {
	job := Job{
		ID:          "job-001",
		Title:       "高级工程师",
		CompanyName: "字节跳动",
		SalaryRange: "30k-50k",
		Tags:        []string{"五险一金", "弹性工作"},
	}

	// 转换为 JSON
	jsonStr, err := json.Marshal(job)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonStr), "job-001")
	assert.Contains(t, string(jsonStr), "高级工程师")
	assert.Contains(t, string(jsonStr), "字节跳动")

	// 从 JSON 解析
	var parsedJob Job
	err = json.Unmarshal(jsonStr, &parsedJob)
	assert.NoError(t, err)
	assert.Equal(t, job.ID, parsedJob.ID)
	assert.Equal(t, job.Title, parsedJob.Title)
	assert.Equal(t, job.CompanyName, parsedJob.CompanyName)
	assert.Equal(t, job.Tags, parsedJob.Tags)
}
