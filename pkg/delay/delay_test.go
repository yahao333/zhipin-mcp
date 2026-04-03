package delay

import (
	"testing"
	"time"

	"github.com/yahao333/zhipin-mcp/configs"
)

func TestRandom(t *testing.T) {
	// 保存原始配置
	origMinDelay := configs.MinDelay
	origMaxDelay := configs.MaxDelay

	// 设置测试值
	configs.MinDelay = 10
	configs.MaxDelay = 20

	// 恢复原始值
	defer func() {
		configs.MinDelay = origMinDelay
		configs.MaxDelay = origMaxDelay
	}()

	// 测试多次调用，确保在范围内
	for i := 0; i < 10; i++ {
		start := time.Now()
		Random()
		elapsed := time.Since(start).Milliseconds()

		if elapsed < 10 || elapsed > 25 { // 允许一些误差
			t.Errorf("Random() 延时超出范围: got %dms, want between 10-20ms", elapsed)
		}
	}
}

func TestRandomWithRange(t *testing.T) {
	minMs := 100
	maxMs := 200

	start := time.Now()
	RandomWithRange(minMs, maxMs)
	elapsed := int(time.Since(start).Milliseconds())

	if elapsed < minMs || elapsed > maxMs+10 { // 允许一些误差
		t.Errorf("RandomWithRange() 延时超出范围: got %dms, want between %d-%dms", elapsed, minMs, maxMs)
	}
}

func TestFixed(t *testing.T) {
	ms := 100

	start := time.Now()
	Fixed(ms)
	elapsed := int(time.Since(start).Milliseconds())

	if elapsed < ms || elapsed > ms+10 { // 允许一些误差
		t.Errorf("Fixed() 延时不准确: got %dms, want %dms", elapsed, ms)
	}
}

func TestRandomWithInvalidRange(t *testing.T) {
	// 测试 min > max 的情况
	start := time.Now()
	RandomWithRange(500, 100) // min > max
	elapsed := int(time.Since(start).Milliseconds())

	// 应该使用 max 作为延时
	if elapsed < 100 || elapsed > 110 {
		t.Errorf("RandomWithRange() 处理无效范围错误: got %dms", elapsed)
	}
}
