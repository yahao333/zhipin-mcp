package zhipin

import (
	"context"
	"time"

	"github.com/go-rod/rod"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/xpzouying/zhipin-mcp/configs"
)

// Search 搜索操作
type Search struct {
	page *rod.Page
}

// NewSearch 创建搜索操作
func NewSearch(page *rod.Page) *Search {
	return &Search{page: page}
}

// SearchJobs 搜索职位
func (s *Search) SearchJobs(ctx context.Context, params SearchParams) (*SearchResult, error) {
	// 构建搜索URL
	url := s.buildSearchURL(params)

	logrus.Infof("搜索职位: keyword=%s, city=%s, page=%d", params.Keyword, params.City, params.Page)

	// 访问搜索页面
	err := s.page.Navigate(url)
	if err != nil {
		return nil, errors.Wrap(err, "访问搜索页失败")
	}

	// 等待页面加载
	s.page.WaitLoad()
	randomDelay()

	// 解析搜索结果
	result, err := s.parseSearchResults(params.Page, params.PageSize)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// buildSearchURL 构建搜索URL
func (s *Search) buildSearchURL(params SearchParams) string {
	url := "https://www.zhipin.com/web/geek/job"

	return url
}

// parseSearchResults 解析搜索结果
func (s *Search) parseSearchResults(page, pageSize int) (*SearchResult, error) {
	result := &SearchResult{
		Page:     page,
		PageSize: pageSize,
	}

	// 等待职位列表加载
	time.Sleep(2 * time.Second)

	// 获取所有职位卡片
	jobCards, err := s.page.Elements(".job-list li")
	if err != nil || len(jobCards) == 0 {
		logrus.Warn("未找到职位列表")
		result.Jobs = []Job{}
		result.Total = 0
		return result, nil
	}

	// 解析每个职位
	for _, card := range jobCards {
		job, err := s.parseJobCard(card)
		if err != nil {
			logrus.Warnf("解析职位卡片失败: %v", err)
			continue
		}
		result.Jobs = append(result.Jobs, job)
	}

	result.Total = len(result.Jobs)

	return result, nil
}

// parseJobCard 解析职位卡片
func (s *Search) parseJobCard(card *rod.Element) (Job, error) {
	job := Job{
		UpdatedAt: time.Now(),
	}

	// 获取职位ID
	jobID, _ := card.Attribute("data-jobid")
	if jobID != nil {
		job.ID = *jobID
	}

	// 获取职位标题
	titleEl, err := card.Element(".job-title")
	if err == nil {
		job.Title, _ = titleEl.Text()
	}

	// 获取公司名称
	companyEl, err := card.Element(".company-name")
	if err == nil {
		job.CompanyName, _ = companyEl.Text()
	}

	// 获取薪资
	salaryEl, err := card.Element(".salary")
	if err == nil {
		job.SalaryRange, _ = salaryEl.Text()
	}

	// 获取城市
	cityEl, err := card.Element(".job-area")
	if err == nil {
		job.City, _ = cityEl.Text()
	}

	// 获取HR信息
	hrEl, err := card.Element(".name")
	if err == nil {
		job.HRName, _ = hrEl.Text()
	}

	return job, nil
}

// randomDelay 随机延时
func randomDelay() {
	minDelay := configs.MinDelay
	maxDelay := configs.MaxDelay
	if minDelay <= 0 {
		minDelay = 3000
	}
	if maxDelay <= 0 {
		maxDelay = 8000
	}
	time.Sleep(time.Duration(minDelay) * time.Millisecond)
}
