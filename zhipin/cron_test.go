package zhipin

import (
	"testing"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
)

// TestCronManager_NewCronManager 测试创建定时任务管理器
func TestCronManager_NewCronManager(t *testing.T) {
	mgr := NewCronManager()

	assert.NotNil(t, mgr)
	assert.NotNil(t, mgr.cron)
	assert.NotNil(t, mgr.tasks)
}

// TestCronManager_AddTask 测试添加定时任务
func TestCronManager_AddTask(t *testing.T) {
	mgr := NewCronManager()

	task := &CronTaskInfo{
		ID:       1,
		TaskName: "test-task",
		CronExpr: "0 0 * * *", // 每天午夜
		Keyword:  "Golang",
		City:     "Beijing",
		IsActive: true,
	}

	id, err := mgr.AddTask(task)
	assert.NoError(t, err)
	assert.Greater(t, id, 0)

	// 清理
	mgr.Stop()
}

// TestCronManager_AddTask_InvalidCron 测试无效的Cron表达式
func TestCronManager_AddTask_InvalidCron(t *testing.T) {
	mgr := NewCronManager()

	task := &CronTaskInfo{
		ID:       2,
		TaskName: "invalid-task",
		CronExpr: "invalid-cron",
	}

	_, err := mgr.AddTask(task)
	assert.Error(t, err)
}

// TestCronManager_RemoveTask 测试移除定时任务
func TestCronManager_RemoveTask(t *testing.T) {
	mgr := NewCronManager()

	task := &CronTaskInfo{
		ID:       3,
		TaskName: "remove-task",
		CronExpr: "0 0 * * *",
		Keyword:  "Python",
		City:     "Shanghai",
	}

	id, err := mgr.AddTask(task)
	assert.NoError(t, err)
	assert.Greater(t, id, 0)

	// 移除任务
	err = mgr.RemoveTask(3)
	assert.NoError(t, err)

	// 再次移除应该不报错
	err = mgr.RemoveTask(3)
	assert.NoError(t, err)

	mgr.Stop()
}

// TestCronManager_RemoveTask_NotExist 测试移除不存在的任务
func TestCronManager_RemoveTask_NotExist(t *testing.T) {
	mgr := NewCronManager()

	// 移除不存在的任务
	err := mgr.RemoveTask(999)
	assert.NoError(t, err)
}

// TestCronManager_SetSearchCallback 测试设置搜索回调
func TestCronManager_SetSearchCallback(t *testing.T) {
	mgr := NewCronManager()

	// 设置回调
	mgr.SetSearchCallback(func(keyword, city string) error {
		return nil
	})

	// 验证回调已设置
	mgr.mu.RLock()
	assert.NotNil(t, mgr.searchCallback)
	mgr.mu.RUnlock()

	// 执行任务触发回调
	task := &CronTaskInfo{
		ID:       4,
		TaskName: "callback-test",
		CronExpr: "0 0 * * *",
		Keyword:  "Go",
		City:     "Beijing",
	}
	_, err := mgr.AddTask(task)
	assert.NoError(t, err)

	// 手动执行任务
	mgr.executeTask(task)

	// 注意：由于 cron 是异步执行的，这里只验证回调设置正确

	mgr.Stop()
}

// TestCronManager_ListTasks 测试列出所有任务
func TestCronManager_ListTasks(t *testing.T) {
	mgr := NewCronManager()

	// 初始状态应该没有任务
	entries := mgr.ListTasks()
	assert.Empty(t, entries)

	// 添加任务
	task := &CronTaskInfo{
		ID:       5,
		TaskName: "list-task",
		CronExpr: "0 0 * * *",
		Keyword:  "Java",
		City:     "Hangzhou",
	}
	_, err := mgr.AddTask(task)
	assert.NoError(t, err)

	// 列出任务
	entries = mgr.ListTasks()
	assert.Len(t, entries, 1)

	mgr.Stop()
}

// TestCronTaskInfoFields 测试 CronTaskInfo 字段
func TestCronTaskInfoFields(t *testing.T) {
	task := CronTaskInfo{
		ID:       100,
		TaskName: "my-task",
		CronExpr: "0 9 * * *",
		Keyword:  "Engineer",
		City:     "Beijing",
		IsActive: true,
	}

	assert.Equal(t, 100, task.ID)
	assert.Equal(t, "my-task", task.TaskName)
	assert.Equal(t, "0 9 * * *", task.CronExpr)
	assert.Equal(t, "Engineer", task.Keyword)
	assert.Equal(t, "Beijing", task.City)
	assert.True(t, task.IsActive)
}

// TestCronManager_ExecuteTaskWithNilCallback 测试执行任务时回调为nil
func TestCronManager_ExecuteTaskWithNilCallback(t *testing.T) {
	mgr := NewCronManager()

	// 不设置回调，执行任务不应该panic
	task := &CronTaskInfo{
		ID:       6,
		TaskName: "nil-callback-task",
		CronExpr: "0 0 * * *",
		Keyword:  "Test",
		City:     "Beijing",
	}

	// 这应该不会panic
	mgr.executeTask(task)
}

// TestGetCronManager 测试获取默认定时任务管理器
func TestGetCronManager(t *testing.T) {
	// 获取两次应该返回同一个实例
	mgr1 := GetCronManager()
	mgr2 := GetCronManager()

	assert.Same(t, mgr1, mgr2)
}

// TestCronManager_StartStop 测试启动和停止定时任务管理器
func TestCronManager_StartStop(t *testing.T) {
	mgr := NewCronManager()

	// 启动
	mgr.Start()

	// 添加任务
	task := &CronTaskInfo{
		ID:       7,
		TaskName: "start-stop-task",
		CronExpr: "0 0 * * *",
		Keyword:  "Test",
		City:     "Beijing",
	}
	_, err := mgr.AddTask(task)
	assert.NoError(t, err)

	// 停止
	mgr.Stop()
}

// TestCronEntry 测试 cron.Entry 类型
func TestCronEntry(t *testing.T) {
	schedule, _ := cron.ParseStandard("0 0 * * *")
	now := time.Now()
	entry := cron.Entry{
		ID:       1,
		Next:     schedule.Next(now),
		Prev:     schedule.Next(now),
		Schedule: schedule,
	}

	assert.Equal(t, cron.EntryID(1), entry.ID)
	assert.NotNil(t, entry.Schedule)
}
