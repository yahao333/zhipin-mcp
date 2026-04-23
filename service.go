package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/xpzouying/headless_browser"
	"github.com/yahao333/zhipin-mcp/browser"
	"github.com/yahao333/zhipin-mcp/configs"
	"github.com/yahao333/zhipin-mcp/cookies"
	"github.com/yahao333/zhipin-mcp/pkg/delay"
	"github.com/yahao333/zhipin-mcp/zhipin"
)

// ZhipinService BOSS直聘业务服务
type ZhipinService struct {
	browserFactory *defaultBrowserFactory
	database       *defaultDatabase
}

// BrowserFactory 浏览器工厂接口
type BrowserFactory interface {
	NewBrowser(headless bool, binPath string) *headless_browser.Browser
}

// Database 数据库接口
type Database interface {
	SaveAppliedJob(job *AppliedJob) error
	IsJobDelivered(jobID string) (bool, error)
	GetDeliveredJobs(limit, offset int) ([]AppliedJob, int, error)
	UpdateDeliveryStats(success bool) error
	GetTodayStats() (*DeliveryStats, error)
	GetTotalStats() (int, error)
	GetTodayDeliveredCount() (int, error)
	SaveCronTask(task *CronTask) error
	GetCronTasks() ([]CronTask, error)
	GetActiveCronTasks() ([]CronTask, error)
	UpdateCronTask(id int, isActive bool) error
	DeleteCronTask(id int) error
}

// defaultBrowserFactory 默认浏览器工厂
type defaultBrowserFactory struct{}

func (defaultBrowserFactory) NewBrowser(headless bool, binPath string) *headless_browser.Browser {
	return browser.NewBrowser(headless, browser.WithBinPath(binPath))
}

// defaultDatabase 默认数据库实现
type defaultDatabase struct{}

func (defaultDatabase) SaveAppliedJob(job *AppliedJob) error       { return SaveAppliedJob(job) }
func (defaultDatabase) IsJobDelivered(jobID string) (bool, error)  { return IsJobDelivered(jobID) }
func (defaultDatabase) GetDeliveredJobs(limit, offset int) ([]AppliedJob, int, error) {
	return GetDeliveredJobs(limit, offset)
}
func (defaultDatabase) UpdateDeliveryStats(success bool) error            { return UpdateDeliveryStats(success) }
func (defaultDatabase) GetTodayStats() (*DeliveryStats, error)              { return GetTodayStats() }
func (defaultDatabase) GetTotalStats() (int, error)                         { return GetTotalStats() }
func (defaultDatabase) GetTodayDeliveredCount() (int, error)                 { return GetTodayDeliveredCount() }
func (defaultDatabase) SaveCronTask(task *CronTask) error                   { return SaveCronTask(task) }
func (defaultDatabase) GetCronTasks() ([]CronTask, error)                    { return GetCronTasks() }
func (defaultDatabase) GetActiveCronTasks() ([]CronTask, error)              { return GetActiveCronTasks() }
func (defaultDatabase) UpdateCronTask(id int, isActive bool) error          { return UpdateCronTask(id, isActive) }
func (defaultDatabase) DeleteCronTask(id int) error                        { return DeleteCronTask(id) }

// NewZhipinService 创建BOSS直聘服务实例（使用默认依赖）
func NewZhipinService() *ZhipinService {
	return &ZhipinService{
		browserFactory: &defaultBrowserFactory{},
		database:       &defaultDatabase{},
	}
}

// NewZhipinServiceWithConfig 创建服务实例（可注入自定义依赖）
func NewZhipinServiceWithConfig(factory BrowserFactory, db Database) *ZhipinService {
	return &ZhipinService{
		browserFactory: &defaultBrowserFactory{},
		database:       &defaultDatabase{},
	}
}

// DeleteCookies 删除cookies文件，用于登录重置
func (s *ZhipinService) DeleteCookies(ctx context.Context) error {
	cookiePath := cookies.GetCookiesFilePath()
	cookieLoader := cookies.NewLoadCookie(cookiePath)
	return cookieLoader.DeleteCookies()
}

