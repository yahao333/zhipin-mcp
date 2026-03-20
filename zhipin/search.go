package zhipin

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/yahao333/zhipin-mcp/configs"
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
	logrus.Infof("搜索职位: keyword=%s, page=%d", params.Keyword, params.Page)

	// 先访问搜索页面
	err := s.page.Navigate("https://www.zhipin.com/web/geek/job")
	if err != nil {
		return nil, errors.Wrap(err, "访问搜索页失败")
	}

	// 等待页面加载
	s.page.WaitLoad()
	randomDelay()

	// 执行搜索交互
	err = s.performSearch(params)
	if err != nil {
		return nil, err
	}

	// 解析搜索结果
	result, err := s.parseSearchResults(params.Page, params.PageSize)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// performSearch 执行搜索交互
func (s *Search) performSearch(params SearchParams) error {
	// 等待搜索输入框出现
	inputEl, err := s.page.Element(".search-input")
	if err != nil {
		// 尝试其他可能的输入框选择器
		inputEl, err = s.page.Element(".ka-input")
		if err != nil {
			return errors.Wrap(err, "找不到搜索输入框")
		}
	}

	// 清空输入框并输入关键词
	err = inputEl.Input(params.Keyword)
	if err != nil {
		return errors.Wrap(err, "输入关键词失败")
	}

	// 等待一下让输入生效
	time.Sleep(500 * time.Millisecond)

	// 按回车键提交搜索
	err = s.page.Keyboard.Type(input.Enter)
	if err != nil {
		return errors.Wrap(err, "提交搜索失败")
	}

	// 等待搜索结果加载
	err = s.page.WaitLoad()
	if err != nil {
		return errors.Wrap(err, "等待搜索结果加载失败")
	}

	// 等待搜索结果列表出现
	time.Sleep(2 * time.Second)

	return nil
}

