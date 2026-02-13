package diffview

import (
	"testing"
)

// TestPad 测试 pad 函数的填充功能
func TestPad(t *testing.T) {
	tests := []struct {
		input    any
		width    int
		expected string
	}{
		{7, 2, " 7"},
		{7, 3, "  7"},
		{"a", 2, " a"},
		{"a", 3, "  a"},
		{"…", 2, " …"},
		{"…", 3, "  …"},
	}

	for _, tt := range tests {
		result := pad(tt.input, tt.width)
		if result != tt.expected {
			t.Errorf("期望 %q，实际为 %q", tt.expected, result)
		}
	}
}
