package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod"
	"github.com/sirupsen/logrus"
	"github.com/xpzouying/headless_browser"
	"github.com/xpzouying/zhipin-mcp/browser"
	"github.com/xpzouying/zhipin-mcp/configs"
	"github.com/xpzouying/zhipin-mcp/cookies"
	"github.com/xpzouying/zhipin-mcp/zhipin"
)

// ZhipinService BOSS直聘业务服务
type ZhipinService struct{}

// NewZhipinService 创建BOSS直聘服务实例
func NewZhipinService() *ZhipinService {
	return &ZhipinService{}
}

// DeleteCookies 删除cookies文件，用于登录重置
func (s *ZhipinService) DeleteCookies(ctx context.Context) error {
	cookiePath := cookies.GetCookiesFilePath()
	cookieLoader := cookies.NewLoadCookie(cookiePath)
	return cookieLoader.DeleteCookies()
}

// CheckLoginStatus 检查登录状态
func (s *ZhipinService) CheckLoginStatus(ctx context.Context) (*LoginStatusResponse, error) {
	b := newBrowser()
	defer b.Close()

	page := b.NewPage()
	defer page.Close()

	loginAction := zhipin.NewLogin(page)
	isLoggedIn, err := loginAction.CheckLoginStatus(ctx)
	if err != nil {
		return nil, err
	}

	return &LoginStatusResponse{
		IsLoggedIn: isLoggedIn,
		Username:   configs.Username,
	}, nil
}

// GetLoginQrcode 获取登录二维码
func (s *ZhipinService) GetLoginQrcode(ctx context.Context) (*LoginQrcodeResponse, error) {
	b := newBrowser()
	page := b.NewPage()

	deferFunc := func() {
		_ = page.Close()
		b.Close()
	}

	loginAction := zhipin.NewLogin(page)

	// 先检查登录状态
	isLoggedIn, err := loginAction.CheckLoginStatus(ctx)
	if err != nil {
		deferFunc()
		return nil, err
	}

	// 如果已登录，直接返回
	if isLoggedIn {
		defer deferFunc()
		return &LoginQrcodeResponse{
			Timeout:    "0s",
			Img:        "",
			IsLoggedIn: true,
		}, nil
	}

	// 未登录，获取二维码（返回 base64）
	img, loggedIn, err := loginAction.FetchQrcodeImageAsBase64(ctx)
	if err != nil || loggedIn {
		defer deferFunc()
	}
	if err != nil {
		return nil, err
	}

	// 保存二维码到文件用于调试
	if err := saveQrcodeImage(img); err != nil {
		logrus.Warnf("保存二维码图片失败: %v", err)
	} else {
		logrus.Info("二维码图片已保存到 qrcode.png")
	}

	timeout := 4 * time.Minute

	if !loggedIn {
		go func() {
			ctxTimeout, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			defer deferFunc()

			if loginAction.WaitForLogin(ctxTimeout) {
				logrus.Info("登录成功")
				if err := saveCookies(page); err != nil {
					logrus.Warnf("保存 cookies 失败: %v", err)
				} else {
					logrus.Info("cookies 已保存")
				}
			}
		}()
	}

	return &LoginQrcodeResponse{
		Timeout: func() string {
			if loggedIn {
				return "0s"
			}
			return timeout.String()
		}(),
		Img:        img,
		IsLoggedIn: loggedIn,
	}, nil
}

// SearchJobs 搜索职位
func (s *ZhipinService) SearchJobs(ctx context.Context, req *SearchJobsRequest) (*SearchJobsResponse, error) {
	b := newBrowser()
	defer b.Close()

	page := b.NewPage()
	defer page.Close()

	// 检查登录状态
	loginAction := zhipin.NewLogin(page)
	isLoggedIn, err := loginAction.CheckLoginStatus(ctx)
	if err != nil {
		return nil, err
	}
	if !isLoggedIn {
		return nil, errLoginRequired
	}

	// 搜索
	searchAction := zhipin.NewSearch(page)
	params := zhipin.SearchParams{
		Keyword:    req.Keyword,
		City:       req.City,
		District:   req.District,
		Experience: req.Experience,
		Education:  req.Education,
		JobType:    req.JobType,
		Salary:     req.Salary,
		Page:       req.Page,
		PageSize:   req.PageSize,
	}

	result, err := searchAction.SearchJobs(ctx, params)
	if err != nil {
		return nil, err
	}

	return &SearchJobsResponse{
		Jobs:     convertJobs(result.Jobs),
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	}, nil
}

// GetJobDetail 获取职位详情
func (s *ZhipinService) GetJobDetail(ctx context.Context, jobID string) (*JobDetailResponse, error) {
	b := newBrowser()
	defer b.Close()

	page := b.NewPage()
	defer page.Close()

	detailAction := zhipin.NewDetail(page)
	job, err := detailAction.GetJobDetail(ctx, jobID)
	if err != nil {
		return nil, err
	}

	return &JobDetailResponse{
		Job: convertJob(job),
	}, nil
}

