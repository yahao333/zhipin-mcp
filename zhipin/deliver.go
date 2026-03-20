package zhipin

import (
	"context"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Deliver 投递操作
type Deliver struct {
	page *rod.Page
}

// NewDeliver 创建投递操作
func NewDeliver(page *rod.Page) *Deliver {
	return &Deliver{page: page}
}

// DeliverJob 投递简历
func (d *Deliver) DeliverJob(ctx context.Context, jobID string) (*DeliverResult, error) {
	logrus.Debugf("[DeliverJob] ========== 开始投递职位 ==========")
	logrus.Debugf("[DeliverJob] JobID: %s", jobID)

	// 先访问职位详情页
	logrus.Debugf("[DeliverJob] 步骤1: 获取职位详情")
	detail := NewDetail(d.page)
	job, err := detail.GetJobDetail(ctx, jobID)
	if err != nil {
		logrus.Errorf("[DeliverJob] 获取职位详情失败: %v", err)
		return &DeliverResult{
			JobID:   jobID,
			Success: false,
			Message: "获取职位详情失败: " + err.Error(),
		}, err
	}
	logrus.Debugf("[DeliverJob] 职位信息: %s | %s | %s", job.Title, job.CompanyName, job.SalaryRange)

	// 点击投递按钮
	logrus.Debugf("[DeliverJob] 步骤2: 点击投递按钮")
	err = d.clickDeliverButton()
	if err != nil {
		logrus.Errorf("[DeliverJob] 点击投递按钮失败: %v", err)
		return &DeliverResult{
			JobID:   jobID,
			Success: false,
			Message: "点击投递按钮失败: " + err.Error(),
		}, err
	}
	logrus.Debugf("[DeliverJob] 投递按钮已点击")

	// 等待投递结果
	logrus.Debugf("[DeliverJob] 步骤3: 等待投递结果 (2秒)")
	time.Sleep(2 * time.Second)

	// 检查投递结果
	logrus.Debugf("[DeliverJob] 步骤4: 检查投递结果")
	result, err := d.checkDeliverResult()
	if err != nil {
		logrus.Errorf("[DeliverJob] 检查投递结果失败: %v", err)
		return &DeliverResult{
			JobID:   jobID,
			Success: false,
			Message: "检查投递结果失败: " + err.Error(),
		}, err
	}

	logrus.Debugf("[DeliverJob] 投递结果: Success=%v, Message=%s", result.Success, result.Message)

	if result.Success {
		logrus.Infof("职位投递成功: %s - %s", jobID, job.Title)
	} else {
		logrus.Warnf("职位投递失败: %s - %s, %s", jobID, job.Title, result.Message)
	}

	logrus.Debugf("[DeliverJob] ========== 投递职位完成 ==========")
	return result, nil
}

// DeliverJobFromSearchList 从搜索列表投递
func (d *Deliver) DeliverJobFromSearchList(jobID string) (*DeliverResult, error) {
	logrus.Debugf("[DeliverJobFromSearchList] ========== 从搜索列表投递 ==========")
	logrus.Debugf("[DeliverJobFromSearchList] JobID: %s", jobID)

	// 查找职位卡片
	logrus.Debugf("[DeliverJobFromSearchList] 查找职位卡片")
	selector := "[data-jobid='" + jobID + "']"
	card, err := d.page.Element(selector)
	if err != nil {
		logrus.Debugf("[DeliverJobFromSearchList] 未找到职位卡片，切换到详情页投递: %v", err)
		return d.DeliverJobFromDetail(jobID)
	}
	logrus.Debugf("[DeliverJobFromSearchList] 已找到职位卡片")

	// 点击投递按钮 - 尝试多个选择器
	logrus.Debugf("[DeliverJobFromSearchList] 查找投递按钮")
	selectors := []string{
		".btn-startchat", // "立即沟通" 按钮 - 新版页面结构
		".btn-deliver",   // 投递按钮 - 旧版
	}

	var deliverBtn *rod.Element
	for _, selector := range selectors {
		logrus.Debugf("[DeliverJobFromSearchList] 尝试选择器: %s", selector)
		deliverBtn, err = card.Element(selector)
		if err == nil {
			logrus.Debugf("[DeliverJobFromSearchList] 找到按钮: %s", selector)
			break
		}
	}

	if deliverBtn == nil {
		logrus.Debugf("[DeliverJobFromSearchList] 未找到投递按钮，切换到详情页投递: %v", err)
		return d.DeliverJobFromDetail(jobID)
	}
	logrus.Debugf("[DeliverJobFromSearchList] 已找到投递按钮")

	logrus.Debugf("[DeliverJobFromSearchList] 点击投递按钮")
	err = deliverBtn.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		logrus.Errorf("[DeliverJobFromSearchList] 点击投递按钮失败: %v", err)
		return &DeliverResult{
			JobID:   jobID,
			Success: false,
			Message: "点击投递按钮失败",
		}, err
	}
	logrus.Debugf("[DeliverJobFromSearchList] 投递按钮已点击")

	// 等待投递
	logrus.Debugf("[DeliverJobFromSearchList] 等待投递结果 (2秒)")
	time.Sleep(2 * time.Second)

	// 检查结果
	logrus.Debugf("[DeliverJobFromSearchList] 检查投递结果")
	result, err := d.checkDeliverResult()
	if err != nil {
		logrus.Errorf("[DeliverJobFromSearchList] 检查结果失败: %v", err)
	}
	logrus.Debugf("[DeliverJobFromSearchList] ========== 投递完成 ==========")
	return result, err
}

