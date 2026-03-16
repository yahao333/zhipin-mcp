package browser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestWithBinPath 测试BinPath选项
func TestWithBinPath(t *testing.T) {
	opt := WithBinPath("/custom/path/chrome")

	cfg := &browserConfig{}
	opt(cfg)

	assert.Equal(t, "/custom/path/chrome", cfg.binPath)
}

// TestBrowserConfig_Structure 测试浏览器配置结构
func TestBrowserConfig_Structure(t *testing.T) {
	cfg := &browserConfig{
		binPath: "/test/path",
	}

	assert.Equal(t, "/test/path", cfg.binPath)
}

// TestOptionFunc_Signature 测试选项函数签名
func TestOptionFunc_Signature(t *testing.T) {
	// 测试选项函数类型正确
	opt := WithBinPath("/test/path")

	cfg := &browserConfig{}
	opt(cfg)

	assert.Equal(t, "/test/path", cfg.binPath)
}

// TestWithBinPathEmpty 测试空BinPath
func TestWithBinPathEmpty(t *testing.T) {
	opt := WithBinPath("")

	cfg := &browserConfig{}
	opt(cfg)

	assert.Equal(t, "", cfg.binPath)
}

// TestMultipleOptions 测试多个选项
func TestMultipleOptions(t *testing.T) {
	opts := []Option{
		WithBinPath("/path/bin"),
	}

	cfg := &browserConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	assert.Equal(t, "/path/bin", cfg.binPath)
}