// CheckLoginStatus 检查登录状态
func (s *ZhipinService) CheckLoginStatus(ctx context.Context) (*LoginStatusResponse, error) {
	factory := s.browserFactory
	if factory == nil {
		factory = &defaultBrowserFactory{}
	}
	browser := factory.NewBrowser(configs.GetEffectiveHeadless(), configs.GetBinPath())
	defer browser.Close()

	page := browser.NewPage()
	defer page.Close()

	loginAction := zhipin.NewLogin(page)
	isLoggedIn, err := loginAction.CheckLoginStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("ZhipinService.CheckLoginStatus: %w", err)
	}

	return &LoginStatusResponse{
		IsLoggedIn: isLoggedIn,
		Username:   configs.Username,
	}, nil
}

// GetLoginQrcode 获取登录二维码
func (s *ZhipinService) GetLoginQrcode(ctx context.Context) (*LoginQrcodeResponse, error) {
	browser := s.browserFactory.NewBrowser(configs.GetEffectiveHeadless(), configs.GetBinPath())
	page := browser.NewPage()

	deferFunc := func() {
		_ = page.Close()
		browser.Close()
	}

	loginAction := zhipin.NewLogin(page)

	// 先检查登录状态
	isLoggedIn, err := loginAction.CheckLoginStatus(ctx)
	if err != nil {
		deferFunc()
		return nil, fmt.Errorf("ZhipinService.CheckLoginStatus: %w", err)
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
		return nil, fmt.Errorf("ZhipinService.GetLoginQrcode: %w", err)
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

// GetLoginQrcodeWithBrowser 获取登录二维码（非 headless 模式，显示浏览器窗口）
// 用于 headless 模式下扫码登录，会临时切换到非 headless 模式
func (s *ZhipinService) GetLoginQrcodeWithBrowser(ctx context.Context) (*LoginQrcodeResponse, error) {
	// 保存原始 headless 设置
	originalHeadless := configs.IsHeadless()

	// 临时设置为非 headless 模式
	logrus.Info("临时切换到非 headless 模式以显示二维码")
	configs.SetHeadless(false)

	// 确保最后恢复原始设置
	defer func() {
		configs.ResetHeadlessOverride()
		logrus.Infof("恢复 headless 模式: %v", originalHeadless)
	}()

	// 创建非 headless 浏览器
	browser := browser.NewBrowser(false, browser.WithBinPath(configs.GetBinPath()))
	defer browser.Close()

	page := browser.NewPage()
	defer page.Close()

	loginAction := zhipin.NewLogin(page)

	// 先检查登录状态
	isLoggedIn, err := loginAction.CheckLoginStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("ZhipinService.GetLoginQrcodeWithBrowser: %w", err)
	}

	// 如果已登录，直接返回
	if isLoggedIn {
		return &LoginQrcodeResponse{
			Timeout:    "0s",
			Img:        "",
			IsLoggedIn: true,
			Message:    "已登录",
		}, nil
	}

	// 访问登录页并获取二维码（不返回 base64，让用户直接在浏览器中扫码）
	_, loggedIn, err := loginAction.FetchQrcodeImage(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "获取二维码失败")
	}
	// 如果在获取二维码过程中已经登录了
	if loggedIn {
		return &LoginQrcodeResponse{
			Timeout:    "0s",
			Img:        "",
			IsLoggedIn: true,
			Message:    "已登录",
		}, nil
	}

	// 保持页面打开，等待用户扫码登录
	// 这里会阻塞直到登录成功或超时
	logrus.Info("请在弹出的浏览器窗口中扫码登录...")

	timeout := 4 * time.Minute
	ctxTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 等待登录成功
	success := loginAction.WaitForLogin(ctxTimeout)
	if !success {
		return &LoginQrcodeResponse{
			Timeout:    timeout.String(),
			Img:        "",
			IsLoggedIn: false,
			Message:    "扫码登录超时，请重试",
		}, nil
	}

	// 登录成功，保存 cookies
	if err := saveCookies(page); err != nil {
		logrus.Warnf("保存 cookies 失败: %v", err)
	} else {
		logrus.Info("cookies 已保存")
	}

	return &LoginQrcodeResponse{
		Timeout:    "0s",
		Img:        "",
		IsLoggedIn: true,
		Message:    "登录成功",
	}, nil
}

