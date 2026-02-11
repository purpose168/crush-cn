package common

import (
	"strings"

	"github.com/charmbracelet/crush/internal/ui/styles"
)

// Scrollbar 根据内容和视口大小渲染垂直滚动条。
// 如果内容适合视口（无需滚动），则返回空字符串。
func Scrollbar(s *styles.Styles, height, contentSize, viewportSize, offset int) string {
	if height <= 0 || contentSize <= viewportSize {
		return ""
	}

	// 计算滑块大小（最小 1 个字符）。
	thumbSize := max(1, height*viewportSize/contentSize)

	// 计算滑块位置。
	maxOffset := contentSize - viewportSize
	if maxOffset <= 0 {
		return ""
	}

	// 计算滑块开始的位置。
	trackSpace := height - thumbSize
	thumbPos := 0
	if trackSpace > 0 && maxOffset > 0 {
		thumbPos = min(trackSpace, offset*trackSpace/maxOffset)
	}

	// 构建滚动条。
	var sb strings.Builder
	for i := range height {
		if i > 0 {
			sb.WriteString("\n")
		}
		if i >= thumbPos && i < thumbPos+thumbSize {
			sb.WriteString(s.Dialog.ScrollbarThumb.Render(styles.ScrollbarThumb))
		} else {
			sb.WriteString(s.Dialog.ScrollbarTrack.Render(styles.ScrollbarTrack))
		}
	}

	return sb.String()
}
