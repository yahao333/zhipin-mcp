package zhipin

import (
	"sync"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

// CronTaskInfo 定时任务信息
type CronTaskInfo struct {
	ID       int
	TaskName string
	CronExpr string
	Keyword  string
	City     string
	IsActive bool
}

// CronManager 定时任务管理器
type CronManager struct {
	cron           *cron.Cron
	tasks          map[int]cron.EntryID
	mu             sync.RWMutex
	searchCallback func(keyword, city string) error
}

// NewCronManager 创建定时任务管理器
func NewCronManager() *CronManager {
	return &CronManager{
		cron:  cron.New(),
		tasks: make(map[int]cron.EntryID),
	}
}

// Start 启动定时任务管理器
func (m *CronManager) Start() {
	m.cron.Start()
	logrus.Infof("定时任务管理器已启动")
}

// Stop 停止定时任务管理器
func (m *CronManager) Stop() {
	ctx := m.cron.Stop()
	<-ctx.Done()
	logrus.Infof("定时任务管理器已停止")
}

// AddTask 添加定时任务
func (m *CronManager) AddTask(task *CronTaskInfo) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 解析cron表达式
	_, err := cron.ParseStandard(task.CronExpr)
	if err != nil {
		return 0, err
	}

	// 添加任务
	id, err := m.cron.AddFunc(task.CronExpr, func() {
		m.executeTask(task)
	})
	if err != nil {
		return 0, err
	}

	m.tasks[task.ID] = id
	logrus.Infof("添加定时任务成功: %s, 表达式: %s", task.TaskName, task.CronExpr)

	return int(id), nil
}

// RemoveTask 移除定时任务
func (m *CronManager) RemoveTask(taskID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	id, ok := m.tasks[taskID]
	if !ok {
		return nil
	}

	m.cron.Remove(id)
	delete(m.tasks, taskID)
	logrus.Infof("移除定时任务成功: %d", taskID)

	return nil
}

// executeTask 执行定时任务
func (m *CronManager) executeTask(task *CronTaskInfo) {
	logrus.Infof("执行定时任务: %s, 关键词: %s, 城市: %s", task.TaskName, task.Keyword, task.City)

	// 执行搜索和投递
	if m.searchCallback != nil {
		err := m.searchCallback(task.Keyword, task.City)
		if err != nil {
			logrus.Errorf("定时任务执行失败: %v", err)
		}
	}
}

// SetSearchCallback 设置搜索回调
func (m *CronManager) SetSearchCallback(fn func(keyword, city string) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.searchCallback = fn
}

// ListTasks 列出所有任务
func (m *CronManager) ListTasks() []cron.Entry {
	return m.cron.Entries()
}

// DefaultCronManager 默认定时任务管理器
var defaultCronManager *CronManager
var cronOnce sync.Once

// GetCronManager 获取默认定时任务管理器
func GetCronManager() *CronManager {
	cronOnce.Do(func() {
		defaultCronManager = NewCronManager()
	})
	return defaultCronManager
}

// StartCron 启动定时任务
func StartCron() {
	GetCronManager().Start()
}

// StopCron 停止定时任务
func StopCron() {
	GetCronManager().Stop()
}
