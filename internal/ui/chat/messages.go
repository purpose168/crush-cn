package chat

import (
	"fmt"
	"image"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/catwalk/pkg/catwalk"
	"charm.land/lipgloss/v2"
	"github.com/purpose168/crush-cn/internal/config"
	"github.com/purpose168/crush-cn/internal/message"
	"github.com/purpose168/crush-cn/internal/ui/anim"
	"github.com/purpose168/crush-cn/internal/ui/attachments"
	"github.com/purpose168/crush-cn/internal/ui/common"
	"github.com/purpose168/crush-cn/internal/ui/list"
	"github.com/purpose168/crush-cn/internal/ui/styles"
)

// MessageLeftPaddingTotal 是边框和内边距占用的总宽度。
// 我们还会限制文本最大宽度为 maxTextWidth(120)，以确保文本可读性。
const MessageLeftPaddingTotal = 2

// maxTextWidth 是文本消息的最大宽度
const maxTextWidth = 120

// Identifiable 是可提供唯一标识符的项目的接口。
type Identifiable interface {
	ID() string
}

// Animatable 是支持动画的项目的接口。
type Animatable interface {
	StartAnimation() tea.Cmd
	Animate(msg anim.StepMsg) tea.Cmd
}

// Expandable 是可展开或折叠的项目的接口。
type Expandable interface {
	// ToggleExpanded 切换项目的展开状态。
	// 返回项目当前是否处于展开状态。
	ToggleExpanded() bool
}

// KeyEventHandler 是可处理键盘事件的项目的接口。
type KeyEventHandler interface {
	HandleKeyEvent(key tea.KeyMsg) (bool, tea.Cmd)
}

// MessageItem 表示可在 UI 中显示并可作为 [list.List] 一部分的 [message.Message] 项目，
// 通过唯一 ID 进行标识。
type MessageItem interface {
	list.Item
	list.RawRenderable
	Identifiable
}

// HighlightableMessageItem 是支持高亮显示的消息项目。
type HighlightableMessageItem interface {
	MessageItem
	list.Highlightable
}

// FocusableMessageItem 是支持焦点的消息项目。
type FocusableMessageItem interface {
	MessageItem
	list.Focusable
}

// SendMsg 表示发送聊天消息的消息。
type SendMsg struct {
	Text        string
	Attachments []message.Attachment
}

type highlightableMessageItem struct {
	startLine   int
	startCol    int
	endLine     int
	endCol      int
	highlighter list.Highlighter
}

var _ list.Highlightable = (*highlightableMessageItem)(nil)

// isHighlighted 返回项目是否设置了高亮范围。
func (h *highlightableMessageItem) isHighlighted() bool {
	return h.startLine != -1 || h.endLine != -1
}

// renderHighlighted 在必要时对内容进行高亮处理。
func (h *highlightableMessageItem) renderHighlighted(content string, width, height int) string {
	if !h.isHighlighted() {
		return content
	}
	area := image.Rect(0, 0, width, height)
	return list.Highlight(content, area, h.startLine, h.startCol, h.endLine, h.endCol, h.highlighter)
}

// SetHighlight 实现 list.Highlightable 接口。
func (h *highlightableMessageItem) SetHighlight(startLine int, startCol int, endLine int, endCol int) {
	// 调整列位置以适应样式的左侧内边距（边框 + 内边距），
	// 因为我们只高亮内容部分。
	offset := MessageLeftPaddingTotal
	h.startLine = startLine
	h.startCol = max(0, startCol-offset)
	h.endLine = endLine
	if endCol >= 0 {
		h.endCol = max(0, endCol-offset)
	} else {
		h.endCol = endCol
	}
}

// Highlight 实现 list.Highlightable 接口。
func (h *highlightableMessageItem) Highlight() (startLine int, startCol int, endLine int, endCol int) {
	return h.startLine, h.startCol, h.endLine, h.endCol
}

func defaultHighlighter(sty *styles.Styles) *highlightableMessageItem {
	return &highlightableMessageItem{
		startLine:   -1,
		startCol:    -1,
		endLine:     -1,
		endCol:      -1,
		highlighter: list.ToHighlighter(sty.TextSelection),
	}
}

// cachedMessageItem 缓存已渲染的消息内容以避免重复渲染。
//
// 该结构应用于任何可以存储渲染缓存版本的消息，例如用户消息、助手消息等。
//
// 思考(kujtim): 我们应该考虑为不同宽度存储渲染结果是否高效，
// 这可能会导致内存占用问题。
type cachedMessageItem struct {
	// rendered 是缓存的渲染字符串
	rendered string
	// width 和 height 是缓存渲染的尺寸
	width  int
	height int
}

// getCachedRender 如果存在指定宽度的缓存渲染，则返回该缓存。
func (c *cachedMessageItem) getCachedRender(width int) (string, int, bool) {
	if c.width == width && c.rendered != "" {
		return c.rendered, c.height, true
	}
	return "", 0, false
}

// setCachedRender 设置缓存的渲染结果。
func (c *cachedMessageItem) setCachedRender(rendered string, width, height int) {
	c.rendered = rendered
	c.width = width
	c.height = height
}

// clearCache 清除缓存的渲染结果。
func (c *cachedMessageItem) clearCache() {
	c.rendered = ""
	c.width = 0
	c.height = 0
}

// focusableMessageItem 是可获得焦点的消息项目的基础结构。
type focusableMessageItem struct {
	focused bool
}

// SetFocused 实现 MessageItem 接口。
func (f *focusableMessageItem) SetFocused(focused bool) {
	f.focused = focused
}