// DeliverJobFromDetail 从详情页投递
func (d *Deliver) DeliverJobFromDetail(jobID string) (*DeliverResult, error) {
	logrus.Debugf("[DeliverJobFromDetail] ========== 从详情页投递 ==========")
	logrus.Debugf("[DeliverJobFromDetail] JobID: %s", jobID)

	// 访问详情页
	logrus.Debugf("[DeliverJobFromDetail] 访问详情页")
	url := "https://www.zhipin.com/job_detail/" + jobID + ".html"
	logrus.Debugf("[DeliverJobFromDetail] URL: %s", url)
	err := d.page.Navigate(url)
	if err != nil {
		logrus.Errorf("[DeliverJobFromDetail] 访问详情页失败: %v", err)
		return &DeliverResult{
			JobID:   jobID,
			Success: false,
			Message: "访问详情页失败",
		}, err
	}
	logrus.Debugf("[DeliverJobFromDetail] 页面导航成功，等待加载")

	d.page.WaitLoad()
	logrus.Debugf("[DeliverJobFromDetail] 页面加载完成")

	// 点击投递按钮
	logrus.Debugf("[DeliverJobFromDetail] 点击投递按钮")
	err = d.clickDeliverButton()
	if err != nil {
		logrus.Errorf("[DeliverJobFromDetail] 点击投递按钮失败: %v", err)
		return &DeliverResult{
			JobID:   jobID,
			Success: false,
			Message: "点击投递按钮失败: " + err.Error(),
		}, err
	}
	logrus.Debugf("[DeliverJobFromDetail] 投递按钮已点击")

	// 等待投递结果
	logrus.Debugf("[DeliverJobFromDetail] 等待投递结果 (2秒)")
	time.Sleep(2 * time.Second)

	// 检查投递结果
	logrus.Debugf("[DeliverJobFromDetail] 检查投递结果")
	result, err := d.checkDeliverResult()
	if err != nil {
		logrus.Errorf("[DeliverJobFromDetail] 检查结果失败: %v", err)
	}
	logrus.Debugf("[DeliverJobFromDetail] ========== 投递完成 ==========")
	return result, err
}

// clickDeliverButton 点击投递按钮
func (d *Deliver) clickDeliverButton() error {
	logrus.Debugf("[clickDeliverButton] 查找投递按钮")

	// 尝试多个投递按钮选择器
	selectors := []string{
		".btn-startchat",     // "立即沟通" 按钮 - 新版页面结构
		".btn-deliver",       // 投递按钮 - 旧版
		"button.btn-primary", // 主要按钮
	}

	var btn *rod.Element
	var err error

	for _, selector := range selectors {
		logrus.Debugf("[clickDeliverButton] 尝试选择器: %s", selector)
		btn, err = d.page.Element(selector)
		if err == nil {
			logrus.Debugf("[clickDeliverButton] 找到按钮: %s", selector)
			break
		}
		logrus.Debugf("[clickDeliverButton] 未找到: %s, 尝试下一个", selector)
	}

	if btn == nil {
		logrus.Error("[clickDeliverButton] 找不到投递按钮")
		return errors.New("找不到投递按钮")
	}

	logrus.Debugf("[clickDeliverButton] 执行点击")
	return btn.Click(proto.InputMouseButtonLeft, 1)
}

