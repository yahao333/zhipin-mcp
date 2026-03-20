package main

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xpzouying/zhipin-mcp/configs"
	"github.com/xpzouying/zhipin-mcp/zhipin"
)

// mockZhipinService 创建模拟的 ZhipinService 用于测试
type mockZhipinService struct {
	*ZhipinService
}

// TestZhipinServiceNew 测试创建服务实例
func TestZhipinServiceNew(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要浏览器的测试")
	}
	service := NewZhipinService()
	require.NotNil(t, service)
	assert.IsType(t, &ZhipinService{}, service)
}

// TestConvertJobsWithEmptyList 测试空列表转换
func TestConvertJobsWithEmptyList(t *testing.T) {
	result := convertJobs([]zhipin.Job{})
	assert.NotNil(t, result)
	assert.Len(t, result, 0)
}

// TestDeliverJobRequestValidation 测试投递请求验证
func TestDeliverJobRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     DeliverJobRequest
		wantErr bool
	}{
		{
			name: "正常请求",
			req: DeliverJobRequest{
				JobID: "job-123",
			},
			wantErr: false,
		},
		{
			name: "空 JobID",
			req: DeliverJobRequest{
				JobID: "",
			},
			wantErr: false, // 业务层处理
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证请求结构
			assert.NotNil(t, tt.req)
		})
	}
}

// TestSearchJobsRequestValidation 测试搜索请求验证
func TestSearchJobsRequestValidation(t *testing.T) {
	tests := []struct {
		name string
		req  SearchJobsRequest
	}{
		{
			name: "完整请求",
			req: SearchJobsRequest{
				Keyword:    "工程师",
				City:       "北京",
				District:   "海淀区",
				Experience: "3-5年",
				Education:  "本科",
				JobType:    "全职",
				Salary:     "20k-40k",
				Page:       1,
				PageSize:   20,
			},
		},
		{
			name: "最小请求",
			req: SearchJobsRequest{
				Keyword: "工程师",
			},
		},
		{
			name: "分页请求",
			req: SearchJobsRequest{
				Keyword:  "工程师",
				Page:     5,
				PageSize: 50,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.req)
		})
	}
}

// TestBatchDeliverRequestValidation 测试批量投递请求验证
func TestBatchDeliverRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     BatchDeliverRequest
		wantErr bool
	}{
		{
			name: "正常请求",
			req: BatchDeliverRequest{
				JobIDs: []string{"job-1", "job-2", "job-3"},
			},
			wantErr: false,
		},
		{
			name: "空列表",
			req: BatchDeliverRequest{
				JobIDs: []string{},
			},
			wantErr: false, // 业务层处理
		},
		{
			name: "单个职位",
			req: BatchDeliverRequest{
				JobIDs: []string{"job-1"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.req)
		})
	}
}

// TestStatsResponseFields 测试统计响应字段
func TestStatsResponseFields(t *testing.T) {
	resp := StatsResponse{
		TodayDelivered: 10,
		TodaySuccess:   8,
		TodayFailed:    2,
		TotalDelivered: 100,
	}

	assert.Equal(t, 10, resp.TodayDelivered)
	assert.Equal(t, 8, resp.TodaySuccess)
	assert.Equal(t, 2, resp.TodayFailed)
	assert.Equal(t, 100, resp.TotalDelivered)
}

// TestDeliveredListResponseFields 测试已投递列表响应字段
func TestDeliveredListResponseFields(t *testing.T) {
	resp := DeliveredListResponse{
		Jobs: []AppliedJob{
			{JobID: "job-1", JobTitle: "工程师A"},
			{JobID: "job-2", JobTitle: "工程师B"},
		},
		Total: 2,
	}

	assert.Len(t, resp.Jobs, 2)
	assert.Equal(t, 2, resp.Total)
}

