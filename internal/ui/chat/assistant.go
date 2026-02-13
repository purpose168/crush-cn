package chat

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/purpose168/crush-cn/internal/message"
	"github.com/purpose168/crush-cn/internal/ui/anim"
	"github.com/purpose168/crush-cn/internal/ui/common"
	"github.com/purpose168/crush-cn/internal/ui/styles"
)

// assistantMessageTruncateFormat 是助手消息被截断时显示的文本。
const assistantMessageTruncateFormat = "… (已隐藏 %d 行) [点击或按空格键展开]"

// maxCollapsedThinkingHeight 定义思考内容折叠时的最大高度。
const maxCollapsedThinkingHeight = 10

// AssistantMessageItem 表示聊天界面中的助手消息项。
//
// 该消息项包含思考内容和主要消息内容，但不包括工具调用。
type AssistantMessageItem struct {
	*highlightableMessageItem
	*cachedMessageItem
	*focusableMessageItem

	message           *message.Message
	sty               *styles.Styles
	anim              *anim.Anim
	thinkingExpanded  bool
	thinkingBoxHeight int // 跟踪已渲染的思考框高度，用于点击检测。
}

// NewAssistantMessageItem 创建一个新的助手消息项。
func NewAssistantMessageItem(sty *styles.Styles, message *message.Message) MessageItem {
	a := &AssistantMessageItem{
		highlightableMessageItem: defaultHighlighter(sty),
		cachedMessageItem:        &cachedMessageItem{},
		focusableMessageItem:     &focusableMessageItem{},
		message:                  message,
		sty:                      sty,
	}

	a.anim = anim.New(anim.Settings{
		ID:          a.ID(),
		Size:        15,
		GradColorA:  sty.Primary,
		GradColorB:  sty.Secondary,
		LabelColor:  sty.FgBase,
		CycleColors: true,
	})
	return a
}

// StartAnimation 如果助手消息应该显示旋转动画，则启动动画。
func (a *AssistantMessageItem) StartAnimation() tea.Cmd {
	if !a.isSpinning() {
		return nil
	}
	return a.anim.Start()
}

// Animate 如果助手消息应该显示旋转动画，则推进动画进度。
func (a *AssistantMessageItem) Animate(msg anim.StepMsg) tea.Cmd {
	if !a.isSpinning() {
		return nil
	}
	return a.anim.Animate(msg)
}

// ID 实现 MessageItem 接口。
func (a *AssistantMessageItem) ID() string {
	return a.message.ID
}

// RawRender 实现 [MessageItem] 接口。
func (a *AssistantMessageItem) RawRender(width int) string {
	cappedWidth := cappedMessageWidth(width)

	var spinner string
	if a.isSpinning() {
		spinner = a.renderSpinning()
	}

	content, height, ok := a.getCachedRender(cappedWidth)
	if !ok {
		content = a.renderMessageContent(cappedWidth)
		height = lipgloss.Height(content)
		// 缓存已渲染的内容
		a.setCachedRender(content, cappedWidth, height)
	}

	highlightedContent := a.renderHighlighted(content, cappedWidth, height)
	if spinner != "" {
		if highlightedContent != "" {
			highlightedContent += "\n\n"
		}
		return highlightedContent + spinner
	}

	return highlightedContent
}

// Render 实现 MessageItem 接口。
func (a *AssistantMessageItem) Render(width int) string {
	style := a.sty.Chat.Message.AssistantBlurred
	if a.focused {
		style = a.sty.Chat.Message.AssistantFocused
	}
	return style.Render(a.RawRender(width))
}

// renderMessageContent 渲染消息内容，包括思考内容、主要内容和结束原因。
func (a *AssistantMessageItem) renderMessageContent(width int) string {
	var messageParts []string
	thinking := strings.TrimSpace(a.message.ReasoningContent().Thinking)
	content := strings.TrimSpace(a.message.Content().Text)
	// 如果消息包含推理内容，则首先添加
	if thinking != "" {
		messageParts = append(messageParts, a.renderThinking(a.message.ReasoningContent().Thinking, width))
	}

	// 然后添加主要内容
	if content != "" {
		// 在思考内容和主要内容之间添加间隔
		if thinking != "" {
			messageParts = append(messageParts, "")
		}
		messageParts = append(messageParts, a.renderMarkdown(content, width))
	}

	// 最后添加任何结束原因信息
	if a.message.IsFinished() {
		switch a.message.FinishReason() {
		case message.FinishReasonCanceled:
			messageParts = append(messageParts, a.sty.Base.Italic(true).Render("已取消"))
		case message.FinishReasonError:
			messageParts = append(messageParts, a.renderError(width))
		}
	}

	return strings.Join(messageParts, "\n")
}

