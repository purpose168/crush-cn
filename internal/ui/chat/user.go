package chat

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/purpose168/crush-cn/internal/message"
	"github.com/purpose168/crush-cn/internal/ui/attachments"
	"github.com/purpose168/crush-cn/internal/ui/common"
	"github.com/purpose168/crush-cn/internal/ui/styles"
)

// UserMessageItem 表示聊天界面中的用户消息项。
type UserMessageItem struct {
	*highlightableMessageItem // 可高亮消息项
	*cachedMessageItem        // 缓存消息项
	*focusableMessageItem     // 可聚焦消息项

	attachments *attachments.Renderer // 附件渲染器
	message     *message.Message      // 消息内容
	sty         *styles.Styles        // 样式配置
}

// NewUserMessageItem 创建一个新的用户消息项实例。
func NewUserMessageItem(sty *styles.Styles, message *message.Message, attachments *attachments.Renderer) MessageItem {
	return &UserMessageItem{
		highlightableMessageItem: defaultHighlighter(sty),
		cachedMessageItem:        &cachedMessageItem{},
		focusableMessageItem:     &focusableMessageItem{},
		attachments:              attachments,
		message:                  message,
		sty:                      sty,
	}
}

// RawRender 实现 [MessageItem] 接口，渲染原始消息内容。
func (m *UserMessageItem) RawRender(width int) string {
	// 计算限制后的消息宽度
	cappedWidth := cappedMessageWidth(width)

	// 尝试从缓存获取已渲染的内容
	content, height, ok := m.getCachedRender(cappedWidth)
	// 缓存命中，直接返回已缓存的内容
	if ok {
		return m.renderHighlighted(content, cappedWidth, height)
	}

	// 创建 Markdown 渲染器
	renderer := common.MarkdownRenderer(m.sty, cappedWidth)

	// 获取消息文本内容并去除首尾空白
	msgContent := strings.TrimSpace(m.message.Content().Text)
	// 渲染 Markdown 内容
	result, err := renderer.Render(msgContent)
	if err != nil {
		// 渲染失败时使用原始文本
		content = msgContent
	} else {
		// 移除末尾的换行符
		content = strings.TrimSuffix(result, "\n")
	}

	// 如果消息包含二进制内容（附件），则渲染附件
	if len(m.message.BinaryContent()) > 0 {
		attachmentsStr := m.renderAttachments(cappedWidth)
		if content == "" {
			// 如果文本内容为空，仅显示附件
			content = attachmentsStr
		} else {
			// 否则将文本和附件合并显示
			content = strings.Join([]string{content, "", attachmentsStr}, "\n")
		}
	}

	// 计算渲染后的内容高度
	height = lipgloss.Height(content)
	// 缓存渲染结果
	m.setCachedRender(content, cappedWidth, height)
	return m.renderHighlighted(content, cappedWidth, height)
}

// Render 实现 MessageItem 接口，渲染带样式的用户消息。
func (m *UserMessageItem) Render(width int) string {
	// 默认使用未聚焦状态的样式
	style := m.sty.Chat.Message.UserBlurred
	// 如果消息处于聚焦状态，使用聚焦样式
	if m.focused {
		style = m.sty.Chat.Message.UserFocused
	}
	return style.Render(m.RawRender(width))
}

// ID 实现 MessageItem 接口，返回消息的唯一标识符。
func (m *UserMessageItem) ID() string {
	return m.message.ID
}

// renderAttachments 渲染消息中的附件内容。
func (m *UserMessageItem) renderAttachments(width int) string {
	// 构建附件列表
	var attachments []message.Attachment
	// 遍历消息中的二进制内容，转换为附件格式
	for _, at := range m.message.BinaryContent() {
		attachments = append(attachments, message.Attachment{
			FileName: at.Path,    // 文件路径作为文件名
			MimeType: at.MIMEType, // MIME 类型
		})
	}
	// 调用附件渲染器渲染附件列表
	return m.attachments.Render(attachments, false, width)
}

// HandleKeyEvent 实现 KeyEventHandler 接口，处理键盘事件。
func (m *UserMessageItem) HandleKeyEvent(key tea.KeyMsg) (bool, tea.Cmd) {
	// 检查按键是否为 "c" 或 "y"（复制快捷键）
	if k := key.String(); k == "c" || k == "y" {
		text := m.message.Content().Text
		// 复制消息文本到剪贴板
		return true, common.CopyToClipboard(text, "消息已复制到剪贴板")
	}
	return false, nil
}
