package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yahao333/zhipin-mcp/cookies"
)

// MCP 工具处理函数

// handleCheckLoginStatus 处理检查登录状态
func (s *AppServer) handleCheckLoginStatus(ctx context.Context) *MCPToolResult {
	logrus.Info("MCP: 检查登录状态")

	status, err := s.zhipinService.CheckLoginStatus(ctx)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "检查登录状态失败: " + err.Error(),
			}},
			IsError: true,
		}
	}

	var resultText string
	if status.IsLoggedIn {
		resultText = fmt.Sprintf("✅ 已登录\n用户名: %s\n\n你可以使用其他功能了。", status.Username)
	} else {
		resultText = fmt.Sprintf("❌ 未登录\n\n请使用 get_login_qrcode 工具获取二维码进行登录。")
	}

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: resultText,
		}},
	}
}

// handleGetLoginQrcode 处理获取登录二维码
func (s *AppServer) handleGetLoginQrcode(ctx context.Context) *MCPToolResult {
	logrus.Info("MCP: 获取登录二维码")

	result, err := s.zhipinService.GetLoginQrcode(ctx)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "获取登录二维码失败: " + err.Error()}},
			IsError: true,
		}
	}

	if result.IsLoggedIn {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "你当前已处于登录状态"}},
		}
	}

	now := time.Now()
	deadline := func() string {
		d, err := time.ParseDuration(result.Timeout)
		if err != nil {
			return now.Format("2006-01-02 15:04:05")
		}
		return now.Add(d).Format("2006-01-02 15:04:05")
	}()

	contents := []MCPContent{
		{Type: "text", Text: "请用BOSS直聘 App 在 " + deadline + " 前扫码登录 👇"},
		{
			Type:     "image",
			MimeType: "image/png",
			Data:     result.Img,
		},
	}
	return &MCPToolResult{Content: contents}
}

// handleDeleteCookies 处理删除cookies
func (s *AppServer) handleDeleteCookies(ctx context.Context) *MCPToolResult {
	logrus.Info("MCP: 删除cookies，重置登录状态")

	err := s.zhipinService.DeleteCookies(ctx)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "删除cookies失败: " + err.Error()}},
			IsError: true,
		}
	}

	cookiePath := cookies.GetCookiesFilePath()
	resultText := fmt.Sprintf("Cookies 已成功删除，登录状态已重置。\n\n删除的文件路径: %s\n\n下次操作时，需要重新登录。", cookiePath)
	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: resultText,
		}},
	}
}

// handleSearchJobs 处理搜索职位
func (s *AppServer) handleSearchJobs(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	logrus.Info("MCP: 搜索职位")

	// 解析参数
	keyword, _ := args["keyword"].(string)
	city, _ := args["city"].(string)
	district, _ := args["district"].(string)
	experience, _ := args["experience"].(string)
	education, _ := args["education"].(string)
	jobType, _ := args["job_type"].(string)
	salary, _ := args["salary"].(string)
	page := 1
	if p, ok := args["page"].(float64); ok {
		page = int(p)
	}

	if keyword == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "请提供搜索关键词"}},
			IsError: true,
		}
	}

	logrus.Infof("MCP: 搜索职位 - keyword=%s, city=%s, page=%d", keyword, city, page)

	req := &SearchJobsRequest{
		Keyword:    keyword,
		City:       city,
		District:   district,
		Experience: experience,
		Education:  education,
		JobType:    jobType,
		Salary:     salary,
		Page:       page,
		PageSize:   10,
	}

	result, err := s.zhipinService.SearchJobs(ctx, req)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "搜索失败: " + err.Error()}},
			IsError: true,
		}
	}

	if len(result.Jobs) == 0 {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "未找到匹配的职位"}},
		}
	}

	// 格式化输出
	text := fmt.Sprintf("找到 %d 个职位（当前第 %d 页）:\n\n", result.Total, result.Page)
	for i, job := range result.Jobs {
		text += fmt.Sprintf("%d. %s\n", i+1, job.Title)
		text += fmt.Sprintf("   公司: %s\n", job.CompanyName)
		text += fmt.Sprintf("   薪资: %s\n", job.SalaryRange)
		text += fmt.Sprintf("   地点: %s\n", job.City)
		text += fmt.Sprintf("   ID: %s\n\n", job.ID)
	}
	text += "使用 deliver_job 工具可以投递简历"

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: text}},
	}
}