// DeliverJob 投递简历
func (s *ZhipinService) DeliverJob(ctx context.Context, req *DeliverJobRequest) (*DeliverJobResponse, error) {
	// 检查每日投递上限
	count, err := GetTodayDeliveredCount()
	if err != nil {
		return nil, err
	}
	if count >= configs.MaxDaily {
		return &DeliverJobResponse{
			JobID:   req.JobID,
			Success: false,
			Message: "今日投递已达上限",
		}, nil
	}

	// 检查是否已投递
	isDelivered, err := IsJobDelivered(req.JobID)
	if err != nil {
		logrus.Warnf("检查投递状态失败: %v", err)
	}
	if isDelivered {
		return &DeliverJobResponse{
			JobID:   req.JobID,
			Success: false,
			Message: "该职位已投递过",
		}, nil
	}

	// 投递
	b := newBrowser()
	defer b.Close()

	page := b.NewPage()
	defer page.Close()

	deliverAction := zhipin.NewDeliver(page)
	result, err := deliverAction.DeliverJob(ctx, req.JobID)
	if err != nil {
		return nil, err
	}

	// 保存投递记录
	if result.Success {
		appliedJob := &AppliedJob{
			JobID:     req.JobID,
			JobTitle:  result.Message,
			Status:    "success",
			AppliedAt: time.Now(),
		}
		_ = SaveAppliedJob(appliedJob)
		_ = UpdateDeliveryStats(true)
	} else {
		_ = UpdateDeliveryStats(false)
	}

	return &DeliverJobResponse{
		JobID:   req.JobID,
		Success: result.Success,
		Message: result.Message,
	}, nil
}

// DeliveredList 获取已投递列表
func (s *ZhipinService) DeliveredList(ctx context.Context, limit, offset int) (*DeliveredListResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	jobs, total, err := GetDeliveredJobs(limit, offset)
	if err != nil {
		return nil, err
	}

	return &DeliveredListResponse{
		Jobs:  jobs,
		Total: total,
	}, nil
}

// BatchDeliver 批量投递
func (s *ZhipinService) BatchDeliver(ctx context.Context, jobIDs []string) (*BatchDeliverResponse, error) {
	response := &BatchDeliverResponse{
		Total:   len(jobIDs),
		Results: []DeliverJobResponse{},
	}

	// 获取今日已投递数量
	todayCount, err := GetTodayDeliveredCount()
	if err != nil {
		return nil, err
	}

	b := newBrowser()
	defer b.Close()

	page := b.NewPage()
	defer page.Close()

	deliverAction := zhipin.NewDeliver(page)

	for i, jobID := range jobIDs {
		// 检查每日上限
		if todayCount+response.Success >= configs.MaxDaily {
			response.Results = append(response.Results, DeliverJobResponse{
				JobID:   jobID,
				Success: false,
				Message: "今日投递已达上限",
			})
			continue
		}

		// 检查是否已投递
		isDelivered, _ := IsJobDelivered(jobID)
		if isDelivered {
			response.Results = append(response.Results, DeliverJobResponse{
				JobID:   jobID,
				Success: false,
				Message: "该职位已投递过",
			})
			continue
		}

		// 随机延时
		randomDelay()

		// 投递
		result, err := deliverAction.DeliverJobFromSearchList(jobID)
		if err != nil {
			result = &zhipin.DeliverResult{
				JobID:   jobID,
				Success: false,
				Message: err.Error(),
			}
		}

		response.Results = append(response.Results, DeliverJobResponse{
			JobID:   jobID,
			Success: result.Success,
			Message: result.Message,
		})

		if result.Success {
			response.Success++
			todayCount++

			// 保存投递记录
			appliedJob := &AppliedJob{
				JobID:     jobID,
				JobTitle:  result.Message,
				Status:    "success",
				AppliedAt: time.Now(),
			}
			_ = SaveAppliedJob(appliedJob)
			_ = UpdateDeliveryStats(true)
		} else {
			response.Failed++
			_ = UpdateDeliveryStats(false)
		}

		logrus.Infof("批量投递进度: %d/%d, 成功: %d, 失败: %d", i+1, len(jobIDs), response.Success, response.Failed)
	}

	return response, nil
}

// GetStats 获取投递统计
func (s *ZhipinService) GetStats(ctx context.Context) (*StatsResponse, error) {
	// 今日统计
	todayStats, err := GetTodayStats()
	if err != nil {
		return nil, err
	}

	// 总统计
	total, err := GetTotalStats()
	if err != nil {
		return nil, err
	}

	return &StatsResponse{
		TodayDelivered: todayStats.TotalDelivered,
		TodaySuccess:   todayStats.SuccessCount,
		TodayFailed:    todayStats.FailedCount,
		TotalDelivered: total,
	}, nil
}

