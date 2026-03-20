package zhipin

import (
	"context"
	"fmt"
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
	logrus.Infof("========== [DEBUG SearchJobs] 开始搜索职位 ==========")
	logrus.Infof("[DEBUG SearchJobs] 搜索参数: keyword=%s, page=%d, pageSize=%d, district=%s",
		params.Keyword, params.Page, params.PageSize, params.District)

	// ===== DEBUG: 步骤1 - 访问搜索页面 =====
	logrus.Debugf("[DEBUG SearchJobs] 步骤1: 访问搜索页面 https://www.zhipin.com/web/geek/job")
	err := s.page.Navigate("https://www.zhipin.com/web/geek/job")
	if err != nil {
		logrus.Errorf("[DEBUG SearchJobs] 访问搜索页失败: %v", err)
		return nil, errors.Wrap(err, "访问搜索页失败")
	}
	logrus.Debugf("[DEBUG SearchJobs] 页面导航成功")

	// ===== DEBUG: 步骤2 - 等待页面加载 =====
	logrus.Debugf("[DEBUG SearchJobs] 步骤2: 等待页面加载")
	startWait := time.Now()
	s.page.WaitLoad()
	logrus.Debugf("[DEBUG SearchJobs] 页面加载完成, 耗时: %v", time.Since(startWait))

	// 获取页面加载后的状态
	pageURLAfterNav := getPageURL(s.page)
	pageTitleAfterNav := getPageTitle(s.page)
	logrus.Debugf("[DEBUG SearchJobs] 导航后页面URL: %s", pageURLAfterNav)
	logrus.Debugf("[DEBUG SearchJobs] 导航后页面标题: %s", pageTitleAfterNav)

	// ===== DEBUG: 步骤3 - 随机延时 =====
	logrus.Debugf("[DEBUG SearchJobs] 步骤3: 执行随机延时")
	delayStart := time.Now()
	randomDelay()
	logrus.Debugf("[DEBUG SearchJobs] 随机延时完成, 耗时: %v", time.Since(delayStart))

	// ===== DEBUG: 步骤4 - 执行搜索交互 =====
	logrus.Debugf("[DEBUG SearchJobs] 步骤4: 执行搜索交互 performSearch")
	searchStart := time.Now()
	err = s.performSearch(params)
	if err != nil {
		logrus.Errorf("[DEBUG SearchJobs] 执行搜索交互失败: %v", err)
		return nil, err
	}
	logrus.Debugf("[DEBUG SearchJobs] 执行搜索交互完成, 耗时: %v", time.Since(searchStart))

	// 获取搜索后的页面状态
	pageURLAfterSearch := getPageURL(s.page)
	pageTitleAfterSearch := getPageTitle(s.page)
	logrus.Debugf("[DEBUG SearchJobs] 搜索后页面URL: %s", pageURLAfterSearch)
	logrus.Debugf("[DEBUG SearchJobs] 搜索后页面标题: %s", pageTitleAfterSearch)

	// ===== DEBUG: 步骤5 - 解析搜索结果 =====
	logrus.Debugf("[DEBUG SearchJobs] 步骤5: 解析搜索结果 parseSearchResults")
	parseStart := time.Now()
	result, err := s.parseSearchResults(params.Page, params.PageSize)
	if err != nil {
		logrus.Errorf("[DEBUG SearchJobs] 解析搜索结果失败: %v", err)
		return nil, err
	}
	logrus.Debugf("[DEBUG SearchJobs] 解析搜索结果完成, 耗时: %v", time.Since(parseStart))

	logrus.Infof("[DEBUG SearchJobs] 搜索完成! 共找到 %d 个职位", result.Total)
	logrus.Infof("========== [DEBUG SearchJobs] 搜索职位结束 ==========")

	return result, nil
}

