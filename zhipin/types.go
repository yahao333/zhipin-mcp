package zhipin

import "time"

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
	JobType     string    `json:"job_type"`
	CompanySize string    `json:"company_size"`
	HRName      string    `json:"hr_name"`
	HRActive    string    `json:"hr_active"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SearchResult 搜索结果
type SearchResult struct {
	Jobs     []Job `json:"jobs"`
	Total    int   `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// SearchParams 搜索参数
type SearchParams struct {
	Keyword    string
	City       string
	District   string
	Experience string
	Education  string
	JobType    string
	Salary     string
	Page       int
	PageSize   int
}

// DeliverResult 投递结果
type DeliverResult struct {
	JobID   string `json:"job_id"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// LoginResult 登录结果
type LoginResult struct {
	Success  bool   `json:"success"`
	Username string `json:"username,omitempty"`
	Message  string `json:"message,omitempty"`
}
