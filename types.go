package main

import (
	"time"
)

// Job 职位信息
type Job struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	CompanyName string    `json:"company_name"`
	SalaryRange string    `json:"salary_range"`
	City        string    `json:"city"`
	District    string    `json:"district"`
	Experience  string    `json:"experience"`
	Education   string    `json:"education"`
	JobType     string    `json:"job_type"`     // 全职/兼职/实习
	CompanySize string    `json:"company_size"` // 公司规模
	HRName      string    `json:"hr_name"`      // HR姓名
	HRActive    string    `json:"hr_active"`    // HR活跃度
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AppliedJob 已投递职位
type AppliedJob struct {
	ID          int       `json:"id"`
	JobID       string    `json:"job_id"`
	JobTitle    string    `json:"job_title"`
	CompanyName string    `json:"company_name"`
	SalaryRange string    `json:"salary_range"`
	City        string    `json:"city"`
	AppliedAt   time.Time `json:"applied_at"`
	Status      string    `json:"status"`        // success, failed
	ErrorMsg    string    `json:"error_message"` // 错误信息
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DeliveryStats 投递统计
type DeliveryStats struct {
	ID              int        `json:"id"`
	Date            string     `json:"date"`
	TotalDelivered  int        `json:"total_delivered"`
	SuccessCount    int        `json:"success_count"`
	FailedCount     int        `json:"failed_count"`
	LastDeliveredAt *time.Time `json:"last_delivered_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// CronTask 定时任务
type CronTask struct {
	ID        int        `json:"id"`
	TaskName  string     `json:"task_name"`
	CronExpr  string     `json:"cron_expression"`
	Keyword   string     `json:"keyword"`
	City      string     `json:"city"`
	IsActive  bool       `json:"is_active"`
	LastRunAt *time.Time `json:"last_run_at"`
	NextRunAt *time.Time `json:"next_run_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// SearchJobsRequest 搜索职位请求
type SearchJobsRequest struct {
	Keyword    string `json:"keyword" binding:"required"`
	District   string `json:"district"`   // 区域
	Experience string `json:"experience"` // 经验要求
	Education  string `json:"education"`  // 学历要求
	JobType    string `json:"job_type"`   // 全职/兼职/实习
	Salary     string `json:"salary"`     // 薪资范围
	Page       int    `json:"page"`       // 页码
	PageSize   int    `json:"page_size"`  // 每页数量
}

// SearchJobsResponse 搜索职位响应
type SearchJobsResponse struct {
	Jobs     []Job `json:"jobs"`
	Total    int   `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// JobDetailResponse 职位详情响应
type JobDetailResponse struct {
	Job Job `json:"job"`
}

// DeliverJobRequest 投递职位请求
type DeliverJobRequest struct {
	JobID string `json:"job_id" binding:"required"`
}

// DeliverJobResponse 投递职位响应
type DeliverJobResponse struct {
	JobID   string `json:"job_id"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// BatchDeliverRequest 批量投递请求
type BatchDeliverRequest struct {
	JobIDs []string `json:"job_ids" binding:"required"`
}

// BatchDeliverResponse 批量投递响应
type BatchDeliverResponse struct {
	Total   int                  `json:"total"`
	Success int                  `json:"success"`
	Failed  int                  `json:"failed"`
	Results []DeliverJobResponse `json:"results"`
}

// DeliveredListResponse 已投递列表响应
type DeliveredListResponse struct {
	Jobs  []AppliedJob `json:"jobs"`
	Total int          `json:"total"`
}

// StatsResponse 统计响应
type StatsResponse struct {
	TodayDelivered int `json:"today_delivered"`
	TodaySuccess   int `json:"today_success"`
	TodayFailed    int `json:"today_failed"`
	TotalDelivered int `json:"total_delivered"`
}

// LoginStatusResponse 登录状态响应
type LoginStatusResponse struct {
	IsLoggedIn bool   `json:"is_logged_in"`
	Username   string `json:"username,omitempty"`
}

// LoginQrcodeResponse 登录二维码响应
type LoginQrcodeResponse struct {
	Timeout    string `json:"timeout"`
	IsLoggedIn bool   `json:"is_logged_in"`
	Img        string `json:"img,omitempty"`     // base64 编码的图片 (data:image/png;base64,...)
	Message    string `json:"message,omitempty"` // 附加消息（如登录成功、超时等）
}

// ConfigResponse 配置响应
type ConfigResponse struct {
	Username   string `json:"username"`
	MaxDaily   int    `json:"max_daily"`
	Headless   bool   `json:"headless"`
	CronActive bool   `json:"cron_active"`
}

// UpdateConfigRequest 更新配置请求
type UpdateConfigRequest struct {
	Username string `json:"username"`
	Password string `json:"password"` // AES加密后的密码
	MaxDaily int    `json:"max_daily"`
}

// CronTaskRequest 创建定时任务请求
type CronTaskRequest struct {
	TaskName string `json:"task_name" binding:"required"`
	CronExpr string `json:"cron_expression" binding:"required"`
	Keyword  string `json:"keyword" binding:"required"`
	City     string `json:"city"`
}

// CronTaskResponse 定时任务响应
type CronTaskResponse struct {
	Task   CronTask `json:"task"`
	Result string   `json:"result"`
}

// MCPContent MCP内容
type MCPContent struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Data     string `json:"data,omitempty"`
}

// MCPToolResult MCP工具结果
type MCPToolResult struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"is_error,omitempty"`
}

// MessageStatus 消息状态
type MessageStatus string

const (
	MessageStatusDelivered MessageStatus = "delivered" // 已送达
	MessageStatusRead      MessageStatus = "read"      // 已读
	MessageStatusUnknown   MessageStatus = "unknown"   // 未知
)

// Message 消息
type Message struct {
	PersonName    string        `json:"person_name"`    // 人名称（HR姓名）
	CompanyName   string        `json:"company_name"`   // 公司名称
	JobTitle      string        `json:"job_title"`      // 职位名称
	Avatar        string        `json:"avatar"`         // 头像URL
	MessageDigest string        `json:"message_digest"` // 消息摘要
	Time          time.Time     `json:"time"`           // 时间
	UnreadCount   int           `json:"unread_count"`   // 未读数量
	Status        MessageStatus `json:"status"`         // 状态
}

// MessageListResponse 消息列表响应
type MessageListResponse struct {
	Messages []Message `json:"messages"`
}
