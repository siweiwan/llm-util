package util

import (
	"fmt"
	"testing"
)

func TestGetFirstXChars(t *testing.T) {
	tests := []struct {
		input    string
		x        int
		expected string
	}{
		{"你好，世界！", 3, "你好，"},
		{"你好，世界！", 5, "你好，世界"},
		{"Go语言", 2, "Go"},
		{"单字节", 4, "单字节"},
		{"测试", 0, ""}, // 0 个字符
		{"", 3, ""},     // 空字符串
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("input: %s, x: %d", test.input, test.x), func(t *testing.T) {
			result := GetFirstXChars(test.input, test.x)
			if result != test.expected {
				t.Errorf("expected %s, got %s", test.expected, result)
			}
		})
	}
}
