package common

import (
	"cmp"
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/home"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/charmbracelet/x/ansi"
)

// PrettyPath 格式化文件路径，使用主目录缩写并应用静音样式。
func PrettyPath(t *styles.Styles, path string, width int) string {
	formatted := home.Short(path)
	return t.Muted.Width(width).Render(formatted)
}

// ModelContextInfo 包含模型的令牌使用和成本信息。
type ModelContextInfo struct {
	ContextUsed  int64
	ModelContext int64
	Cost         float64
}

// ModelInfo 渲染模型信息，包括名称、提供商、推理设置和可选的上下文使用/成本。
func ModelInfo(t *styles.Styles, modelName, providerName, reasoningInfo string, context *ModelContextInfo, width int) string {
	modelIcon := t.Subtle.Render(styles.ModelIcon)
	modelName = t.Base.Render(modelName)

	// 构建第一行，包含模型名称和可选的提供商在同一行
	var firstLine string
	if providerName != "" {
		providerInfo := t.Muted.Render(fmt.Sprintf("via %s", providerName))
		modelWithProvider := fmt.Sprintf("%s %s %s", modelIcon, modelName, providerInfo)

		// 检查是否适合一行
		if lipgloss.Width(modelWithProvider) <= width {
			firstLine = modelWithProvider
		} else {
			// 如果不适合，将提供商放在下一行
			firstLine = fmt.Sprintf("%s %s", modelIcon, modelName)
		}
	} else {
		firstLine = fmt.Sprintf("%s %s", modelIcon, modelName)
	}

	parts := []string{firstLine}

	// 如果提供商不适合第一行，将其作为第二行添加
	if providerName != "" && !strings.Contains(firstLine, "via") {
		providerInfo := fmt.Sprintf("via %s", providerName)
		parts = append(parts, t.Muted.PaddingLeft(2).Render(providerInfo))
	}

	if reasoningInfo != "" {
		parts = append(parts, t.Subtle.PaddingLeft(2).Render(reasoningInfo))
	}

	if context != nil {
		formattedInfo := formatTokensAndCost(t, context.ContextUsed, context.ModelContext, context.Cost)
		parts = append(parts, lipgloss.NewStyle().PaddingLeft(2).Render(formattedInfo))
	}

	return lipgloss.NewStyle().Width(width).Render(
		lipgloss.JoinVertical(lipgloss.Left, parts...),
	)
}

// formatTokensAndCost 格式化令牌使用和成本，使用适当的单位（K/M）和上下文窗口的百分比。
func formatTokensAndCost(t *styles.Styles, tokens, contextWindow int64, cost float64) string {
	var formattedTokens string
	switch {
	case tokens >= 1_000_000:
		formattedTokens = fmt.Sprintf("%.1fM", float64(tokens)/1_000_000)
	case tokens >= 1_000:
		formattedTokens = fmt.Sprintf("%.1fK", float64(tokens)/1_000)
	default:
		formattedTokens = fmt.Sprintf("%d", tokens)
	}

	if strings.HasSuffix(formattedTokens, ".0K") {
		formattedTokens = strings.Replace(formattedTokens, ".0K", "K", 1)
	}
	if strings.HasSuffix(formattedTokens, ".0M") {
		formattedTokens = strings.Replace(formattedTokens, ".0M", "M", 1)
	}

	percentage := (float64(tokens) / float64(contextWindow)) * 100

	formattedCost := t.Muted.Render(fmt.Sprintf("$%.2f", cost))

	formattedTokens = t.Subtle.Render(fmt.Sprintf("(%s)", formattedTokens))
	formattedPercentage := t.Muted.Render(fmt.Sprintf("%d%%", int(percentage)))
	formattedTokens = fmt.Sprintf("%s %s", formattedPercentage, formattedTokens)
	if percentage > 80 {
		formattedTokens = fmt.Sprintf("%s %s", styles.LSPWarningIcon, formattedTokens)
	}

	return fmt.Sprintf("%s %s", formattedTokens, formattedCost)
}

// StatusOpts 定义渲染状态行的选项，包括图标、标题、描述和可选的额外内容。
type StatusOpts struct {
	Icon             string // 如果为空，则不显示图标
	Title            string
	TitleColor       color.Color
	Description      string
	DescriptionColor color.Color
	ExtraContent     string // 在描述后追加的额外内容
}

// Status 渲染一个状态行，包括图标、标题、描述和额外内容。如果描述超过可用宽度，则截断。
func Status(t *styles.Styles, opts StatusOpts, width int) string {
	icon := opts.Icon
	title := opts.Title
	description := opts.Description

	titleColor := cmp.Or(opts.TitleColor, t.Muted.GetForeground())
	descriptionColor := cmp.Or(opts.DescriptionColor, t.Subtle.GetForeground())

	title = t.Base.Foreground(titleColor).Render(title)

	if description != "" {
		extraContentWidth := lipgloss.Width(opts.ExtraContent)
		if extraContentWidth > 0 {
			extraContentWidth += 1
		}
		description = ansi.Truncate(description, width-lipgloss.Width(icon)-lipgloss.Width(title)-2-extraContentWidth, "…")
		description = t.Base.Foreground(descriptionColor).Render(description)
	}

	var content []string
	if icon != "" {
		content = append(content, icon)
	}
	content = append(content, title)
	if description != "" {
		content = append(content, description)
	}
	if opts.ExtraContent != "" {
		content = append(content, opts.ExtraContent)
	}

	return strings.Join(content, " ")
}

// Section 渲染一个节标题，带有标题和填充剩余宽度的水平线。
func Section(t *styles.Styles, text string, width int, info ...string) string {
	char := styles.SectionSeparator
	length := lipgloss.Width(text) + 1
	remainingWidth := width - length

	var infoText string
	if len(info) > 0 {
		infoText = strings.Join(info, " ")
		if len(infoText) > 0 {
			infoText = " " + infoText
			remainingWidth -= lipgloss.Width(infoText)
		}
	}

	text = t.Section.Title.Render(text)
	if remainingWidth > 0 {
		text = text + " " + t.Section.Line.Render(strings.Repeat(char, remainingWidth)) + infoText
	}
	return text
}

// DialogTitle 渲染一个对话框标题，带有填充剩余宽度的装饰线。
func DialogTitle(t *styles.Styles, title string, width int, fromColor, toColor color.Color) string {
	char := "╱"
	length := lipgloss.Width(title) + 1
	remainingWidth := width - length
	if remainingWidth > 0 {
		lines := strings.Repeat(char, remainingWidth)
		lines = styles.ApplyForegroundGrad(t, lines, fromColor, toColor)
		title = title + " " + lines
	}
	return title
}