// renderThinking 渲染思考/推理内容及其页脚。
func (a *AssistantMessageItem) renderThinking(thinking string, width int) string {
	renderer := common.PlainMarkdownRenderer(a.sty, width)
	rendered, err := renderer.Render(thinking)
	if err != nil {
		rendered = thinking
	}
	rendered = strings.TrimSpace(rendered)

	lines := strings.Split(rendered, "\n")
	totalLines := len(lines)

	isTruncated := totalLines > maxCollapsedThinkingHeight
	if !a.thinkingExpanded && isTruncated {
		lines = lines[totalLines-maxCollapsedThinkingHeight:]
		hint := a.sty.Chat.Message.ThinkingTruncationHint.Render(
			fmt.Sprintf(assistantMessageTruncateFormat, totalLines-maxCollapsedThinkingHeight),
		)
		lines = append([]string{hint, ""}, lines...)
	}

	thinkingStyle := a.sty.Chat.Message.ThinkingBox.Width(width)
	result := thinkingStyle.Render(strings.Join(lines, "\n"))
	a.thinkingBoxHeight = lipgloss.Height(result)

	var footer string
	// 如果思考已完成，添加思考用时作为页脚
	if !a.message.IsThinking() || len(a.message.ToolCalls()) > 0 {
		duration := a.message.ThinkingDuration()
		if duration.String() != "0s" {
			footer = a.sty.Chat.Message.ThinkingFooterTitle.Render("思考用时 ") +
				a.sty.Chat.Message.ThinkingFooterDuration.Render(duration.String())
		}
	}

	if footer != "" {
		result += "\n\n" + footer
	}

	return result
}

// renderMarkdown 将内容渲染为 Markdown 格式。
func (a *AssistantMessageItem) renderMarkdown(content string, width int) string {
	renderer := common.MarkdownRenderer(a.sty, width)
	result, err := renderer.Render(content)
	if err != nil {
		return content
	}
	return strings.TrimSuffix(result, "\n")
}

func (a *AssistantMessageItem) renderSpinning() string {
	if a.message.IsThinking() {
		a.anim.SetLabel("思考中")
	} else if a.message.IsSummaryMessage {
		a.anim.SetLabel("总结中")
	}
	return a.anim.Render()
}

// renderError 渲染错误消息。
func (a *AssistantMessageItem) renderError(width int) string {
	finishPart := a.message.FinishPart()
	errTag := a.sty.Chat.Message.ErrorTag.Render("错误")
	truncated := ansi.Truncate(finishPart.Message, width-2-lipgloss.Width(errTag), "...")
	title := fmt.Sprintf("%s %s", errTag, a.sty.Chat.Message.ErrorTitle.Render(truncated))
	details := a.sty.Chat.Message.ErrorDetails.Width(width - 2).Render(finishPart.Details)
	return fmt.Sprintf("%s\n\n%s", title, details)
}

// isSpinning 返回助手消息是否仍在生成中。
func (a *AssistantMessageItem) isSpinning() bool {
	isThinking := a.message.IsThinking()
	isFinished := a.message.IsFinished()
	hasContent := strings.TrimSpace(a.message.Content().Text) != ""
	hasToolCalls := len(a.message.ToolCalls()) > 0
	return (isThinking || !isFinished) && !hasContent && !hasToolCalls
}

// SetMessage 用于更新底层的消息对象。
func (a *AssistantMessageItem) SetMessage(message *message.Message) tea.Cmd {
	wasSpinning := a.isSpinning()
	a.message = message
	a.clearCache()
	if !wasSpinning && a.isSpinning() {
		return a.StartAnimation()
	}
	return nil
}

// ToggleExpanded 切换思考框的展开状态。
func (a *AssistantMessageItem) ToggleExpanded() {
	a.thinkingExpanded = !a.thinkingExpanded
	a.clearCache()
}

// HandleMouseClick 实现 MouseClickable 接口。
func (a *AssistantMessageItem) HandleMouseClick(btn ansi.MouseButton, x, y int) bool {
	if btn != ansi.MouseLeft {
		return false
	}
	// 检查点击是否在思考框内
	if a.thinkingBoxHeight > 0 && y < a.thinkingBoxHeight {
		a.ToggleExpanded()
		return true
	}
	return false
}

// HandleKeyEvent 实现 KeyEventHandler 接口。
func (a *AssistantMessageItem) HandleKeyEvent(key tea.KeyMsg) (bool, tea.Cmd) {
	if k := key.String(); k == "c" || k == "y" {
		text := a.message.Content().Text
		return true, common.CopyToClipboard(text, "消息已复制到剪贴板")
	}
	return false, nil
}