// performSearch 执行搜索交互
func (s *Search) performSearch(params SearchParams) error {
	logrus.Debugf("[DEBUG performSearch] 开始执行搜索交互, 关键词: %s", params.Keyword)

	// ===== DEBUG: 步骤1 - 查找搜索输入框 =====
	logrus.Debugf("[DEBUG performSearch] 步骤1: 查找搜索输入框")

	// 先获取页面当前状态
	pageState := getPageState(s.page)
	logrus.Debugf("[DEBUG performSearch] 页面当前状态: %s", pageState)

	inputSelectors := []string{
		".search-input-box .input",       // 搜索框容器内的input
		".c-search-input .input",        // 搜索输入容器
		"input.input[placeholder*='职位']", // 带placeholder的input
		".search-input",                 // 旧选择器
		".ka-input",                     // 旧选择器
		"input[name='query']",           // 旧选择器
		"#keyword",                      // 旧选择器
		"input[placeholder*='职位']",    // 旧选择器
	}
	var inputEl *rod.Element
	var err error

	for _, selector := range inputSelectors {
		logrus.Debugf("[DEBUG performSearch] 尝试选择器: %s", selector)
		inputEl, err = s.page.Element(selector)
		if err == nil {
			logrus.Debugf("[DEBUG performSearch] 找到输入框: %s", selector)
			break
		}
		logrus.Debugf("[DEBUG performSearch] 选择器 %s 未找到: %v", selector, err)
	}

	if inputEl == nil {
		logrus.Errorf("[DEBUG performSearch] 找不到搜索输入框, 已尝试: %v", inputSelectors)
		return errors.Wrap(err, "找不到搜索输入框")
	}

	// 获取输入框信息
	inputClass, _ := inputEl.Attribute("class")
	inputID, _ := inputEl.Attribute("id")
	inputPlaceholder, _ := inputEl.Attribute("placeholder")
	inputName, _ := inputEl.Attribute("name")
	logrus.Debugf("[DEBUG performSearch] 输入框信息: class=%v, id=%v, name=%v, placeholder=%v",
		inputClass, inputID, inputName, inputPlaceholder)

	// ===== DEBUG: 步骤2 - 清空并输入关键词 =====
	logrus.Debugf("[DEBUG performSearch] 步骤2: 清空输入框并输入关键词")
	// 先清空输入框
	err = inputEl.Input("")
	if err != nil {
		logrus.Errorf("[DEBUG performSearch] 清空输入框失败: %v", err)
		return errors.Wrap(err, "清空输入框失败")
	}

	// 输入关键词
	inputStart := time.Now()
	err = inputEl.Input(params.Keyword)
	if err != nil {
		logrus.Errorf("[DEBUG performSearch] 输入关键词失败: %v", err)
		return errors.Wrap(err, "输入关键词失败")
	}
	logrus.Debugf("[DEBUG performSearch] 关键词输入完成, 耗时: %v", time.Since(inputStart))

	// 获取输入后的值
	inputValueResult, _ := s.page.Eval("(el) => el.value", inputEl)
	inputValue := ""
	if inputValueResult != nil {
		inputValue = inputValueResult.Value.Get("value").Str()
	}
	logrus.Debugf("[DEBUG performSearch] 输入框当前值: %s", inputValue)

	// ===== DEBUG: 步骤3 - 等待输入生效 =====
	logrus.Debugf("[DEBUG performSearch] 步骤3: 等待输入生效")
	inputEffectStart := time.Now()
	time.Sleep(500 * time.Millisecond)
	logrus.Debugf("[DEBUG performSearch] 等待完成, 耗时: %v", time.Since(inputEffectStart))

	// ===== DEBUG: 步骤4 - 按回车键提交搜索 =====
	logrus.Debugf("[DEBUG performSearch] 步骤4: 按回车键提交搜索")
	typeStart := time.Now()
	err = s.page.Keyboard.Type(input.Enter)
	if err != nil {
		logrus.Errorf("[DEBUG performSearch] 提交搜索失败: %v", err)
		return errors.Wrap(err, "提交搜索失败")
	}
	logrus.Debugf("[DEBUG performSearch] 回车键提交完成, 耗时: %v", time.Since(typeStart))

	// ===== DEBUG: 步骤5 - 等待搜索结果加载 =====
	logrus.Debugf("[DEBUG performSearch] 步骤5: 等待搜索结果加载")
	waitLoadStart := time.Now()
	err = s.page.WaitLoad()
	if err != nil {
		logrus.Errorf("[DEBUG performSearch] 等待搜索结果加载失败: %v", err)
		return errors.Wrap(err, "等待搜索结果加载失败")
	}
	logrus.Debugf("[DEBUG performSearch] 搜索结果加载完成, 耗时: %v", time.Since(waitLoadStart))

	// ===== DEBUG: 步骤6 - 等待搜索结果列表出现 =====
	logrus.Debugf("[DEBUG performSearch] 步骤6: 等待搜索结果列表出现")
	listWaitStart := time.Now()
	time.Sleep(2 * time.Second)
	logrus.Debugf("[DEBUG performSearch] 列表等待完成, 耗时: %v", time.Since(listWaitStart))

	// 获取最终页面状态
	pageURLFinal := getPageURL(s.page)
	pageTitleFinal := getPageTitle(s.page)
	logrus.Debugf("[DEBUG performSearch] 搜索完成后页面URL: %s", pageURLFinal)
	logrus.Debugf("[DEBUG performSearch] 搜索完成后页面标题: %s", pageTitleFinal)

	logrus.Debugf("[DEBUG performSearch] 搜索交互执行完成")

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

// getPageURL 获取页面URL
func getPageURL(page *rod.Page) string {
	if page == nil {
		return "page is nil"
	}
	result, err := page.Eval("() => window.location.href")
	if err != nil {
		logrus.Warnf("[DEBUG] getPageURL failed: %v", err)
		return ""
	}
	if result == nil {
		logrus.Warnf("[DEBUG] getPageURL result is nil")
		return ""
	}
	return result.Value.Get("value").Str()
}

// getPageTitle 获取页面标题
func getPageTitle(page *rod.Page) string {
	if page == nil {
		return "page is nil"
	}
	result, err := page.Eval("() => document.title")
	if err != nil {
		logrus.Warnf("[DEBUG] getPageTitle failed: %v", err)
		return ""
	}
	if result == nil {
		logrus.Warnf("[DEBUG] getPageTitle result is nil")
		return ""
	}
	return result.Value.Get("value").Str()
}

// getPageState 获取页面当前状态信息
func getPageState(page *rod.Page) string {
	if page == nil {
		return "page is nil"
	}

	// 获取页面URL
	url := getPageURL(page)
	// 获取页面标题
	title := getPageTitle(page)
	// 获取body是否存在
	bodyExistsResult, _ := page.Eval("() => document.body !== null")
	bodyExists := "unknown"
	if bodyExistsResult != nil {
		bodyExists = fmt.Sprintf("%v", bodyExistsResult.Value)
	}
	// 获取页面是否加载完成
	readyStateResult, _ := page.Eval("() => document.readyState")
	readyState := "unknown"
	if readyStateResult != nil {
		readyState = readyStateResult.Value.Str()
	}

	return fmt.Sprintf("url=%s, title=%s, bodyExists=%s, readyState=%s",
		url, title, bodyExists, readyState)
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
