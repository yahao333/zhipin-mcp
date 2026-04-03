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
