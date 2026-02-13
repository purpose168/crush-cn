package ansiext

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Escape 将控制字符替换为其 Unicode 控制图片表示形式，以确保它们在 UI 中正确显示。
func Escape(content string) string {
	var sb strings.Builder
	sb.Grow(len(content))
	for _, r := range content {
		switch {
		case r >= 0 && r <= 0x1f: // Control characters 0x00-0x1F
			sb.WriteRune('\u2400' + r)
		case r == ansi.DEL:
			sb.WriteRune('\u2421')
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
