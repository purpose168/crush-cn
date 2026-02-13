// Package attachments 提供附件管理的用户界面组件
// 该包实现了附件列表的显示、添加和删除功能
package attachments

import (
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/purpose168/crush-cn/internal/message"
)

// maxFilename 定义文件名显示的最大字符数
const maxFilename = 15

// Keymap 定义附件组件的键盘快捷键映射
type Keymap struct {
	DeleteMode,
	DeleteAll,
	Escape key.Binding
}

// New 创建一个新的附件管理器实例
// 参数:
//   - renderer: 渲染器实例，用于渲染附件显示样式
//   - keyMap: 键盘快捷键映射配置
// 返回:
//   - *Attachments: 初始化后的附件管理器实例
func New(renderer *Renderer, keyMap Keymap) *Attachments {
	return &Attachments{
		keyMap:   keyMap,
		renderer: renderer,
	}
}

// Attachments 管理附件列表的UI组件
// 提供附件的添加、删除和显示功能
type Attachments struct {
	renderer *Renderer            // 渲染器，负责附件的样式渲染
	keyMap   Keymap               // 键盘快捷键映射
	list     []message.Attachment // 附件列表
	deleting bool                 // 是否处于删除模式
}

// List 返回当前附件列表
// 返回:
//   - []message.Attachment: 当前所有附件的切片
func (m *Attachments) List() []message.Attachment { return m.list }

// Reset 清空附件列表
func (m *Attachments) Reset() { m.list = nil }

// Update 处理消息更新，包括添加附件和键盘交互
// 参数:
//   - msg: 接收到的消息，可以是附件消息或键盘消息
// 返回:
//   - bool: 是否需要重新渲染界面
func (m *Attachments) Update(msg tea.Msg) bool {
	switch msg := msg.(type) {
	case message.Attachment:
		// 收到新附件，添加到列表中
		m.list = append(m.list, msg)
		return true
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keyMap.DeleteMode):
			// 进入删除模式（仅当列表不为空时）
			if len(m.list) > 0 {
				m.deleting = true
			}
			return true
		case m.deleting && key.Matches(msg, m.keyMap.Escape):
			// 在删除模式下按ESC键退出删除模式
			m.deleting = false
			return true
		case m.deleting && key.Matches(msg, m.keyMap.DeleteAll):
			// 在删除模式下删除所有附件
			m.deleting = false
			m.list = nil
			return true
		case m.deleting:
			// 处理数字键以删除单个附件
			r := msg.Code
			if r >= '0' && r <= '9' {
				num := int(r - '0')
				if num < len(m.list) {
					m.list = slices.Delete(m.list, num, num+1)
				}
				m.deleting = false
			}
			return true
		}
	}
	return false
}

// Render 渲染附件列表的显示内容
// 参数:
//   - width: 可用的显示宽度
// 返回:
//   - string: 渲染后的字符串表示
func (m *Attachments) Render(width int) string {
	return m.renderer.Render(m.list, m.deleting, width)
}

// NewRenderer 创建一个新的附件渲染器实例
// 参数:
//   - normalStyle: 普通文本的样式
//   - deletingStyle: 删除模式下数字的样式
//   - imageStyle: 图片附件图标的样式
//   - textStyle: 文本附件图标的样式
// 返回:
//   - *Renderer: 初始化后的渲染器实例
func NewRenderer(normalStyle, deletingStyle, imageStyle, textStyle lipgloss.Style) *Renderer {
	return &Renderer{
		normalStyle:   normalStyle,
		textStyle:     textStyle,
		imageStyle:    imageStyle,
		deletingStyle: deletingStyle,
	}
}

// Renderer 负责附件列表的样式渲染
type Renderer struct {
	normalStyle, textStyle, imageStyle, deletingStyle lipgloss.Style
}

// Render 渲染附件列表为可显示的字符串
// 参数:
//   - attachments: 要渲染的附件列表
//   - deleting: 是否处于删除模式
//   - width: 可用的显示宽度
// 返回:
//   - string: 渲染后的字符串，包含所有附件的视觉表示
func (r *Renderer) Render(attachments []message.Attachment, deleting bool, width int) string {
	var chips []string

	// 计算单个附件项的最大宽度（图标 + 文件名）
	maxItemWidth := lipgloss.Width(r.imageStyle.String() + r.normalStyle.Render(strings.Repeat("x", maxFilename)))
	// 计算可以完整显示的附件数量
	fits := int(math.Floor(float64(width)/float64(maxItemWidth))) - 1

	for i, att := range attachments {
		filename := filepath.Base(att.FileName)
		// 如果文件名过长，进行截断处理
		if ansi.StringWidth(filename) > maxFilename {
			filename = ansi.Truncate(filename, maxFilename, "…")
		}

		if deleting {
			// 删除模式：显示数字索引和文件名
			chips = append(
				chips,
				r.deletingStyle.Render(fmt.Sprintf("%d", i)),
				r.normalStyle.Render(filename),
			)
		} else {
			// 正常模式：显示图标和文件名
			chips = append(
				chips,
				r.icon(att).String(),
				r.normalStyle.Render(filename),
			)
		}

		// 如果超出显示范围，显示剩余附件数量
		if i == fits && len(attachments) > i {
			chips = append(chips, lipgloss.NewStyle().Width(maxItemWidth).Render(fmt.Sprintf("%d more…", len(attachments)-fits)))
			break
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, chips...)
}

// icon 根据附件类型返回对应的图标样式
// 参数:
//   - a: 附件对象
// 返回:
//   - lipgloss.Style: 对应的图标样式（图片样式或文本样式）
func (r *Renderer) icon(a message.Attachment) lipgloss.Style {
	if a.IsImage() {
		return r.imageStyle
	}
	return r.textStyle
}
