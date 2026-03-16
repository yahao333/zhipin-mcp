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
	// 先访问职位详情页
	detail := NewDetail(d.page)
	job, err := detail.GetJobDetail(ctx, jobID)
	if err != nil {
		return &DeliverResult{
			JobID:   jobID,
			Success: false,
			Message: "获取职位详情失败: " + err.Error(),
		}, err
	}

	// 点击投递按钮
	err = d.clickDeliverButton()
	if err != nil {
		return &DeliverResult{
			JobID:   jobID,
			Success: false,
			Message: "点击投递按钮失败: " + err.Error(),
		}, err
	}

	// 等待投递结果
	time.Sleep(2 * time.Second)

	// 检查投递结果
	result, err := d.checkDeliverResult()
	if err != nil {
		return &DeliverResult{
			JobID:   jobID,
			Success: false,
			Message: "检查投递结果失败: " + err.Error(),
		}, err
	}

	if result.Success {
		logrus.Infof("职位投递成功: %s - %s", jobID, job.Title)
	} else {
		logrus.Warnf("职位投递失败: %s - %s, %s", jobID, job.Title, result.Message)
	}

	return result, nil
}

// DeliverJobFromSearchList 从搜索列表投递
func (d *Deliver) DeliverJobFromSearchList(jobID string) (*DeliverResult, error) {
	// 查找职位卡片
	selector := "[data-jobid='" + jobID + "']"
	card, err := d.page.Element(selector)
	if err != nil {
		return d.DeliverJobFromDetail(jobID)
	}

	// 点击投递按钮
	deliverBtn, err := card.Element(".btn-deliver")
	if err != nil {
		// 尝试点击整个卡片
		return d.DeliverJobFromDetail(jobID)
	}

	err = deliverBtn.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return &DeliverResult{
			JobID:   jobID,
			Success: false,
			Message: "点击投递按钮失败",
		}, err
	}

	// 等待投递
	time.Sleep(2 * time.Second)

	// 检查结果
	return d.checkDeliverResult()
}

// DeliverJobFromDetail 从详情页投递
func (d *Deliver) DeliverJobFromDetail(jobID string) (*DeliverResult, error) {
	// 访问详情页
	url := "https://www.zhipin.com/job_detail/" + jobID + ".html"
	err := d.page.Navigate(url)
	if err != nil {
		return &DeliverResult{
			JobID:   jobID,
			Success: false,
			Message: "访问详情页失败",
		}, err
	}

	d.page.WaitLoad()

	// 点击投递按钮
	err = d.clickDeliverButton()
	if err != nil {
		return &DeliverResult{
			JobID:   jobID,
			Success: false,
			Message: "点击投递按钮失败: " + err.Error(),
		}, err
	}

	// 等待投递结果
	time.Sleep(2 * time.Second)

	// 检查投递结果
	return d.checkDeliverResult()
}

// clickDeliverButton 点击投递按钮
func (d *Deliver) clickDeliverButton() error {
	// 查找投递按钮
	btn, err := d.page.Element(".btn-deliver")
	if err != nil {
		// 尝试其他选择器
		btn, err = d.page.Element("button.btn-primary")
		if err != nil {
			return errors.New("找不到投递按钮")
		}
	}

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

	// 检查页面是否有成功提示
	html, err := d.page.HTML()
	if err != nil {
		return result, err
	}

	// 成功提示
	if strings.Contains(html, "投递成功") || strings.Contains(html, "发送成功") || strings.Contains(html, "已发送") {
		result.Success = true
		result.Message = "简历投递成功"
		return result, nil
	}

	// 失败提示
	if strings.Contains(html, "投递失败") || strings.Contains(html, "发送失败") {
		result.Success = false
		result.Message = "简历投递失败"
		return result, nil
	}

	// 重复投递
	if strings.Contains(html, "今日投递") || strings.Contains(html, "已投递") {
		result.Success = false
		result.Message = "该职位已投递"
		return result, nil
	}

	// 默认认为成功（BOSS直聘有时候不会显示明确成功提示）
	result.Success = true
	result.Message = "简历已投递"

	return result, nil
}

// BatchDeliver 批量投递
func (d *Deliver) BatchDeliver(jobIDs []string) ([]DeliverResult, error) {
	results := []DeliverResult{}

	for i, jobID := range jobIDs {
		logrus.Infof("批量投递进度: %d/%d", i+1, len(jobIDs))

		// 每次投递前延时
		if i > 0 {
			time.Sleep(3 * time.Second)
		}

		result, err := d.DeliverJobFromSearchList(jobID)
		if err != nil {
			result = &DeliverResult{
				JobID:   jobID,
				Success: false,
				Message: err.Error(),
			}
		}

		results = append(results, *result)

		// 投递成功或失败都继续
		logrus.Infof("投递结果: %s - %s", jobID, result.Message)
	}

	return results, nil
}
