package dialog

import (
	"image/color"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/purpose168/crush-cn/internal/ui/common"
	"github.com/purpose168/crush-cn/internal/ui/styles"
)

// InputCursor 调整对话框内输入字段的光标位置。
func InputCursor(t *styles.Styles, cur *tea.Cursor) *tea.Cursor {
	if cur != nil {
		titleStyle := t.Dialog.Title
		dialogStyle := t.Dialog.View
		inputStyle := t.Dialog.InputPrompt
		// 调整光标位置以考虑对话框布局
		cur.X += inputStyle.GetBorderLeftSize() +
			inputStyle.GetMarginLeft() +
			inputStyle.GetPaddingLeft() +
			dialogStyle.GetBorderLeftSize() +
			dialogStyle.GetPaddingLeft() +
			dialogStyle.GetMarginLeft()
		cur.Y += titleStyle.GetVerticalFrameSize() +
			inputStyle.GetBorderTopSize() +
			inputStyle.GetMarginTop() +
			inputStyle.GetPaddingTop() +
			inputStyle.GetBorderBottomSize() +
			inputStyle.GetMarginBottom() +
			inputStyle.GetPaddingBottom() +
			dialogStyle.GetPaddingTop() +
			dialogStyle.GetMarginTop() +
			dialogStyle.GetBorderTopSize()
	}
	return cur
}

// RenderContext 是一个对话框渲染上下文，可用于渲染
// 常见的对话框布局。
type RenderContext struct {
	// Styles 是用于渲染的样式。
	Styles *styles.Styles
	// TitleStyle 是对话框标题的样式，默认使用 Styles.Dialog.Title
	TitleStyle lipgloss.Style
	// ViewStyle 是对话框标题的样式，默认使用 Styles.Dialog.View
	ViewStyle lipgloss.Style
	// TitleGradientFromColor 是标题渐变开始的颜色，默认为 Style.Primary
	TitleGradientFromColor color.Color
	// TitleGradientToColor 是标题渐变开始的颜色，默认为 Style.Secondary
	TitleGradientToColor color.Color
	// Width 是对话框的总宽度，包括任何边距、边框和内边距。
	Width int
	// Gap 是内容部分之间的间隙。零表示没有间隙。
	Gap int
	// Title 是对话框的标题。这将使用默认对话框标题样式进行样式化，
	// 并添加到内容部分切片的前面。
	Title string
	// TitleInfo 是在标题旁边显示的附加信息。此部分按原样显示，
	// 任何样式必须在设置此字段之前应用。
	TitleInfo string
	// Parts 是对话框的渲染部分。
	Parts []string
	// Help 是帮助视图内容。这将使用默认对话框帮助样式
	// 添加到内容部分切片。
	Help string
	// IsOnboarding 指示是否作为引导流程的一部分渲染对话框。
	// 这意味着内容将在屏幕的左下角渲染。
	IsOnboarding bool
}

// NewRenderContext 使用提供的样式和宽度创建一个新的 RenderContext。
func NewRenderContext(t *styles.Styles, width int) *RenderContext {
	return &RenderContext{
		Styles:                 t,
		TitleStyle:             t.Dialog.Title,
		ViewStyle:              t.Dialog.View,
		TitleGradientFromColor: t.Primary,
		TitleGradientToColor:   t.Secondary,
		Width:                  width,
		Parts:                  []string{},
	}
}

// AddPart 向对话框添加一个渲染部分。
func (rc *RenderContext) AddPart(part string) {
	if len(part) > 0 {
		rc.Parts = append(rc.Parts, part)
	}
}

// Render 使用提供的上下文渲染对话框。
func (rc *RenderContext) Render() string {
	titleStyle := rc.TitleStyle
	dialogStyle := rc.ViewStyle.Width(rc.Width)

	var parts []string

	if len(rc.Title) > 0 {
		var titleInfoWidth int
		if len(rc.TitleInfo) > 0 {
			titleInfoWidth = lipgloss.Width(rc.TitleInfo)
		}
		title := common.DialogTitle(rc.Styles, rc.Title,
			max(0, rc.Width-dialogStyle.GetHorizontalFrameSize()-
				titleStyle.GetHorizontalFrameSize()-
				titleInfoWidth), rc.TitleGradientFromColor, rc.TitleGradientToColor)
		if len(rc.TitleInfo) > 0 {
			title += rc.TitleInfo
		}
		parts = append(parts, titleStyle.Render(title))
		if rc.Gap > 0 {
			parts = append(parts, make([]string, rc.Gap)...)
		}
	}

	if rc.Gap <= 0 {
		parts = append(parts, rc.Parts...)
	} else {
		for i, p := range rc.Parts {
			if len(p) > 0 {
				parts = append(parts, p)
			}
			if i < len(rc.Parts)-1 {
				parts = append(parts, make([]string, rc.Gap)...)
			}
		}
	}

	if len(rc.Help) > 0 {
		if rc.Gap > 0 {
			parts = append(parts, make([]string, rc.Gap)...)
		}
		helpWidth := rc.Width - dialogStyle.GetHorizontalFrameSize()
		helpStyle := rc.Styles.Dialog.HelpView
		helpStyle = helpStyle.Width(helpWidth)
		helpView := ansi.Truncate(helpStyle.Render(rc.Help), helpWidth-1, "")
		parts = append(parts, helpView)
	}

	content := strings.Join(parts, "\n")
	if rc.IsOnboarding {
		return content
	}
	return dialogStyle.Render(content)
}