// SearchJobs 搜索职位
func (s *ZhipinService) SearchJobs(ctx context.Context, req *SearchJobsRequest) (*SearchJobsResponse, error) {
	browser := s.browserFactory.NewBrowser(configs.GetEffectiveHeadless(), configs.GetBinPath())
	defer browser.Close()

	page := browser.NewPage()
	defer page.Close()

	// 检查登录状态
	loginAction := zhipin.NewLogin(page)
	isLoggedIn, err := loginAction.CheckLoginStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("ZhipinService.SearchJobs: %w", err)
	}
	if !isLoggedIn {
		return nil, errLoginRequired
	}

	// 搜索
	searchAction := zhipin.NewSearch(page)
	params := zhipin.SearchParams{
		Keyword:    req.Keyword,
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
		return nil, fmt.Errorf("ZhipinService.SearchJobs: %w", err)
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
	logrus.Debugf("[ZhipinService.GetJobDetail] ========== 开始获取职位详情 ==========")
	logrus.Debugf("[ZhipinService.GetJobDetail] 接收到的 jobID: %s", jobID)

	browser := s.browserFactory.NewBrowser(configs.GetEffectiveHeadless(), configs.GetBinPath())
	defer browser.Close()
	logrus.Debugf("[ZhipinService.GetJobDetail] 浏览器实例创建完成")

	page := browser.NewPage()
	defer page.Close()
	logrus.Debugf("[ZhipinService.GetJobDetail] 页面实例创建完成")

	detailAction := zhipin.NewDetail(page)
	logrus.Debugf("[ZhipinService.GetJobDetail] Detail action 实例创建完成，准备调用 GetJobDetail")

	job, err := detailAction.GetJobDetail(ctx, jobID)
	if err != nil {
		logrus.Errorf("[ZhipinService.GetJobDetail] detailAction.GetJobDetail 失败: %v", err)
		return nil, fmt.Errorf("ZhipinService.GetJobDetail: %w", err)
	}

	logrus.Debugf("[ZhipinService.GetJobDetail] 获取到职位信息: %+v", job)

	result := &JobDetailResponse{
		Job: convertJob(job),
	}
	logrus.Debugf("[ZhipinService.GetJobDetail] 转换后的响应: %+v", result)
	logrus.Debugf("[ZhipinService.GetJobDetail] ========== 获取职位详情完成 ==========")

	return result, nil
}