// parseSearchResults 解析搜索结果
func (s *Search) parseSearchResults(page, pageSize int) (*SearchResult, error) {
	result := &SearchResult{
		Page:     page,
		PageSize: pageSize,
	}

	// ===== DEBUG: 获取页面基本信息 =====
	// 使用 JS 获取页面URL和标题
	pageURLResult, _ := s.page.Eval("() => window.location.href")
	pageURL := ""
	if pageURLResult != nil {
		pageURL = pageURLResult.Value.Get("value").Str()
	}
	pageTitleResult, _ := s.page.Eval("() => document.title")
	pageTitle := ""
	if pageTitleResult != nil {
		pageTitle = pageTitleResult.Value.Get("value").Str()
	}
	logrus.Debugf("[DEBUG parseSearchResults] 页面URL: %s", pageURL)
	logrus.Debugf("[DEBUG parseSearchResults] 页面标题: %s", pageTitle)

	// 获取页面HTML用于调试
	pageHTML, _ := s.page.HTML()
	// 只保存前5000字符用于调试
	debugHTML := pageHTML
	if len(debugHTML) > 5000 {
		debugHTML = debugHTML[:5000]
	}
	logrus.Debugf("[DEBUG parseSearchResults] 页面HTML前5000字符: %s", debugHTML)

	// 尝试多个选择器来获取职位列表
	var jobCards []*rod.Element
	var err error
	var selectorUsed string

	// 选择器列表和调试信息
	selectors := []string{
		".rec-job-list .job-card-box",
		".job-card-wrap .job-card-box",
		".job-list li",
		".job-card",
		".job-list .job-card",
		".geek-job-list .job-card-box",
	}

	for _, selector := range selectors {
		logrus.Debugf("[DEBUG parseSearchResults] 尝试选择器: %s", selector)
		jobCards, err = s.page.Elements(selector)
		if err == nil && len(jobCards) > 0 {
			selectorUsed = selector
			logrus.Debugf("[DEBUG parseSearchResults] 选择器 %s 成功，找到 %d 个元素", selector, len(jobCards))
			break
		}
		logrus.Debugf("[DEBUG parseSearchResults] 选择器 %s 失败: err=%v, len=%d", selector, err, len(jobCards))
	}

	// 备用：尝试获取body下所有直接子元素的div
	if len(jobCards) == 0 {
		logrus.Debugf("[DEBUG parseSearchResults] 尝试获取页面中所有可能包含职位的元素...")
		// 尝试获取包含"职位"的元素
		allDivs, _ := s.page.Elements("div")
		logrus.Debugf("[DEBUG parseSearchResults] 页面中共有 %d 个div元素", len(allDivs))
		for i, div := range allDivs {
			if i < 5 { // 只打印前5个
				class, _ := div.Attribute("class")
				id, _ := div.Attribute("id")
				logrus.Debugf("[DEBUG parseSearchResults] div[%d] class=%v, id=%v", i, class, id)
			}
		}
	}

	if err != nil || len(jobCards) == 0 {
		logrus.Warn("未找到职位列表")
		logrus.Warnf("[DEBUG parseSearchResults] 最终使用的选择器: %s, 找到元素: %d, error: %v", selectorUsed, len(jobCards), err)
		result.Jobs = []Job{}
		result.Total = 0
		return result, nil
	}

	logrus.Infof("找到 %d 个职位卡片, 使用选择器: %s", len(jobCards), selectorUsed)

	// ===== DEBUG: 保存完整页面HTML到文件 =====
	debugFilePath := "./docs/search_debug_" + time.Now().Format("20060102_150405") + ".html"
	_ = os.MkdirAll("./docs", 0755)
	_ = os.WriteFile(debugFilePath, []byte(pageHTML), 0644)
	logrus.Infof("[DEBUG parseSearchResults] 页面HTML已保存到: %s", debugFilePath)

	// 解析每个职位
	for idx, card := range jobCards {
		logrus.Debugf("[DEBUG parseSearchResults] 开始解析第 %d 个职位卡片", idx+1)

		// 获取卡片HTML用于调试
		cardHTML, _ := card.HTML()
		cardDebugLen := 1000
		if len(cardHTML) < cardDebugLen {
			cardDebugLen = len(cardHTML)
		}
		logrus.Debugf("[DEBUG parseSearchResults] 卡片%d HTML前%d字符: %s", idx+1, cardDebugLen, cardHTML[:cardDebugLen])

		// 获取卡片的关键属性
		cardJobID, _ := card.Attribute("data-jobid")
		cardJobURL, _ := card.Attribute("data-job-url")
		logrus.Debugf("[DEBUG parseSearchResults] 卡片%d 属性: data-jobid=%v, data-job-url=%v", idx+1, cardJobID, cardJobURL)

		job, err := s.parseJobCard(card)
		if err != nil {
			logrus.Warnf("解析职位卡片失败: %v", err)
			logrus.Debugf("[DEBUG parseSearchResults] 卡片%d 解析失败详情: %v", idx+1, err)
			continue
		}
		logrus.Debugf("[DEBUG parseSearchResults] 卡片%d 解析成功: jobID=%s, title=%s, company=%s",
			idx+1, job.ID, job.Title, job.CompanyName)
		result.Jobs = append(result.Jobs, job)
	}

	result.Total = len(result.Jobs)
	logrus.Infof("[DEBUG parseSearchResults] 解析完成, 共解析 %d 个职位", result.Total)

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

	// 尝试从data-job-url获取URL
	jobURL, _ := card.Attribute("data-job-url")
	if jobURL != nil {
		job.URL = "https://www.zhipin.com" + *jobURL
	}

	// 获取职位标题 - 尝试多个选择器
	titleEl, err := card.Element(".job-title")
	if err == nil {
		job.Title, _ = titleEl.Text()
	}

	// 获取公司名称 - 尝试 .boss-name
	companyEl, err := card.Element(".boss-name")
	if err == nil {
		job.CompanyName, _ = companyEl.Text()
	}

	// 获取薪资 - 尝试 .job-salary
	salaryEl, err := card.Element(".job-salary")
	if err == nil {
		job.SalaryRange, _ = salaryEl.Text()
	}

	// 获取城市/地点 - 尝试 .company-location
	locationEl, err := card.Element(".company-location")
	if err == nil {
		job.City, _ = locationEl.Text()
	}

	// 获取HR信息 - 尝试 .name
	hrEl, err := card.Element(".name")
	if err == nil {
		job.HRName, _ = hrEl.Text()
	}

	// 解析 tag-list 获取经验、学历、标签
	tagList, err := card.Elements(".tag-list li")
	if err == nil && len(tagList) > 0 {
		for i, tagEl := range tagList {
			tagText, _ := tagEl.Text()
			tagText = strings.TrimSpace(tagText)
			if tagText == "" {
				continue
			}

			switch i {
			case 0:
				// 第一个通常是经验要求
				job.Experience = tagText
			case 1:
				// 第二个通常是学历要求
				job.Education = tagText
			default:
				// 其他的作为标签
				if tagText != "" {
					job.Tags = append(job.Tags, tagText)
				}
			}
		}
	}

	// 获取公司规模 - 尝试 .company-info 下的文本
	companyInfoEl, err := card.Element(".company-info")
	if err == nil {
		companyInfo, _ := companyInfoEl.Text()
		// 公司规模通常在描述中，如"100-999人"
		if strings.Contains(companyInfo, "人") {
			job.CompanySize = strings.TrimSpace(companyInfo)
		}
	}

	// 获取职位详情URL - 尝试从 a 标签的 href 获取
	linkEl, err := card.Element("a")
	if err == nil {
		href, _ := linkEl.Attribute("href")
		if href != nil && *href != "" {
			if !strings.HasPrefix(*href, "http") {
				job.URL = "https://www.zhipin.com" + *href
			} else {
				job.URL = *href
			}
		}
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
