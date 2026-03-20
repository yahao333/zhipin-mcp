package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
	"github.com/yahao333/zhipin-mcp/configs"
)

var db *sql.DB

// initDatabase 初始化数据库
func initDatabase() error {
	// 确保数据目录存在
	dbPath := configs.DatabasePath
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建数据目录失败: %v", err)
	}

	// 打开数据库
	database, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return fmt.Errorf("打开数据库失败: %v", err)
	}

	// 设置连接池参数
	database.SetMaxOpenConns(10)
	database.SetMaxIdleConns(5)
	database.SetConnMaxLifetime(5 * time.Minute)

	// 创建表
	if err := createTables(database); err != nil {
		return fmt.Errorf("创建表失败: %v", err)
	}

	db = database
	logrus.Infof("数据库初始化成功: %s", dbPath)
	return nil
}

// createTables 创建数据库表
func createTables(database *sql.DB) error {
	// 已投递职位表
	_, err := database.Exec(`
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
	if err != nil {
		return err
	}

	// 创建索引
	_, err = database.Exec(`CREATE INDEX IF NOT EXISTS idx_applied_jobs_job_id ON applied_jobs(job_id)`)
	if err != nil {
		return err
	}
	_, err = database.Exec(`CREATE INDEX IF NOT EXISTS idx_applied_jobs_applied_at ON applied_jobs(applied_at)`)
	if err != nil {
		return err
	}

	// 投递统计表
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
	if err != nil {
		return err
	}
	_, err = database.Exec(`CREATE INDEX IF NOT EXISTS idx_delivery_stats_date ON delivery_stats(date)`)
	if err != nil {
		return err
	}

	// 定时任务表
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
	if err != nil {
		return err
	}
	_, err = database.Exec(`CREATE INDEX IF NOT EXISTS idx_cron_tasks_is_active ON cron_tasks(is_active)`)
	if err != nil {
		return err
	}

	return nil
}

// SaveAppliedJob 保存已投递职位
func SaveAppliedJob(job *AppliedJob) error {
	_, err := db.Exec(`
		INSERT OR REPLACE INTO applied_jobs (job_id, job_title, company_name, salary_range, city, applied_at, status, error_message, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, job.JobID, job.JobTitle, job.CompanyName, job.SalaryRange, job.City, job.AppliedAt, job.Status, job.ErrorMsg, time.Now())
	if err != nil {
		return fmt.Errorf("保存已投递职位失败: %v", err)
	}
	return nil
}

// IsJobDelivered 检查职位是否已投递
func IsJobDelivered(jobID string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM applied_jobs WHERE job_id = ?", jobID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetDeliveredJobs 获取已投递职位列表
func GetDeliveredJobs(limit int, offset int) ([]AppliedJob, int, error) {
	// 获取总数
	var total int
	err := db.QueryRow("SELECT COUNT(*) FROM applied_jobs").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 获取列表
	rows, err := db.Query(`
		SELECT id, job_id, job_title, company_name, salary_range, city, applied_at, status, error_message, created_at, updated_at
		FROM applied_jobs
		ORDER BY applied_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []AppliedJob
	for rows.Next() {
		var job AppliedJob
		err := rows.Scan(&job.ID, &job.JobID, &job.JobTitle, &job.CompanyName, &job.SalaryRange, &job.City, &job.AppliedAt, &job.Status, &job.ErrorMsg, &job.CreatedAt, &job.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		jobs = append(jobs, job)
	}

	return jobs, total, nil
}

// UpdateDeliveryStats 更新投递统计
func UpdateDeliveryStats(success bool) error {
	today := time.Now().Format("2006-01-02")
	now := time.Now()

	// 尝试更新今天的统计
	result, err := db.Exec(`
		UPDATE delivery_stats
		SET total_delivered = total_delivered + 1,
			success_count = success_count + ?,
			failed_count = failed_count + ?,
			last_delivered_at = ?,
			updated_at = ?
		WHERE date = ?
	`, boolToInt(success), boolToInt(!success), now, now, today)
	if err != nil {
		return err
	}

	// 如果没有今天的记录，则插入
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		_, err = db.Exec(`
			INSERT INTO delivery_stats (date, total_delivered, success_count, failed_count, last_delivered_at)
			VALUES (?, 1, ?, ?, ?)
		`, today, boolToInt(success), boolToInt(!success), now)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetTodayStats 获取今日统计
func GetTodayStats() (*DeliveryStats, error) {
	today := time.Now().Format("2006-01-02")
	stats := &DeliveryStats{Date: today}

	err := db.QueryRow(`
		SELECT date, total_delivered, success_count, failed_count, last_delivered_at
		FROM delivery_stats
		WHERE date = ?
	`, today).Scan(&stats.Date, &stats.TotalDelivered, &stats.SuccessCount, &stats.FailedCount, &stats.LastDeliveredAt)

	if err == sql.ErrNoRows {
		return stats, nil
	}
	if err != nil {
		return nil, err
	}
	return stats, nil
}

// GetTotalStats 获取总统计
func GetTotalStats() (int, error) {
	var total int
	err := db.QueryRow("SELECT COALESCE(SUM(total_delivered), 0) FROM delivery_stats").Scan(&total)
	if err != nil {
		return 0, err
	}
	return total, nil
}

// GetTodayDeliveredCount 获取今日已投递数量
func GetTodayDeliveredCount() (int, error) {
	today := time.Now().Format("2006-01-02")
	var count int
	err := db.QueryRow("SELECT COALESCE(total_delivered, 0) FROM delivery_stats WHERE date = ?", today).Scan(&count)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return count, nil
}

// SaveCronTask 保存定时任务
func SaveCronTask(task *CronTask) error {
	_, err := db.Exec(`
		INSERT OR REPLACE INTO cron_tasks (task_name, cron_expression, keyword, city, is_active, last_run_at, next_run_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, task.TaskName, task.CronExpr, task.Keyword, task.City, task.IsActive, task.LastRunAt, task.NextRunAt, time.Now())
	if err != nil {
		return fmt.Errorf("保存定时任务失败: %v", err)
	}
	return nil
}

// GetCronTasks 获取定时任务列表
func GetCronTasks() ([]CronTask, error) {
	rows, err := db.Query(`
		SELECT id, task_name, cron_expression, keyword, city, is_active, last_run_at, next_run_at, created_at, updated_at
		FROM cron_tasks
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []CronTask
	for rows.Next() {
		var task CronTask
		err := rows.Scan(&task.ID, &task.TaskName, &task.CronExpr, &task.Keyword, &task.City, &task.IsActive, &task.LastRunAt, &task.NextRunAt, &task.CreatedAt, &task.UpdatedAt)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// GetActiveCronTasks 获取活跃的定时任务
func GetActiveCronTasks() ([]CronTask, error) {
	rows, err := db.Query(`
		SELECT id, task_name, cron_expression, keyword, city, is_active, last_run_at, next_run_at, created_at, updated_at
		FROM cron_tasks
		WHERE is_active = TRUE
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []CronTask
	for rows.Next() {
		var task CronTask
		err := rows.Scan(&task.ID, &task.TaskName, &task.CronExpr, &task.Keyword, &task.City, &task.IsActive, &task.LastRunAt, &task.NextRunAt, &task.CreatedAt, &task.UpdatedAt)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// UpdateCronTask 更新定时任务状态
func UpdateCronTask(id int, isActive bool) error {
	_, err := db.Exec("UPDATE cron_tasks SET is_active = ?, updated_at = ? WHERE id = ?", isActive, time.Now(), id)
	return err
}

// DeleteCronTask 删除定时任务
func DeleteCronTask(id int) error {
	_, err := db.Exec("DELETE FROM cron_tasks WHERE id = ?", id)
	return err
}

// boolToInt 将bool转换为int
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// closeDatabase 关闭数据库
func closeDatabase() {
	if db != nil {
		db.Close()
	}
}

// GetDatabasePath 获取数据库路径
func GetDatabasePath() string {
	// 如果配置中没有设置，使用默认路径
	if configs.DatabasePath == "" {
		return "./data/zhipin.db"
	}
	return configs.DatabasePath
}

// sanitizeJobID 清理职位ID，去除特殊字符
func sanitizeJobID(jobID string) string {
	// 移除可能的URL参数
	if idx := strings.Index(jobID, "?"); idx != -1 {
		jobID = jobID[:idx]
	}
	return strings.TrimSpace(jobID)
}