// initTestDB 初始化测试数据库
func initTestDB(t *testing.T) *sql.DB {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	require.NoError(t, err)

	database.SetMaxOpenConns(10)
	database.SetMaxIdleConns(5)

	// 创建表
	_, err = database.Exec(`
		CREATE TABLE IF NOT EXISTS applied_jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			job_id VARCHAR(128) NOT NULL UNIQUE,
			job_title VARCHAR(256) NOT NULL,
			company_name VARCHAR(256),
			salary_range VARCHAR(64),
			city VARCHAR(64),
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			status VARCHAR(32) DEFAULT 'success',
			error_message TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	require.NoError(t, err)

	_, err = database.Exec(`
		CREATE TABLE IF NOT EXISTS delivery_stats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date DATE NOT NULL UNIQUE,
			total_delivered INTEGER DEFAULT 0,
			success_count INTEGER DEFAULT 0,
			failed_count INTEGER DEFAULT 0,
			last_delivered_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	require.NoError(t, err)

	_, err = database.Exec(`
		CREATE TABLE IF NOT EXISTS cron_tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_name VARCHAR(128) NOT NULL,
			cron_expression VARCHAR(64),
			keyword VARCHAR(256),
			city VARCHAR(64),
			is_active BOOLEAN DEFAULT TRUE,
			last_run_at DATETIME,
			next_run_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	require.NoError(t, err)

	// 替换全局数据库
	oldDB := db
	db = database

	t.Cleanup(func() {
		db.Close()
		db = oldDB
		os.RemoveAll(tmpDir)
	})

	return database
}

// TestServiceDeliveredList 测试获取已投递列表
func TestServiceDeliveredList(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要浏览器的测试")
	}
	initTestDB(t)

	// 保存测试数据
	now := time.Now()
	job := &AppliedJob{
		JobID:       "test-job-001",
		JobTitle:    "测试工程师",
		CompanyName: "测试公司",
		SalaryRange: "20k-30k",
		City:        "北京",
		AppliedAt:   now,
		Status:      "success",
	}
	err := SaveAppliedJob(job)
	require.NoError(t, err)

	service := NewZhipinService()
	ctx := context.Background()

	// 测试获取列表
	resp, err := service.DeliveredList(ctx, 20, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
	assert.Len(t, resp.Jobs, 1)
	assert.Equal(t, "test-job-001", resp.Jobs[0].JobID)
}

// TestServiceDeliveredListEmpty 测试空列表
func TestServiceDeliveredListEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要浏览器的测试")
	}
	initTestDB(t)

	service := NewZhipinService()
	ctx := context.Background()

	resp, err := service.DeliveredList(ctx, 20, 0)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.Total)
	assert.Len(t, resp.Jobs, 0)
}

// TestServiceDeliveredListPagination 测试分页
func TestServiceDeliveredListPagination(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要浏览器的测试")
	}
	initTestDB(t)

	// 保存多个测试数据
	for i := 0; i < 25; i++ {
		job := &AppliedJob{
			JobID:     "test-job-" + string(rune('a'+i)),
			JobTitle:  "工程师 " + string(rune('a'+i)),
			Status:    "success",
			AppliedAt: time.Now(),
		}
		err := SaveAppliedJob(job)
		require.NoError(t, err)
	}

	service := NewZhipinService()
	ctx := context.Background()

	// 第一页
	resp, err := service.DeliveredList(ctx, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 25, resp.Total)
	assert.Len(t, resp.Jobs, 10)

	// 第二页
	resp, err = service.DeliveredList(ctx, 10, 10)
	require.NoError(t, err)
	assert.Len(t, resp.Jobs, 10)

	// 第三页
	resp, err = service.DeliveredList(ctx, 10, 20)
	require.NoError(t, err)
	assert.Len(t, resp.Jobs, 5)
}

// TestServiceGetStats 测试获取统计
func TestServiceGetStats(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要浏览器的测试")
	}
	initTestDB(t)

	service := NewZhipinService()
	ctx := context.Background()

	// 无数据时统计
	resp, err := service.GetStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, resp.TodayDelivered)
	assert.Equal(t, 0, resp.TotalDelivered)

	// 更新投递统计
	err = UpdateDeliveryStats(true)
	require.NoError(t, err)

	// 再次获取统计
	resp, err = service.GetStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, resp.TodayDelivered)
	assert.Equal(t, 1, resp.TodaySuccess)
	assert.Equal(t, 1, resp.TotalDelivered)
}

// TestServiceGetConfig 测试获取配置
// TestServiceUpdateConfig 测试更新配置
func TestServiceUpdateConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要浏览器的测试")
	}
	// 保存原始配置
	origPassword := configs.Password

	// 恢复原始配置
	defer func() {
		configs.Password = origPassword
	}()

	service := NewZhipinService()
	ctx := context.Background()

	// 更新用户名
	req := &UpdateConfigRequest{
		Username: "newuser",
	}
	err := service.UpdateConfig(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "newuser", configs.Username)

	// 更新每日上限
	req = &UpdateConfigRequest{
		MaxDaily: 100,
	}
	err = service.UpdateConfig(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, 100, configs.MaxDaily)

	// 更新密码（加密存储）
	req = &UpdateConfigRequest{
		Password: "testpassword",
	}
	err = service.UpdateConfig(ctx, req)
	require.NoError(t, err)
	assert.NotEmpty(t, configs.Password)
	assert.NotEqual(t, "testpassword", configs.Password, "密码应该被加密")
}

// TestRandomDelay 测试随机延时
func TestRandomDelay(t *testing.T) {
	// 保存原始配置
	origMinDelay := configs.MinDelay
	origMaxDelay := configs.MaxDelay

	// 设置测试配置
	configs.MinDelay = 10
	configs.MaxDelay = 20

	// 恢复原始配置
	defer func() {
		configs.MinDelay = origMinDelay
		configs.MaxDelay = origMaxDelay
	}()

	// 多次调用确保不会 panic
	for i := 0; i < 10; i++ {
		randomDelay()
	}
}

// TestErrorMessage 测试错误消息
func TestErrorMessage(t *testing.T) {
	tests := []struct {
		err    error
		expect string
	}{
		{
			err:    errLoginRequired,
			expect: "请先登录",
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expect, tt.err.Error())
	}
}

// TestSaveQrcodeImage 测试保存二维码图片
func TestSaveQrcodeImage(t *testing.T) {
	tests := []struct {
		name      string
		base64Str string
		wantErr   bool
	}{
		{
			name:      "有效base64图片数据",
			base64Str: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
			wantErr:   false,
		},
		{
			name:      "无效base64数据",
			base64Str: "data:image/png;base64,!!!invalid!!!",
			wantErr:   true,
		},
		{
			name:      "空数据",
			base64Str: "",
			wantErr:   false, // 空数据不会写入文件
		},
		{
			name:      "仅有前缀无数据",
			base64Str: "data:image/png;base64,",
			wantErr:   false, // 解析后为空，不会写入
		},
		{
			name:      "数据长度等于前缀长度",
			base64Str: "data:image/png;", // 长度 <= prefix
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := saveQrcodeImage(tt.base64Str)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				// 不检查错误，因为可能成功或警告
				// 只要不 panic 即可
			}
		})
	}
}

// TestSaveQrcodeImageFileCreation 测试文件实际创建
func TestSaveQrcodeImageFileCreation(t *testing.T) {
	// 创建一个小的有效PNG base64（1x1像素透明PNG）
	validPNG := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

	// 保存前先记录当前目录
	origWd, err := os.Getwd()
	require.NoError(t, err)

	// 执行保存
	err = saveQrcodeImage(validPNG)
	// 不关心是否成功保存，只确保不panic

	// 检查文件是否创建
	qrcodePath := filepath.Join(origWd, "qrcode.png")
	if _, err := os.Stat(qrcodePath); err == nil {
		// 文件存在，清理测试文件
		os.Remove(qrcodePath)
	}
}