// handleGetJobDetail 处理获取职位详情
func (s *AppServer) handleGetJobDetail(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	logrus.Info("MCP: 获取职位详情")

	jobID, _ := args["job_id"].(string)
	if jobID == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "请提供职位ID"}},
			IsError: true,
		}
	}

	detail, err := s.zhipinService.GetJobDetail(ctx, jobID)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "获取详情失败: " + err.Error()}},
			IsError: true,
		}
	}

	job := detail.Job
	text := fmt.Sprintf("【%s】\n\n", job.Title)
	text += fmt.Sprintf("公司: %s\n", job.CompanyName)
	text += fmt.Sprintf("薪资: %s\n", job.SalaryRange)
	text += fmt.Sprintf("地点: %s %s\n", job.City, job.District)
	text += fmt.Sprintf("经验: %s\n", job.Experience)
	text += fmt.Sprintf("学历: %s\n", job.Education)
	text += fmt.Sprintf("类型: %s\n", job.JobType)
	text += fmt.Sprintf("规模: %s\n", job.CompanySize)
	text += fmt.Sprintf("HR: %s (%s)\n\n", job.HRName, job.HRActive)
	text += fmt.Sprintf("职位描述:\n%s\n\n", job.Description)
	text += fmt.Sprintf("标签: %v\n", job.Tags)
	text += "\n使用 deliver_job 工具可以投递简历"

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: text}},
	}
}

// handleDeliverJob 处理投递简历
func (s *AppServer) handleDeliverJob(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	logrus.Info("MCP: 投递简历")

	jobID, _ := args["job_id"].(string)
	if jobID == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "请提供职位ID"}},
			IsError: true,
		}
	}

	logrus.Infof("MCP: 投递职位 - job_id=%s", jobID)

	result, err := s.zhipinService.DeliverJob(ctx, &DeliverJobRequest{JobID: jobID})
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "投递失败: " + err.Error()}},
			IsError: true,
		}
	}

	var text string
	if result.Success {
		text = fmt.Sprintf("✅ 简历投递成功！\n\n职位ID: %s\n\n请耐心等待HR回复。", result.JobID)
	} else {
		text = fmt.Sprintf("❌ 投递失败\n\n职位ID: %s\n原因: %s", result.JobID, result.Message)
	}

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: text}},
	}
}

// handleDeliveredList 处理已投递列表
func (s *AppServer) handleDeliveredList(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	logrus.Info("MCP: 获取已投递列表")

	limit := 20
	offset := 0
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}
	if o, ok := args["offset"].(float64); ok {
		offset = int(o)
	}

	result, err := s.zhipinService.DeliveredList(ctx, limit, offset)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "获取列表失败: " + err.Error()}},
			IsError: true,
		}
	}

	if len(result.Jobs) == 0 {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "暂无已投递记录"}},
		}
	}

	text := fmt.Sprintf("已投递 %d 个职位:\n\n", result.Total)
	for i, job := range result.Jobs {
		text += fmt.Sprintf("%d. %s\n", i+1, job.JobTitle)
		text += fmt.Sprintf("   公司: %s\n", job.CompanyName)
		text += fmt.Sprintf("   薪资: %s\n", job.SalaryRange)
		text += fmt.Sprintf("   地点: %s\n", job.City)
		text += fmt.Sprintf("   时间: %s\n", job.AppliedAt.Format("2006-01-02 15:04"))
		text += fmt.Sprintf("   状态: %s\n\n", job.Status)
	}

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: text}},
	}
}

// handleBatchDeliver 处理批量投递
func (s *AppServer) handleBatchDeliver(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	logrus.Info("MCP: 批量投递")

	jobIDsInterface, ok := args["job_ids"].([]interface{})
	if !ok {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "请提供职位ID列表"}},
			IsError: true,
		}
	}

	jobIDs := make([]string, 0, len(jobIDsInterface))
	for _, id := range jobIDsInterface {
		if idStr, ok := id.(string); ok {
			jobIDs = append(jobIDs, idStr)
		}
	}

	if len(jobIDs) == 0 {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "职位ID列表为空"}},
			IsError: true,
		}
	}

	logrus.Infof("MCP: 批量投递 - 共 %d 个职位", len(jobIDs))

	result, err := s.zhipinService.BatchDeliver(ctx, jobIDs)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "批量投递失败: " + err.Error()}},
			IsError: true,
		}
	}

	text := fmt.Sprintf("批量投递完成！\n\n")
	text += fmt.Sprintf("总计: %d\n", result.Total)
	text += fmt.Sprintf("成功: %d\n", result.Success)
	text += fmt.Sprintf("失败: %d\n\n", result.Failed)

	text += "详情:\n"
	for i, r := range result.Results {
		status := "✅"
		if !r.Success {
			status = "❌"
		}
		text += fmt.Sprintf("%d. %s 职位ID: %s - %s\n", i+1, status, r.JobID, r.Message)
	}

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: text}},
	}
}

