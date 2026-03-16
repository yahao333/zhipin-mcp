package zhipin

import (
	"context"
	"time"

	"github.com/go-rod/rod"
	"github.com/pkg/errors"
)

// Detail 职位详情操作
type Detail struct {
	page *rod.Page
}

// NewDetail 创建详情操作
func NewDetail(page *rod.Page) *Detail {
	return &Detail{page: page}
}

// GetJobDetail 获取职位详情
func (d *Detail) GetJobDetail(ctx context.Context, jobID string) (*Job, error) {
	// 访问职位详情页
	url := "https://www.zhipin.com/job_detail/" + jobID + ".html"
	err := d.page.Navigate(url)
	if err != nil {
		return nil, errors.Wrap(err, "访问详情页失败")
	}

	// 等待页面加载
	d.page.WaitLoad()

	// 解析详情
	job, err := d.parseJobDetail(jobID)
	if err != nil {
		return nil, err
	}

	return job, nil
}

// parseJobDetail 解析职位详情
func (d *Detail) parseJobDetail(jobID string) (*Job, error) {
	job := &Job{
		ID:        jobID,
		UpdatedAt: time.Now(),
	}

	// 获取职位标题
	titleEl, err := d.page.Element(".job-title")
	if err == nil {
		job.Title, _ = titleEl.Text()
	}

	// 获取薪资
	salaryEl, err := d.page.Element(".salary")
	if err == nil {
		job.SalaryRange, _ = salaryEl.Text()
	}

	// 获取公司名称
	companyEl, err := d.page.Element(".company-name a")
	if err == nil {
		job.CompanyName, _ = companyEl.Text()
	}

	// 获取HR信息
	hrNameEl, err := d.page.Element(".hr-name")
	if err == nil {
		job.HRName, _ = hrNameEl.Text()
	}

	return job, nil
}
