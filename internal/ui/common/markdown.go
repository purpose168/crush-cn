package common

import (
	"charm.land/glamour/v2"
	"github.com/purpose168/crush-cn/internal/ui/styles"
)

// MarkdownRenderer 返回一个使用给定样式和宽度配置的 glamour [glamour.TermRenderer]。
func MarkdownRenderer(sty *styles.Styles, width int) *glamour.TermRenderer {
	r, _ := glamour.NewTermRenderer(
		glamour.WithStyles(sty.Markdown),
		glamour.WithWordWrap(width),
	)
	return r
}

// PlainMarkdownRenderer 返回一个没有颜色的 glamour [glamour.TermRenderer]
// （带有结构的纯文本）和给定的宽度。
func PlainMarkdownRenderer(sty *styles.Styles, width int) *glamour.TermRenderer {
	r, _ := glamour.NewTermRenderer(
		glamour.WithStyles(sty.PlainMarkdown),
		glamour.WithWordWrap(width),
	)
	return r
}
