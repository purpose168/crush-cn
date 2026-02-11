package common

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/ui/styles"
)

// ButtonOpts 定义单个按钮的配置
type ButtonOpts struct {
	// Text 是按钮标签
	Text string
	// UnderlineIndex 是要下划线字符的从 0 开始的索引（-1 表示无）
	UnderlineIndex int
	// Selected 指示此按钮当前是否被选中
	Selected bool
	// Padding 内部水平内边距，如果为 0 则默认为 2
	Padding int
}

// Button 创建一个带有下划线字符和选择状态的按钮
func Button(t *styles.Styles, opts ButtonOpts) string {
	// 根据选择状态选择样式
	style := t.ButtonBlur
	if opts.Selected {
		style = t.ButtonFocus
	}

	text := opts.Text
	if opts.Padding == 0 {
		opts.Padding = 2
	}

	// 索引超出范围
	if opts.UnderlineIndex > -1 && opts.UnderlineIndex > len(text)-1 {
		opts.UnderlineIndex = -1
	}

	text = style.Padding(0, opts.Padding).Render(text)

	if opts.UnderlineIndex != -1 {
		text = lipgloss.StyleRanges(text, lipgloss.NewRange(opts.Padding+opts.UnderlineIndex, opts.Padding+opts.UnderlineIndex+1, style.Underline(true)))
	}

	return text
}

// ButtonGroup 创建一行可选择的按钮
// Spacing 是按钮之间的分隔符
// 使用 " " 或类似字符进行水平布局
// 使用 "\n" 进行垂直布局
// 默认为 "  "（水平）
func ButtonGroup(t *styles.Styles, buttons []ButtonOpts, spacing string) string {
	if len(buttons) == 0 {
		return ""
	}

	if spacing == "" {
		spacing = "  "
	}

	parts := make([]string, len(buttons))
	for i, button := range buttons {
		parts[i] = Button(t, button)
	}

	return strings.Join(parts, spacing)
}
