package zhipin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMinFunction(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{"a < b", 1, 2, 1},
		{"a > b", 5, 3, 3},
		{"a == b", 4, 4, 4},
		{"zero values", 0, 0, 0},
		{"negative numbers", -5, -3, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := min(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSendResult(t *testing.T) {
	tests := []struct {
		name     string
		result   SendResult
		expected SendResult
	}{
		{
			name: "成功发送",
			result: SendResult{
				Success:    true,
				PersonName: "张三",
				Message:    "消息发送成功",
			},
			expected: SendResult{
				Success:    true,
				PersonName: "张三",
				Message:    "消息发送成功",
			},
		},
		{
			name: "发送失败",
			result: SendResult{
				Success:    false,
				PersonName: "李四",
				Message:    "未找到联系人",
			},
			expected: SendResult{
				Success:    false,
				PersonName: "李四",
				Message:    "未找到联系人",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected.Success, tt.result.Success)
			assert.Equal(t, tt.expected.PersonName, tt.result.PersonName)
			assert.Equal(t, tt.expected.Message, tt.result.Message)
		})
	}
}