// DeliverJob 投递简历
func (s *ZhipinService) DeliverJob(ctx context.Context, req *DeliverJobRequest) (*DeliverJobResponse, error) {
	logrus.Debugf("[Service.DeliverJob] ========== 开始投递流程 ==========")
	logrus.Debugf("[Service.DeliverJob] JobID: %s", req.JobID)

	// 步骤1: 检查每日投递上限
	logrus.Debugf("[Service.DeliverJob] 步骤1: 检查每日投递上限")
	count, err := s.database.GetTodayDeliveredCount()
	if err != nil {
		logrus.Errorf("[Service.DeliverJob] 获取今日投递数失败: %v", err)
		return nil, fmt.Errorf("ZhipinService.DeliverJob: %w", err)
	}
	logrus.Debugf("[Service.DeliverJob] 今日已投递: %d, 上限: %d", count, configs.MaxDaily)
	if count >= configs.MaxDaily {
		logrus.Warnf("[Service.DeliverJob] 今日投递已达上限: %d/%d", count, configs.MaxDaily)
		return &DeliverJobResponse{
			JobID:   req.JobID,
			Success: false,
			Message: "今日投递已达上限",
		}, nil
	}
	logrus.Debugf("[Service.DeliverJob] 每日投递检查通过")

	// 步骤2: 检查是否已投递
	logrus.Debugf("[Service.DeliverJob] 步骤2: 检查是否已投递")
	isDelivered, err := s.database.IsJobDelivered(req.JobID)
	if err != nil {
		logrus.Warnf("[Service.DeliverJob] 检查投递状态失败: %v", err)
	}
	if isDelivered {
		logrus.Warnf("[Service.DeliverJob] 该职位已投递过: %s", req.JobID)
		return &DeliverJobResponse{
			JobID:   req.JobID,
			Success: false,
			Message: "该职位已投递过",
		}, nil
	}
	logrus.Debugf("[Service.DeliverJob] 未投递过，可以投递")

	// 步骤3: 初始化浏览器
	logrus.Debugf("[Service.DeliverJob] 步骤3: 初始化浏览器")
	browser := s.browserFactory.NewBrowser(configs.GetEffectiveHeadless(), configs.GetBinPath())
	defer browser.Close()
	logrus.Debugf("[Service.DeliverJob] 浏览器初始化完成")

	// 步骤4: 创建浏览器页面
	logrus.Debugf("[Service.DeliverJob] 步骤4: 创建浏览器页面")
	page := browser.NewPage()
	defer page.Close()
	logrus.Debugf("[Service.DeliverJob] 页面创建完成")

	// 步骤5: 执行投递
	logrus.Debugf("[Service.DeliverJob] 步骤5: 执行投递")
	deliverAction := zhipin.NewDeliver(page)
	result, err := deliverAction.DeliverJob(ctx, req.JobID)
	if err != nil {
		logrus.Errorf("[Service.DeliverJob] 投递执行失败: %v", err)
		return nil, fmt.Errorf("ZhipinService.DeliverJob: %w", err)
	}
	logrus.Debugf("[Service.DeliverJob] 投递执行完成, 结果: Success=%v, Message=%s", result.Success, result.Message)

	// 步骤6: 保存投递记录
	logrus.Debugf("[Service.DeliverJob] 步骤6: 保存投递记录")
	if result.Success {
		logrus.Debugf("[Service.DeliverJob] 投递成功，保存记录")
		appliedJob := &AppliedJob{
			JobID:     req.JobID,
			JobTitle:  result.Message,
			Status:    "success",
			AppliedAt: time.Now(),
		}
		err = s.database.SaveAppliedJob(appliedJob)
		if err != nil {
			logrus.Errorf("[Service.DeliverJob] 保存投递记录失败: %v", err)
		} else {
			logrus.Debugf("[Service.DeliverJob] 投递记录保存成功")
		}

		err = s.database.UpdateDeliveryStats(true)
		if err != nil {
			logrus.Errorf("[Service.DeliverJob] 更新统计失败: %v", err)
		} else {
			logrus.Debugf("[Service.DeliverJob] 统计更新成功")
		}
	} else {
		logrus.Warnf("[Service.DeliverJob] 投递失败，更新失败统计")
		err = s.database.UpdateDeliveryStats(false)
		if err != nil {
			logrus.Errorf("[Service.DeliverJob] 更新失败统计失败: %v", err)
		}
	}

	logrus.Debugf("[Service.DeliverJob] ========== 投递流程完成 ==========")
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

	jobs, total, err := s.database.GetDeliveredJobs(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("ZhipinService.DeliveredList: %w", err)
	}

	return &DeliveredListResponse{
		Jobs:  jobs,
		Total: total,
	}, nil
}

