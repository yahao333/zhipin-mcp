package delay

import (
	"math/rand"
	"time"

	"github.com/yahao333/zhipin-mcp/configs"
)

// Random 随机延时，使用配置文件中的 min_delay 和 max_delay
// 默认范围：3-8秒
func Random() {
	minDelay := configs.MinDelay
	maxDelay := configs.MaxDelay

	if minDelay <= 0 {
		minDelay = 3000
	}
	if maxDelay <= 0 {
		maxDelay = 8000
	}

	// 确保 minDelay <= maxDelay
	if minDelay > maxDelay {
		minDelay = maxDelay
	}

	delayMs := minDelay + rand.Intn(maxDelay-minDelay+1)
	time.Sleep(time.Duration(delayMs) * time.Millisecond)
}

// RandomWithRange 自定义延时范围
func RandomWithRange(minMs, maxMs int) {
	if minMs <= 0 {
		minMs = 1000
	}
	if maxMs <= 0 {
		maxMs = 3000
	}
	if minMs > maxMs {
		minMs = maxMs
	}

	delayMs := minMs + rand.Intn(maxMs-minMs+1)
	time.Sleep(time.Duration(delayMs) * time.Millisecond)
}

// Fixed 固定延时
func Fixed(ms int) {
	if ms <= 0 {
		ms = 1000
	}
	time.Sleep(time.Duration(ms) * time.Millisecond)
}
