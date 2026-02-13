// Package diffview 提供差异视图渲染功能
package diffview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// pad 将值左填充空格到指定宽度，用于对齐行号显示
func pad(v any, width int) string {
	s := fmt.Sprintf("%v", v)
	w := ansi.StringWidth(s)
	if w >= width {
		return s
	}
	return strings.Repeat(" ", width-w) + s
}

// isEven 判断整数是否为偶数
func isEven(n int) bool {
	return n%2 == 0
}

// isOdd 判断整数是否为奇数
func isOdd(n int) bool {
	return !isEven(n)
}

// btoi 将布尔值转换为整数，true 返回 1，false 返回 0
func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ternary 实现三元运算符功能，根据条件返回两个值中的一个
func ternary[T any](cond bool, t, f T) T {
	if cond {
		return t
	}
	return f
}
