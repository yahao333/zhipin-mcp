package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// 创建临时目录
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// 打开数据库
	database, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	require.NoError(t, err, "打开数据库失败")

	// 设置连接池参数
	database.SetMaxOpenConns(10)
	database.SetMaxIdleConns(5)

	// 创建表
	err = createTables(database)
	require.NoError(t, err, "创建表失败")

	// 替换全局数据库变量
	oldDB := db
	db = database

	// 返回清理函数
	cleanup := func() {
		db.Close()
		db = oldDB
		os.RemoveAll(tmpDir)
	}

	return database, cleanup
}

// TestCreateTables 测试创建数据库表
func TestCreateTables(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	// 验证 applied_jobs 表存在
	var count int
	err := database.QueryRow("SELECT COUNT(*) FROM applied_jobs").Scan(&count)
	require.NoError(t, err, "applied_jobs 表应该存在")

	// 验证 delivery_stats 表存在
	err = database.QueryRow("SELECT COUNT(*) FROM delivery_stats").Scan(&count)
	require.NoError(t, err, "delivery_stats 表应该存在")

	// 验证 cron_tasks 表存在
	err = database.QueryRow("SELECT COUNT(*) FROM cron_tasks").Scan(&count)
	require.NoError(t, err, "cron_tasks 表应该存在")
}

// TestSaveAppliedJob 测试保存已投递职位
func TestSaveAppliedJob(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	job := &AppliedJob{
		JobID:       "test-job-001",
		JobTitle:    "高级工程师",
		CompanyName: "测试公司",
		SalaryRange: "20k-35k",
		City:        "北京",
		AppliedAt:   time.Now(),
		Status:      "success",
	}

	err := SaveAppliedJob(job)
	require.NoError(t, err, "保存已投递职位失败")

	// 验证保存成功
	isDelivered, err := IsJobDelivered("test-job-001")
	require.NoError(t, err)
	assert.True(t, isDelivered, "职位应该已投递")
}

// TestSaveAppliedJobReplace 测试保存已投递职位（替换）
func TestSaveAppliedJobReplace(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// 第一次保存
	job1 := &AppliedJob{
		JobID:       "test-job-002",
		JobTitle:    "工程师",
		CompanyName: "公司A",
		Status:      "success",
		AppliedAt:   time.Now(),
	}
	err := SaveAppliedJob(job1)
	require.NoError(t, err)

	// 第二次保存（替换）
	job2 := &AppliedJob{
		JobID:       "test-job-002",
		JobTitle:    "高级工程师",
		CompanyName: "公司B",
		Status:      "failed",
		AppliedAt:   time.Now(),
	}
	err = SaveAppliedJob(job2)
	require.NoError(t, err)

	// 验证只有一条记录
	jobs, _, err := GetDeliveredJobs(10, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, len(jobs), "应该只有一条记录")
}

// TestIsJobDelivered 测试检查职位是否已投递
func TestIsJobDelivered(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// 未投递
	isDelivered, err := IsJobDelivered("non-existent-job")
	require.NoError(t, err)
	assert.False(t, isDelivered, "未投递的职位应返回 false")

	// 已投递
	job := &AppliedJob{
		JobID:     "test-job-003",
		JobTitle:  "测试职位",
		Status:    "success",
		AppliedAt: time.Now(),
	}
	err = SaveAppliedJob(job)
	require.NoError(t, err)

	isDelivered, err = IsJobDelivered("test-job-003")
	require.NoError(t, err)
	assert.True(t, isDelivered, "已投递的职位应返回 true")
}