// BatchDeliver 批量投递
func (s *ZhipinService) BatchDeliver(ctx context.Context, jobIDs []string) (*BatchDeliverResponse, error) {
	logrus.Debugf("[Service.BatchDeliver] ========== 开始批量投递 ==========")
	logrus.Debugf("[Service.BatchDeliver] 总数: %d", len(jobIDs))

	response := &BatchDeliverResponse{
		Total:   len(jobIDs),
		Results: []DeliverJobResponse{},
	}

	// 步骤1: 获取今日已投递数量
	logrus.Debugf("[Service.BatchDeliver] 步骤1: 获取今日已投递数量")
	todayCount, err := s.database.GetTodayDeliveredCount()
	if err != nil {
		logrus.Errorf("[Service.BatchDeliver] 获取今日投递数失败: %v", err)
		return nil, fmt.Errorf("ZhipinService.BatchDeliver: %w", err)
	}
	logrus.Debugf("[Service.BatchDeliver] 今日已投递: %d, 上限: %d", todayCount, configs.MaxDaily)

	// 步骤2: 初始化浏览器
	logrus.Debugf("[Service.BatchDeliver] 步骤2: 初始化浏览器")
	browser := s.browserFactory.NewBrowser(configs.GetEffectiveHeadless(), configs.GetBinPath())
	defer browser.Close()

	page := browser.NewPage()
	defer page.Close()

	deliverAction := zhipin.NewDeliver(page)
	logrus.Debugf("[Service.BatchDeliver] 浏览器初始化完成")

	// 步骤3: 遍历投递
	logrus.Debugf("[Service.BatchDeliver] 步骤3: 开始遍历投递")
	for i, jobID := range jobIDs {
		logrus.Debugf("[Service.BatchDeliver] 处理第 %d/%d 个职位: %s", i+1, len(jobIDs), jobID)

		// 检查每日上限
		logrus.Debugf("[Service.BatchDeliver] 检查每日上限: 已投 %d + 成功 %d >= 上限 %d", todayCount, response.Success, configs.MaxDaily)
		if todayCount+response.Success >= configs.MaxDaily {
			logrus.Warnf("[Service.BatchDeliver] 今日投递已达上限，停止投递")
			response.Results = append(response.Results, DeliverJobResponse{
				JobID:   jobID,
				Success: false,
				Message: "今日投递已达上限",
			})
			continue
		}

		// 检查是否已投递
		logrus.Debugf("[Service.BatchDeliver] 检查是否已投递: %s", jobID)
		isDelivered, _ := s.database.IsJobDelivered(jobID)
		if isDelivered {
			logrus.Debugf("[Service.BatchDeliver] 该职位已投递过，跳过: %s", jobID)
			response.Results = append(response.Results, DeliverJobResponse{
				JobID:   jobID,
				Success: false,
				Message: "该职位已投递过",
			})
			continue
		}
		logrus.Debugf("[Service.BatchDeliver] 未投递过，可以投递")

		// 随机延时
		logrus.Debugf("[Service.BatchDeliver] 执行随机延时 (3-8秒)")
		delay.Random()

		// 投递
		logrus.Debugf("[Service.BatchDeliver] 执行投递: %s", jobID)
		result, err := deliverAction.DeliverJobFromSearchList(jobID)
		if err != nil {
			logrus.Errorf("[Service.BatchDeliver] 投递异常: %v", err)
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

		// 保存投递记录
		if result.Success {
			logrus.Debugf("[Service.BatchDeliver] 投递成功，保存记录: %s", jobID)
			response.Success++
			todayCount++

			appliedJob := &AppliedJob{
				JobID:     jobID,
				JobTitle:  result.Message,
				Status:    "success",
				AppliedAt: time.Now(),
			}
			err = s.database.SaveAppliedJob(appliedJob)
			if err != nil {
				logrus.Errorf("[Service.BatchDeliver] 保存投递记录失败: %v", err)
			}
			err = s.database.UpdateDeliveryStats(true)
			if err != nil {
				logrus.Errorf("[Service.BatchDeliver] 更新统计失败: %v", err)
			}
		} else {
			logrus.Warnf("[Service.BatchDeliver] 投递失败: %s - %s", jobID, result.Message)
			response.Failed++
			_ = UpdateDeliveryStats(false)
		}

		logrus.Infof("批量投递进度: %d/%d, 成功: %d, 失败: %d", i+1, len(jobIDs), response.Success, response.Failed)
	}

	logrus.Infof("[Service.BatchDeliver] 批量投递完成: 总数=%d, 成功=%d, 失败=%d", len(jobIDs), response.Success, response.Failed)
	logrus.Debugf("[Service.BatchDeliver] ========== 批量投递完成 ==========")

	return response, nil
}

// GetStats 获取投递统计
func (s *ZhipinService) GetStats(ctx context.Context) (*StatsResponse, error) {
	// 今日统计
	todayStats, err := s.database.GetTodayStats()
	if err != nil {
		return nil, fmt.Errorf("ZhipinService.GetStats: %w", err)
	}

	// 总统计
	total, err := s.database.GetTotalStats()
	if err != nil {
		return nil, fmt.Errorf("ZhipinService.GetStats: %w", err)
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
	return s.database.SaveCronTask(task)
}

// StopCron 停止定时任务
func (s *ZhipinService) StopCron(ctx context.Context, taskID int) error {
	cronMgr := zhipin.GetCronManager()
	err := cronMgr.RemoveTask(taskID)
	if err != nil {
		return err
	}

	return s.database.UpdateCronTask(taskID, false)
}

// ListMessages 获取消息列表
func (s *ZhipinService) ListMessages(ctx context.Context) (*MessageListResponse, error) {
	logrus.Debugf("[ZhipinService.ListMessages] ========== 开始获取消息列表 ==========")

	browser := s.browserFactory.NewBrowser(configs.GetEffectiveHeadless(), configs.GetBinPath())
	defer browser.Close()
	logrus.Debugf("[ZhipinService.ListMessages] 浏览器实例创建完成")

	page := browser.NewPage()
	defer page.Close()
	logrus.Debugf("[ZhipinService.ListMessages] 页面实例创建完成")

	// 检查登录状态
	loginAction := zhipin.NewLogin(page)
	isLoggedIn, err := loginAction.CheckLoginStatus(ctx)
	if err != nil {
		logrus.Errorf("[ZhipinService.ListMessages] 检查登录状态失败: %v", err)
		return nil, fmt.Errorf("ZhipinService.ListMessages: %w", err)
	}
	if !isLoggedIn {
		logrus.Warnf("[ZhipinService.ListMessages] 未登录")
		return nil, errLoginRequired
	}
	logrus.Debugf("[ZhipinService.ListMessages] 登录状态检查通过")

	// 获取消息列表
	msgAction := zhipin.NewMessageAction(page)
	result, err := msgAction.ListMessages(ctx)
	if err != nil {
		logrus.Errorf("[ZhipinService.ListMessages] msgAction.ListMessages 失败: %v", err)
		return nil, fmt.Errorf("ZhipinService.ListMessages: %w", err)
	}

	logrus.Debugf("[ZhipinService.ListMessages] 获取到消息: %d 条", len(result.Messages))
	logrus.Debugf("[ZhipinService.ListMessages] ========== 获取消息列表完成 ==========")

	return &MessageListResponse{
		Messages: convertMessages(result.Messages),
	}, nil
}

// DeleteMessage 删除消息
// 支持批量删除多个消息，每个消息通过 person_name、company_name、job_title 匹配
func (s *ZhipinService) DeleteMessage(ctx context.Context, req *DeleteMessageRequest) (*DeleteMessageResponse, error) {
	logrus.Debugf("[ZhipinService.DeleteMessage] ========== 开始删除消息 ==========")
	logrus.Debugf("[ZhipinService.DeleteMessage] 待删除消息数: %d", len(req.Messages))

	if len(req.Messages) == 0 {
		return &DeleteMessageResponse{
			Success:  false,
			Messages: []string{"消息列表为空"},
		}, nil
	}

	// 检查每个消息的必填字段
	for i, msg := range req.Messages {
		if msg.PersonName == "" {
			return &DeleteMessageResponse{
				Success:  false,
				Messages: []string{fmt.Sprintf("第 %d 条消息的 person_name 不能为空", i+1)},
			}, nil
		}
	}

	browser := s.browserFactory.NewBrowser(configs.GetEffectiveHeadless(), configs.GetBinPath())
	defer browser.Close()
	logrus.Debugf("[ZhipinService.DeleteMessage] 浏览器实例创建完成")

	page := browser.NewPage()
	defer page.Close()
	logrus.Debugf("[ZhipinService.DeleteMessage] 页面实例创建完成")

	// 检查登录状态
	loginAction := zhipin.NewLogin(page)
	isLoggedIn, err := loginAction.CheckLoginStatus(ctx)
	if err != nil {
		logrus.Errorf("[ZhipinService.DeleteMessage] 检查登录状态失败: %v", err)
		return nil, fmt.Errorf("ZhipinService.DeleteMessage: %w", err)
	}
	if !isLoggedIn {
		logrus.Warnf("[ZhipinService.DeleteMessage] 未登录")
		return nil, errLoginRequired
	}
	logrus.Debugf("[ZhipinService.DeleteMessage] 登录状态检查通过")

	msgAction := zhipin.NewMessageAction(page)

	response := &DeleteMessageResponse{
		Total:    len(req.Messages),
		Messages: make([]string, 0, len(req.Messages)),
	}

	// 遍历删除每个消息
	for i, filter := range req.Messages {
		logrus.Infof("[ZhipinService.DeleteMessage] 删除第 %d/%d 个消息: %s %s %s",
			i+1, len(req.Messages), filter.PersonName, filter.CompanyName, filter.JobTitle)

		// 先刷新页面以获取最新消息列表
		logrus.Debugf("[ZhipinService.DeleteMessage] 刷新消息列表")
		_, err := msgAction.ListMessages(ctx)
		if err != nil {
			logrus.Warnf("[ZhipinService.DeleteMessage] 刷新消息列表失败: %v", err)
		}
		// 随机延时 1-3 秒
		delay.Short()

		// 删除消息
		err = msgAction.DeleteMessage(ctx, filter.PersonName, filter.CompanyName, filter.JobTitle)
		if err != nil {
			logrus.Errorf("[ZhipinService.DeleteMessage] 删除消息失败: %v", err)
			response.Failed++
			response.Messages = append(response.Messages, fmt.Sprintf("%s: 失败 - %s", filter.PersonName, err.Error()))
		} else {
			logrus.Infof("[ZhipinService.DeleteMessage] 删除成功: %s", filter.PersonName)
			response.Deleted++
			response.Messages = append(response.Messages, fmt.Sprintf("%s: 成功", filter.PersonName))
		}
		// 随机延时 3-5 秒
		delay.Medium()
	}

	response.Success = response.Failed == 0
	logrus.Debugf("[ZhipinService.DeleteMessage] ========== 删除消息完成 ==========")
	logrus.Infof("[ZhipinService.DeleteMessage] 结果: 总数=%d, 成功=%d, 失败=%d",
		response.Total, response.Deleted, response.Failed)

	return response, nil
}

// SendMessage 发送消息
func (s *ZhipinService) SendMessage(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, error) {
	logrus.Debugf("[ZhipinService.SendMessage] ========== 开始发送消息 ==========")
	logrus.Debugf("[ZhipinService.SendMessage] 目标: %s, 内容: %s", req.PersonName, req.Content)

	// 参数验证
	if req.PersonName == "" {
		return &SendMessageResponse{
			Success: false,
			Message: "person_name 不能为空",
		}, nil
	}
	if req.Content == "" {
		return &SendMessageResponse{
			Success: false,
			Message: "消息内容不能为空",
		}, nil
	}

	// 创建浏览器
	browser := s.browserFactory.NewBrowser(configs.GetEffectiveHeadless(), configs.GetBinPath())
	defer browser.Close()

	page := browser.NewPage()
	defer page.Close()

	// 检查登录状态
	loginAction := zhipin.NewLogin(page)
	isLoggedIn, err := loginAction.CheckLoginStatus(ctx)
	if err != nil {
		logrus.Errorf("[ZhipinService.SendMessage] 检查登录状态失败: %v", err)
		return nil, fmt.Errorf("ZhipinService.SendMessage: %w", err)
	}
	if !isLoggedIn {
		logrus.Warnf("[ZhipinService.SendMessage] 未登录")
		return nil, errLoginRequired
	}
	logrus.Debugf("[ZhipinService.SendMessage] 登录状态检查通过")

	// 发送消息
	msgAction := zhipin.NewMessageAction(page)
	result, err := msgAction.SendMessage(ctx, req.PersonName, req.CompanyName, req.JobTitle, req.Content)
	if err != nil {
		logrus.Errorf("[ZhipinService.SendMessage] 发送消息失败: %v", err)
		return &SendMessageResponse{
			Success:    false,
			PersonName: req.PersonName,
			Message:    "发送失败: " + err.Error(),
		}, err
	}

	logrus.Debugf("[ZhipinService.SendMessage] ========== 发送消息完成 ==========")

	return &SendMessageResponse{
		Success:    result.Success,
		PersonName: result.PersonName,
		Message:    result.Message,
	}, nil
}

// 辅助函数

func newBrowser() *headless_browser.Browser {
	return browser.NewBrowser(configs.GetEffectiveHeadless(), browser.WithBinPath(configs.GetBinPath()))
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

func convertMessages(messages []zhipin.Message) []Message {
	result := make([]Message, len(messages))
	for i, msg := range messages {
		result[i] = convertMessage(&msg)
	}
	return result
}

func convertMessage(msg *zhipin.Message) Message {
	return Message{
		PersonName:    msg.PersonName,
		CompanyName:   msg.CompanyName,
		JobTitle:      msg.JobTitle,
		Avatar:        msg.Avatar,
		MessageDigest: msg.MessageDigest,
		Time:          msg.Time,
		UnreadCount:   msg.UnreadCount,
		Status:        MessageStatus(msg.Status),
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