// checkDelivered 检查是否已投递
func (d *Deliver) checkDelivered(jobID string) (bool, error) {
	// 检查是否显示"已投递"
	html, _ := d.page.HTML()
	if strings.Contains(html, "已投递") || strings.Contains(html, "发送成功") {
		return true, nil
	}

	return false, nil
}

// checkDeliverResult 检查投递结果
func (d *Deliver) checkDeliverResult() (*DeliverResult, error) {
	result := &DeliverResult{}

	logrus.Debugf("[checkDeliverResult] 获取页面HTML")

	// 检查页面是否有成功提示
	html, err := d.page.HTML()
	if err != nil {
		logrus.Errorf("[checkDeliverResult] 获取HTML失败: %v", err)
		return result, err
	}
	logrus.Debugf("[checkDeliverResult] HTML长度: %d 字符", len(html))

	// 成功提示 - 检查多个关键词
	logrus.Debugf("[checkDeliverResult] 检查成功提示...")
	if strings.Contains(html, "投递成功") || strings.Contains(html, "发送成功") || strings.Contains(html, "已发送") {
		result.Success = true
		result.Message = "简历投递成功"
		logrus.Debugf("[checkDeliverResult] 匹配到成功提示")
		return result, nil
	}

	// 失败提示
	logrus.Debugf("[checkDeliverResult] 检查失败提示...")
	if strings.Contains(html, "投递失败") || strings.Contains(html, "发送失败") {
		result.Success = false
		result.Message = "简历投递失败"
		logrus.Debugf("[checkDeliverResult] 匹配到失败提示")
		return result, nil
	}

	// 重复投递
	logrus.Debugf("[checkDeliverResult] 检查重复投递...")
	if strings.Contains(html, "今日投递") || strings.Contains(html, "已投递") {
		result.Success = false
		result.Message = "该职位已投递"
		logrus.Debugf("[checkDeliverResult] 匹配到重复投递")
		return result, nil
	}

	// 默认认为成功（BOSS直聘有时候不会显示明确成功提示）
	logrus.Warn("[checkDeliverResult] 未匹配到明确提示，默认认为成功")
	result.Success = true
	result.Message = "简历已投递"

	return result, nil
}

// BatchDeliver 批量投递
func (d *Deliver) BatchDeliver(jobIDs []string) ([]DeliverResult, error) {
	results := []DeliverResult{}
	total := len(jobIDs)

	logrus.Debugf("[BatchDeliver] ========== 开始批量投递 ==========")
	logrus.Debugf("[BatchDeliver] 总数: %d", total)

	for i, jobID := range jobIDs {
		logrus.Infof("批量投递进度: %d/%d", i+1, total)
		logrus.Debugf("[BatchDeliver] 当前投递 JobID: %s", jobID)

		// 每次投递前延时
		if i > 0 {
			logrus.Debugf("[BatchDeliver] 投递间隔延时 (3秒)")
			time.Sleep(3 * time.Second)
		}

		result, err := d.DeliverJobFromSearchList(jobID)
		if err != nil {
			logrus.Errorf("[BatchDeliver] 投递异常: %v", err)
			result = &DeliverResult{
				JobID:   jobID,
				Success: false,
				Message: err.Error(),
			}
		}

		results = append(results, *result)

		// 投递成功或失败都继续
		if result.Success {
			logrus.Infof("[BatchDeliver] 投递成功: %s - %s", jobID, result.Message)
		} else {
			logrus.Warnf("[BatchDeliver] 投递失败: %s - %s", jobID, result.Message)
		}
	}

	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}
	logrus.Infof("[BatchDeliver] 批量投递完成: 成功 %d/%d", successCount, total)
	logrus.Debugf("[BatchDeliver] ========== 批量投递完成 ==========")

	return results, nil
}