// TestGetDeliveredJobs 测试获取已投递职位列表
func TestGetDeliveredJobs(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// 保存多个职位
	for i := 0; i < 25; i++ {
		job := &AppliedJob{
			JobID:     "test-job-" + string(rune('a'+i)),
			JobTitle:  "职位 " + string(rune('a'+i)),
			Status:    "success",
			AppliedAt: time.Now(),
		}
		err := SaveAppliedJob(job)
		require.NoError(t, err)
	}

	// 测试分页
	jobs, total, err := GetDeliveredJobs(10, 0)
	require.NoError(t, err)
	assert.Equal(t, 25, total, "总数应为25")
	assert.Len(t, jobs, 10, "第一页应有10条")

	// 第二页
	jobs, total, err = GetDeliveredJobs(10, 10)
	require.NoError(t, err)
	assert.Equal(t, 25, total)
	assert.Len(t, jobs, 10, "第二页应有10条")

	// 第三页
	jobs, total, err = GetDeliveredJobs(10, 20)
	require.NoError(t, err)
	assert.Equal(t, 25, total)
	assert.Len(t, jobs, 5, "第三页应有5条")
}

// TestGetDeliveredJobsEmpty 测试空列表
func TestGetDeliveredJobsEmpty(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	jobs, total, err := GetDeliveredJobs(10, 0)
	require.NoError(t, err)
	assert.Equal(t, 0, total, "总数应为0")
	assert.Len(t, jobs, 0, "列表应为空")
}

// TestUpdateDeliveryStats 测试更新投递统计
func TestUpdateDeliveryStats(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// 第一次成功投递
	err := UpdateDeliveryStats(true)
	require.NoError(t, err)

	// 验证统计
	stats, err := GetTodayStats()
	require.NoError(t, err)
	assert.Equal(t, 1, stats.TotalDelivered, "今日投递应为1")
	assert.Equal(t, 1, stats.SuccessCount, "成功数应为1")
	assert.Equal(t, 0, stats.FailedCount, "失败数应为0")

	// 第二次失败投递
	err = UpdateDeliveryStats(false)
	require.NoError(t, err)

	// 再次验证统计
	stats, err = GetTodayStats()
	require.NoError(t, err)
	assert.Equal(t, 2, stats.TotalDelivered, "今日投递应为2")
	assert.Equal(t, 1, stats.SuccessCount, "成功数应为1")
	assert.Equal(t, 1, stats.FailedCount, "失败数应为1")
}

// TestGetTodayStats 测试获取今日统计
func TestGetTodayStats(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// 无数据时
	stats, err := GetTodayStats()
	require.NoError(t, err)
	assert.Equal(t, 0, stats.TotalDelivered, "今日投递应为0")

	// 有数据时
	err = UpdateDeliveryStats(true)
	require.NoError(t, err)

	stats, err = GetTodayStats()
	require.NoError(t, err)
	assert.Equal(t, 1, stats.TotalDelivered, "今日投递应为1")
}

// TestGetTotalStats 测试获取总统计
func TestGetTotalStats(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// 无数据时
	total, err := GetTotalStats()
	require.NoError(t, err)
	assert.Equal(t, 0, total, "总投递应为0")

	// 有数据时
	err = UpdateDeliveryStats(true)
	require.NoError(t, err)

	total, err = GetTotalStats()
	require.NoError(t, err)
	assert.Equal(t, 1, total, "总投递应为1")
}

// TestGetTodayDeliveredCount 测试获取今日已投递数量
func TestGetTodayDeliveredCount(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// 无数据时
	count, err := GetTodayDeliveredCount()
	require.NoError(t, err)
	assert.Equal(t, 0, count, "今日投递应为0")

	// 有数据时
	err = UpdateDeliveryStats(true)
	require.NoError(t, err)

	count, err = GetTodayDeliveredCount()
	require.NoError(t, err)
	assert.Equal(t, 1, count, "今日投递应为1")
}

