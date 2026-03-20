package zhipin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExtractJobIDFromHref 测试从 href 中提取 jobID
func TestExtractJobIDFromHref(t *testing.T) {
	tests := []struct {
		name     string
		href     string
		expected string
	}{
		{
			name:     "正常的job详情URL",
			href:     "/job_detail/b7514bae474aa1ce0nZ72tq9GFZY.html",
			expected: "b7514bae474aa1ce0nZ72tq9GFZY",
		},
		{
			name:     "带查询参数的URL",
			href:     "/job_detail/abc123xyz.html?source=search&query=go",
			expected: "abc123xyz",
		},
		{
			name:     "空字符串",
			href:     "",
			expected: "",
		},
		{
			name:     "无效的URL格式",
			href:     "/search/jobs?keyword=go",
			expected: "",
		},
		{
			name:     "不包含job_detail的URL",
			href:     "/user/profile/123456",
			expected: "",
		},
		{
			name:     "带其他前缀的job_detail",
			href:     "/web/job_detail/xyz789.html",
			expected: "xyz789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJobIDFromHref(tt.href)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExtractJobIDFromHrefWithRealURLs 测试实际的BOSS直聘URL
func TestExtractJobIDFromHrefWithRealURLs(t *testing.T) {
	tests := []struct {
		name     string
		href     string
		expected string
	}{
		{
			name:     "真实URL示例1",
			href:     "/job_detail/73e29d7ca568b6781nZ-2tS9GFZY.html",
			expected: "73e29d7ca568b6781nZ-2tS9GFZY",
		},
		{
			name:     "真实URL示例2",
			href:     "/job_detail/b9a97c3a6866e6be1nZ_2tS9GFJY.html",
			expected: "b9a97c3a6866e6be1nZ_2tS9GFJY",
		},
		{
			name:     "绝对URL也能提取",
			href:     "https://www.zhipin.com/job_detail/abc123.html",
			expected: "abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJobIDFromHref(tt.href)
			assert.Equal(t, tt.expected, result)
		})
	}
}