// GetConfig 获取配置
func (s *ZhipinService) GetConfig(ctx context.Context) (*ConfigResponse, error) {
	cfg := configs.GetConfig()
	return &ConfigResponse{
		Username:   cfg.Account.Username,
		MaxDaily:   cfg.Delivery.MaxDaily,
		Headless:   cfg.Browser.Headless,
		CronActive: cfg.Cron.Enabled,
	}, nil
}

// UpdateConfig 更新配置
func (s *ZhipinService) UpdateConfig(ctx context.Context, req *UpdateConfigRequest) error {
	// 密码加密存储
	if req.Password != "" {
		encrypted, err := configs.Encrypt(req.Password)
		if err != nil {
			return err
		}
		configs.Password = encrypted
	}

	if req.Username != "" {
		configs.Username = req.Username
	}

	if req.MaxDaily > 0 {
		configs.MaxDaily = req.MaxDaily
	}

	return nil
}

// StartCron 启动定时任务
func (s *ZhipinService) StartCron(ctx context.Context, task *CronTask) error {
	cronMgr := zhipin.GetCronManager()

	// 转换为 CronTaskInfo
	taskInfo := &zhipin.CronTaskInfo{
		ID:       task.ID,
		TaskName: task.TaskName,
		CronExpr: task.CronExpr,
		Keyword:  task.Keyword,
		City:     task.City,
		IsActive: task.IsActive,
	}

	// 设置搜索回调
	cronMgr.SetSearchCallback(func(keyword, city string) error {
		// 创建搜索请求
		req := &SearchJobsRequest{
			Keyword:  keyword,
			City:     city,
			Page:     1,
			PageSize: 10,
		}

		// 搜索职位
		result, err := s.SearchJobs(ctx, req)
		if err != nil {
			return err
		}

		// 自动投递前5个职位
		jobIDs := make([]string, 0, 5)
		for i, job := range result.Jobs {
			if i >= 5 {
				break
			}
			jobIDs = append(jobIDs, job.ID)
		}

		if len(jobIDs) > 0 {
			_, err = s.BatchDeliver(ctx, jobIDs)
			if err != nil {
				return err
			}
		}

		return nil
	})

	// 添加任务
	_, err := cronMgr.AddTask(taskInfo)
	if err != nil {
		return err
	}

	// 保存到数据库
	return SaveCronTask(task)
}

// StopCron 停止定时任务
func (s *ZhipinService) StopCron(ctx context.Context, taskID int) error {
	cronMgr := zhipin.GetCronManager()
	err := cronMgr.RemoveTask(taskID)
	if err != nil {
		return err
	}

	return UpdateCronTask(taskID, false)
}

// 辅助函数

func newBrowser() *headless_browser.Browser {
	return browser.NewBrowser(configs.IsHeadless(), browser.WithBinPath(configs.GetBinPath()))
}

// saveCookies 保存浏览器 cookies 到文件
func saveCookies(page *rod.Page) error {
	cks, err := page.Browser().GetCookies()
	if err != nil {
		return err
	}

	data, err := json.Marshal(cks)
	if err != nil {
		return err
	}

	cookieLoader := cookies.NewLoadCookie(cookies.GetCookiesFilePath())
	return cookieLoader.SaveCookies(data)
}

// saveQrcodeImage 保存二维码图片到文件（用于调试）
func saveQrcodeImage(base64Data string) error {
	// 解析 base64 数据（去除 data:image/png;base64, 前缀）
	prefix := "data:image/png;base64,"
	if len(base64Data) <= len(prefix) {
		return nil
	}

	imgData, err := base64.StdEncoding.DecodeString(base64Data[len(prefix):])
	if err != nil {
		return err
	}

	// 保存到当前目录的 qrcode.png
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	path := filepath.Join(dir, "qrcode.png")
	return os.WriteFile(path, imgData, 0644)
}

func randomDelay() {
	minDelay := configs.MinDelay
	maxDelay := configs.MaxDelay
	if minDelay <= 0 {
		minDelay = 3000
	}
	if maxDelay <= 0 {
		maxDelay = 8000
	}

	delay := minDelay + rand.Intn(maxDelay-minDelay)
	time.Sleep(time.Duration(delay) * time.Millisecond)
}

func convertJobs(jobs []zhipin.Job) []Job {
	result := make([]Job, len(jobs))
	for i, job := range jobs {
		result[i] = convertJob(&job)
	}
	return result
}

func convertJob(job *zhipin.Job) Job {
	return Job{
		ID:          job.ID,
		Title:       job.Title,
		CompanyName: job.CompanyName,
		SalaryRange: job.SalaryRange,
		City:        job.City,
		District:    job.District,
		Experience:  job.Experience,
		Education:   job.Education,
		JobType:     job.JobType,
		CompanySize: job.CompanySize,
		HRName:      job.HRName,
		HRActive:    job.HRActive,
		Description: job.Description,
		Tags:        job.Tags,
		UpdatedAt:   job.UpdatedAt,
	}
}

// 错误定义
var (
	errLoginRequired = &Error{"请先登录"}
)

// Error 自定义错误
type Error struct {
	Msg string
}

func (e *Error) Error() string {
	return e.Msg
}