// TestSaveCronTask 测试保存定时任务
func TestSaveCronTask(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	task := &CronTask{
		TaskName: "测试任务",
		CronExpr: "0 9 * * *",
		Keyword:  "工程师",
		City:     "北京",
		IsActive: true,
	}

	err := SaveCronTask(task)
	require.NoError(t, err)

	// 验证保存成功
	tasks, err := GetCronTasks()
	require.NoError(t, err)
	assert.Len(t, tasks, 1, "应有1个任务")
	assert.Equal(t, "测试任务", tasks[0].TaskName, "任务名称应一致")
}

// TestGetCronTasks 测试获取定时任务列表
func TestGetCronTasks(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// 保存多个任务
	for i := 0; i < 3; i++ {
		task := &CronTask{
			TaskName: "任务" + string(rune('1'+i)),
			CronExpr: "0 9 * * *",
			Keyword:  "关键词",
			City:     "北京",
			IsActive: true,
		}
		err := SaveCronTask(task)
		require.NoError(t, err)
	}

	tasks, err := GetCronTasks()
	require.NoError(t, err)
	assert.Len(t, tasks, 3, "应有3个任务")
}

// TestGetActiveCronTasks 测试获取活跃的定时任务
func TestGetActiveCronTasks(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// 保存活跃任务
	task1 := &CronTask{
		TaskName: "活跃任务",
		CronExpr: "0 9 * * *",
		IsActive: true,
	}
	err := SaveCronTask(task1)
	require.NoError(t, err)

	// 保存非活跃任务
	task2 := &CronTask{
		TaskName: "非活跃任务",
		CronExpr: "0 10 * * *",
		IsActive: false,
	}
	err = SaveCronTask(task2)
	require.NoError(t, err)

	// 获取活跃任务
	tasks, err := GetActiveCronTasks()
	require.NoError(t, err)
	assert.Len(t, tasks, 1, "应有1个活跃任务")
	assert.Equal(t, "活跃任务", tasks[0].TaskName)
}

// TestUpdateCronTask 测试更新定时任务状态
func TestUpdateCronTask(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// 保存任务
	task := &CronTask{
		TaskName: "测试任务",
		CronExpr: "0 9 * * *",
		IsActive: true,
	}
	err := SaveCronTask(task)
	require.NoError(t, err)

	// 获取任务ID
	tasks, err := GetCronTasks()
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	taskID := tasks[0].ID

	// 更新为非活跃
	err = UpdateCronTask(taskID, false)
	require.NoError(t, err)

	// 验证更新成功
	tasks, err = GetCronTasks()
	require.NoError(t, err)
	assert.False(t, tasks[0].IsActive, "任务应变为非活跃")
}

// TestDeleteCronTask 测试删除定时任务
func TestDeleteCronTask(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// 保存任务
	task := &CronTask{
		TaskName: "待删除任务",
		CronExpr: "0 9 * * *",
		IsActive: true,
	}
	err := SaveCronTask(task)
	require.NoError(t, err)

	// 获取任务ID
	tasks, err := GetCronTasks()
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	taskID := tasks[0].ID

	// 删除任务
	err = DeleteCronTask(taskID)
	require.NoError(t, err)

	// 验证删除成功
	tasks, err = GetCronTasks()
	require.NoError(t, err)
	assert.Len(t, tasks, 0, "任务列表应为空")
}

// TestBoolToInt 测试布尔转整数
func TestBoolToInt(t *testing.T) {
	assert.Equal(t, 1, boolToInt(true), "true 应转换为 1")
	assert.Equal(t, 0, boolToInt(false), "false 应转换为 0")
}

// TestSanitizeJobID 测试清理职位ID
func TestSanitizeJobID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "正常ID",
			input:    "job123",
			expected: "job123",
		},
		{
			name:     "带URL参数",
			input:    "job123?from=search",
			expected: "job123",
		},
		{
			name:     "带多个参数",
			input:    "job456?a=1&b=2",
			expected: "job456",
		},
		{
			name:     "带空格",
			input:    "  job789  ",
			expected: "job789",
		},
		{
			name:     "纯空格",
			input:    "   ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeJobID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
