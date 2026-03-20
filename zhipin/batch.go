package zhipin

import (
	"context"
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yahao333/zhipin-mcp/configs"
)

// Batch 批量投递操作
type Batch struct {
	deliver *Deliver
}

// NewBatch 创建批量投递操作
func NewBatch(deliver *Deliver) *Batch {
	return &Batch{deliver: deliver}
}

// DeliverJobs 批量投递职位
func (b *Batch) DeliverJobs(ctx context.Context, jobs []Job) ([]DeliverResult, error) {
	results := []DeliverResult{}

	for i, job := range jobs {
		logrus.Infof("批量投递进度: %d/%d - %s", i+1, len(jobs), job.Title)

		// 随机延时
		b.randomDelay()

		// 投递
		result, err := b.deliver.DeliverJob(ctx, job.ID)
		if err != nil {
			logrus.Warnf("投递失败: %s - %v", job.Title, err)
			result.Message = err.Error()
		}

		results = append(results, *result)

		// 投递成功后延时
		if result.Success {
			b.randomDelay()
		}
	}

	return results, nil
}

// randomDelay 随机延时
func (b *Batch) randomDelay() {
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
