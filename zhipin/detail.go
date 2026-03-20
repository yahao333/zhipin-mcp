package zhipin

import (
	"context"
	"time"

	"github.com/go-rod/rod"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
	logrus.Debugf("[Detail.GetJobDetail] ========== 开始获取职位详情 ==========")
	logrus.Debugf("[Detail.GetJobDetail] 接收到的 jobID: %s", jobID)

	// 访问职位详情页
	url := "https://www.zhipin.com/job_detail/" + jobID + ".html"
	logrus.Debugf("[Detail.GetJobDetail] 准备导航到 URL: %s", url)
	err := d.page.Navigate(url)
	if err != nil {
		logrus.Errorf("[Detail.GetJobDetail] Navigate 失败: %v", err)
		return nil, errors.Wrap(err, "访问详情页失败")
	}
	logrus.Debugf("[Detail.GetJobDetail] Navigate 成功")

	// 等待页面加载
	logrus.Debugf("[Detail.GetJobDetail] 等待页面加载...")
	d.page.WaitLoad()
	logrus.Debugf("[Detail.GetJobDetail] 页面加载完成")

	// 解析详情
	logrus.Debugf("[Detail.GetJobDetail] 开始解析详情...")
	job, err := d.parseJobDetail(jobID)
	if err != nil {
		logrus.Errorf("[Detail.GetJobDetail] parseJobDetail 失败: %v", err)
		return nil, err
	}

	logrus.Debugf("[Detail.GetJobDetail] 解析完成, job=%+v", job)
	logrus.Debugf("[Detail.GetJobDetail] ========== 获取职位详情完成 ==========")

	return job, nil
}

// parseJobDetail 解析职位详情
func (d *Detail) parseJobDetail(jobID string) (*Job, error) {
	logrus.Debugf("[Detail.parseJobDetail] ========== 开始解析职位详情 ==========")
	logrus.Debugf("[Detail.parseJobDetail] 解析 jobID: %s", jobID)

	job := &Job{
		ID:        jobID,
		UpdatedAt: time.Now(),
	}

	// 获取职位标题
	logrus.Debugf("[Detail.parseJobDetail] 查找职位标题 .job-title")
	titleEl, err := d.page.Element(".job-title")
	if err == nil {
		job.Title, _ = titleEl.Text()
		logrus.Debugf("[Detail.parseJobDetail] 职位标题: %s", job.Title)
	} else {
		logrus.Warnf("[Detail.parseJobDetail] 未找到职位标题元素: %v", err)
	}

	// 获取薪资
	logrus.Debugf("[Detail.parseJobDetail] 查找薪资 .salary")
	salaryEl, err := d.page.Element(".salary")
	if err == nil {
		job.SalaryRange, _ = salaryEl.Text()
		logrus.Debugf("[Detail.parseJobDetail] 薪资范围: %s", job.SalaryRange)
	} else {
		logrus.Warnf("[Detail.parseJobDetail] 未找到薪资元素: %v", err)
	}

	// 获取公司名称
	logrus.Debugf("[Detail.parseJobDetail] 查找公司名称 .company-name a")
	companyEl, err := d.page.Element(".company-name a")
	if err == nil {
		job.CompanyName, _ = companyEl.Text()
		logrus.Debugf("[Detail.parseJobDetail] 公司名称: %s", job.CompanyName)
	} else {
		logrus.Warnf("[Detail.parseJobDetail] 未找到公司名称元素: %v", err)
	}

	// 获取HR信息
	logrus.Debugf("[Detail.parseJobDetail] 查找HR信息 .hr-name")
	hrNameEl, err := d.page.Element(".hr-name")
	if err == nil {
		job.HRName, _ = hrNameEl.Text()
		logrus.Debugf("[Detail.parseJobDetail] HR名称: %s", job.HRName)
	} else {
		logrus.Warnf("[Detail.parseJobDetail] 未找到HR名称元素: %v", err)
	}

	logrus.Debugf("[Detail.parseJobDetail] 解析完成, 最终职位信息: %+v", job)
	logrus.Debugf("[Detail.parseJobDetail] ========== 解析职位详情完成 ==========")

	return job, nil
}