// AssistantInfoID 返回助手信息项目的稳定 ID。
func AssistantInfoID(messageID string) string {
	return fmt.Sprintf("%s:assistant-info", messageID)
}

// AssistantInfoItem 在助手完成响应后渲染模型信息和响应时间。
type AssistantInfoItem struct {
	*cachedMessageItem

	id                  string
	message             *message.Message
	sty                 *styles.Styles
	cfg                 *config.Config
	lastUserMessageTime time.Time
}

// NewAssistantInfoItem 创建一个新的 AssistantInfoItem。
func NewAssistantInfoItem(sty *styles.Styles, message *message.Message, cfg *config.Config, lastUserMessageTime time.Time) MessageItem {
	return &AssistantInfoItem{
		cachedMessageItem:   &cachedMessageItem{},
		id:                  AssistantInfoID(message.ID),
		message:             message,
		sty:                 sty,
		cfg:                 cfg,
		lastUserMessageTime: lastUserMessageTime,
	}
}

// ID 实现 MessageItem 接口。
func (a *AssistantInfoItem) ID() string {
	return a.id
}

// RawRender 实现 MessageItem 接口。
func (a *AssistantInfoItem) RawRender(width int) string {
	innerWidth := max(0, width-MessageLeftPaddingTotal)
	content, _, ok := a.getCachedRender(innerWidth)
	if !ok {
		content = a.renderContent(innerWidth)
		height := lipgloss.Height(content)
		a.setCachedRender(content, innerWidth, height)
	}
	return content
}

// Render 实现 MessageItem 接口。
func (a *AssistantInfoItem) Render(width int) string {
	return a.sty.Chat.Message.SectionHeader.Render(a.RawRender(width))
}

func (a *AssistantInfoItem) renderContent(width int) string {
	finishData := a.message.FinishPart()
	if finishData == nil {
		return ""
	}
	finishTime := time.Unix(finishData.Time, 0)
	duration := finishTime.Sub(a.lastUserMessageTime)
	infoMsg := a.sty.Chat.Message.AssistantInfoDuration.Render(duration.String())
	icon := a.sty.Chat.Message.AssistantInfoIcon.Render(styles.ModelIcon)
	model := a.cfg.GetModel(a.message.Provider, a.message.Model)
	if model == nil {
		model = &catwalk.Model{Name: "未知模型"}
	}
	modelFormatted := a.sty.Chat.Message.AssistantInfoModel.Render(model.Name)
	providerName := a.message.Provider
	if providerConfig, ok := a.cfg.Providers.Get(a.message.Provider); ok {
		providerName = providerConfig.Name
	}
	provider := a.sty.Chat.Message.AssistantInfoProvider.Render(fmt.Sprintf("通过 %s", providerName))
	assistant := fmt.Sprintf("%s %s %s %s", icon, modelFormatted, provider, infoMsg)
	return common.Section(a.sty, assistant, width)
}

// cappedMessageWidth 返回消息内容的最大宽度以确保可读性。
func cappedMessageWidth(availableWidth int) int {
	return min(availableWidth-MessageLeftPaddingTotal, maxTextWidth)
}

// ExtractMessageItems 从 [message.Message] 中提取 [MessageItem]。
// 它返回消息的所有部分作为 [MessageItem] 列表。
//
// 对于包含工具调用的助手消息，传入 toolResults 映射以关联结果。
// 使用 BuildToolResultMap 从会话中的所有消息创建此映射。
func ExtractMessageItems(sty *styles.Styles, msg *message.Message, toolResults map[string]message.ToolResult) []MessageItem {
	switch msg.Role {
	case message.User:
		r := attachments.NewRenderer(
			sty.Attachments.Normal,
			sty.Attachments.Deleting,
			sty.Attachments.Image,
			sty.Attachments.Text,
		)
		return []MessageItem{NewUserMessageItem(sty, msg, r)}
	case message.Assistant:
		var items []MessageItem
		if ShouldRenderAssistantMessage(msg) {
			items = append(items, NewAssistantMessageItem(sty, msg))
		}
		for _, tc := range msg.ToolCalls() {
			var result *message.ToolResult
			if tr, ok := toolResults[tc.ID]; ok {
				result = &tr
			}
			items = append(items, NewToolMessageItem(
				sty,
				msg.ID,
				tc,
				result,
				msg.FinishReason() == message.FinishReasonCanceled,
			))
		}
		return items
	}
	return []MessageItem{}
}

// ShouldRenderAssistantMessage 判断是否应该渲染助手消息。
//
// 在某些情况下，助手消息仅包含工具调用，因此我们不希望渲染空消息。
func ShouldRenderAssistantMessage(msg *message.Message) bool {
	content := strings.TrimSpace(msg.Content().Text)
	thinking := strings.TrimSpace(msg.ReasoningContent().Thinking)
	isError := msg.FinishReason() == message.FinishReasonError
	isCancelled := msg.FinishReason() == message.FinishReasonCanceled
	hasToolCalls := len(msg.ToolCalls()) > 0
	return !hasToolCalls || content != "" || thinking != "" || msg.IsThinking() || isError || isCancelled
}

// BuildToolResultMap 从消息列表创建工具调用 ID 到其结果的映射。
// 工具结果消息（role == message.Tool）包含应关联到助手消息中工具调用的结果。
func BuildToolResultMap(messages []*message.Message) map[string]message.ToolResult {
	resultMap := make(map[string]message.ToolResult)
	for _, msg := range messages {
		if msg.Role == message.Tool {
			for _, result := range msg.ToolResults() {
				if result.ToolCallID != "" {
					resultMap[result.ToolCallID] = result
				}
			}
		}
	}
	return resultMap
}