// handleStartCron 处理启动定时任务
func (s *AppServer) handleStartCron(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	logrus.Info("MCP: 启动定时任务")

	taskName, _ := args["task_name"].(string)
	cronExpr, _ := args["cron_expression"].(string)
	keyword, _ := args["keyword"].(string)
	city, _ := args["city"].(string)

	if taskName == "" || cronExpr == "" || keyword == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "请提供任务名称、Cron表达式和搜索关键词"}},
			IsError: true,
		}
	}

	task := &CronTask{
		TaskName: taskName,
		CronExpr: cronExpr,
		Keyword:  keyword,
		City:     city,
		IsActive: true,
	}

	err := s.zhipinService.StartCron(ctx, task)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "启动定时任务失败: " + err.Error()}},
			IsError: true,
		}
	}

	text := fmt.Sprintf("✅ 定时任务已启动！\n\n任务名称: %s\n搜索关键词: %s\n城市: %s\nCron表达式: %s", taskName, keyword, city, cronExpr)

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: text}},
	}
}

// handleStopCron 处理停止定时任务
func (s *AppServer) handleStopCron(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	logrus.Info("MCP: 停止定时任务")

	taskIDFloat, ok := args["task_id"].(float64)
	if !ok {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "请提供任务ID"}},
			IsError: true,
		}
	}

	taskID := int(taskIDFloat)
	err := s.zhipinService.StopCron(ctx, taskID)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "停止定时任务失败: " + err.Error()}},
			IsError: true,
		}
	}

	text := fmt.Sprintf("✅ 定时任务已停止！\n\n任务ID: %d", taskID)

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: text}},
	}
}

// handleGetConfig 处理获取配置
func (s *AppServer) handleGetConfig(ctx context.Context) *MCPToolResult {
	logrus.Info("MCP: 获取配置")

	cfg, err := s.zhipinService.GetConfig(ctx)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "获取配置失败: " + err.Error()}},
			IsError: true,
		}
	}

	text := "当前配置:\n\n"
	text += fmt.Sprintf("用户名: %s\n", cfg.Username)
	text += fmt.Sprintf("每日投递上限: %d\n", cfg.MaxDaily)
	text += fmt.Sprintf("无头模式: %v\n", cfg.Headless)
	text += fmt.Sprintf("定时任务: %v\n", cfg.CronActive)

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: text}},
	}
}

// handleUpdateConfig 处理更新配置
func (s *AppServer) handleUpdateConfig(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	logrus.Info("MCP: 更新配置")

	username, _ := args["username"].(string)
	password, _ := args["password"].(string)
	maxDaily := 0
	if m, ok := args["max_daily"].(float64); ok {
		maxDaily = int(m)
	}

	req := &UpdateConfigRequest{
		Username: username,
		Password: password,
		MaxDaily: maxDaily,
	}

	err := s.zhipinService.UpdateConfig(ctx, req)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "更新配置失败: " + err.Error()}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: "✅ 配置已更新"}},
	}
}

// handleGetStats 处理获取统计
func (s *AppServer) handleGetStats(ctx context.Context) *MCPToolResult {
	logrus.Info("MCP: 获取统计")

	stats, err := s.zhipinService.GetStats(ctx)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "获取统计失败: " + err.Error()}},
			IsError: true,
		}
	}

	text := "投递统计:\n\n"
	text += fmt.Sprintf("今日已投递: %d\n", stats.TodayDelivered)
	text += fmt.Sprintf("今日成功: %d\n", stats.TodaySuccess)
	text += fmt.Sprintf("今日失败: %d\n", stats.TodayFailed)
	text += fmt.Sprintf("累计投递: %d\n", stats.TotalDelivered)

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: text}},
	}
}

// 辅助函数：解析JSON
func parseJSON(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}
