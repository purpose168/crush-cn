package model

import (
	"context"
	"log/slog"

	tea "charm.land/bubbletea/v2"

	"github.com/purpose168/crush-cn/internal/message"
)

// promptHistoryLoadedMsg 当提示历史加载完成时发送的消息类型。
type promptHistoryLoadedMsg struct {
	messages []string
}

// loadPromptHistory 加载用户消息以供历史导航使用。
func (m *UI) loadPromptHistory() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var messages []message.Message
		var err error

		if m.session != nil {
			messages, err = m.com.App.Messages.ListUserMessages(ctx, m.session.ID)
		} else {
			messages, err = m.com.App.Messages.ListAllUserMessages(ctx)
		}
		if err != nil {
			slog.Error("加载提示历史失败", "error", err)
			return promptHistoryLoadedMsg{messages: nil}
		}

		texts := make([]string, 0, len(messages))
		for _, msg := range messages {
			if text := msg.Content().Text; text != "" {
				texts = append(texts, text)
			}
		}
		return promptHistoryLoadedMsg{messages: texts}
	}
}

// handleHistoryUp 处理向上箭头键用于历史导航。
func (m *UI) handleHistoryUp(msg tea.Msg) tea.Cmd {
	// 从光标位置(0,0)导航到较旧的历史条目。
	if m.textarea.Length() == 0 || m.isAtEditorStart() {
		if m.historyPrev() {
			// 发送此消息以使文本区域将视图移动到正确的位置，
			// 如果没有此操作，光标会显示在错误的位置。
			ta, cmd := m.textarea.Update(nil)
			m.textarea = ta
			return cmd
		}
	}

	// 首先在进入历史记录之前将光标移动到开头。
	if m.textarea.Line() == 0 {
		m.textarea.CursorStart()
		return nil
	}

	// 让文本区域处理正常的光标移动。
	ta, cmd := m.textarea.Update(msg)
	m.textarea = ta
	return cmd
}

// handleHistoryDown 处理向下箭头键用于历史导航。
func (m *UI) handleHistoryDown(msg tea.Msg) tea.Cmd {
	// 从文本末尾导航到较新的历史条目。
	if m.isAtEditorEnd() {
		if m.historyNext() {
			// 发送此消息以使文本区域将视图移动到正确的位置，
			// 如果没有此操作，光标会显示在错误的位置。
			ta, cmd := m.textarea.Update(nil)
			m.textarea = ta
			return cmd
		}
	}

	// 首先在导航历史记录之前将光标移动到末尾。
	if m.textarea.Line() == max(m.textarea.LineCount()-1, 0) {
		m.textarea.MoveToEnd()
		ta, cmd := m.textarea.Update(nil)
		m.textarea = ta
		return cmd
	}

	// 让文本区域处理正常的光标移动。
	ta, cmd := m.textarea.Update(msg)
	m.textarea = ta
	return cmd
}

// handleHistoryEscape 处理退出键用于退出历史导航。
func (m *UI) handleHistoryEscape(msg tea.Msg) tea.Cmd {
	// 浏览历史记录时返回当前草稿。
	if m.promptHistory.index >= 0 {
		m.promptHistory.index = -1
		m.textarea.Reset()
		m.textarea.InsertString(m.promptHistory.draft)
		ta, cmd := m.textarea.Update(nil)
		m.textarea = ta
		return cmd
	}

	// 让文本区域正常处理退出键。
	ta, cmd := m.textarea.Update(msg)
	m.textarea = ta
	return cmd
}

// updateHistoryDraft 当文本被修改时更新历史状态。
func (m *UI) updateHistoryDraft(oldValue string) {
	if m.textarea.Value() != oldValue {
		m.promptHistory.draft = m.textarea.Value()
		m.promptHistory.index = -1
	}
}

// historyPrev 将文本区域内容更改为历史记录中的上一条消息。
// 如果找不到上一条消息则返回false。
func (m *UI) historyPrev() bool {
	if len(m.promptHistory.messages) == 0 {
		return false
	}
	if m.promptHistory.index == -1 {
		m.promptHistory.draft = m.textarea.Value()
	}
	nextIndex := m.promptHistory.index + 1
	if nextIndex >= len(m.promptHistory.messages) {
		return false
	}
	m.promptHistory.index = nextIndex
	m.textarea.Reset()
	m.textarea.InsertString(m.promptHistory.messages[nextIndex])
	m.textarea.MoveToBegin()
	return true
}

// historyNext 将文本区域内容更改为历史记录中的下一条消息。
// 如果找不到下一条消息则返回false。
func (m *UI) historyNext() bool {
	if m.promptHistory.index < 0 {
		return false
	}
	nextIndex := m.promptHistory.index - 1
	if nextIndex < 0 {
		m.promptHistory.index = -1
		m.textarea.Reset()
		m.textarea.InsertString(m.promptHistory.draft)
		return true
	}
	m.promptHistory.index = nextIndex
	m.textarea.Reset()
	m.textarea.InsertString(m.promptHistory.messages[nextIndex])
	return true
}

// historyReset 重置历史记录，但不清除消息内容。
// 它只是将当前草稿设置为空并重置历史记录中的位置。
func (m *UI) historyReset() {
	m.promptHistory.index = -1
	m.promptHistory.draft = ""
}

// isAtEditorStart 检查是否在文本区域的第0行第0列位置。
func (m *UI) isAtEditorStart() bool {
	return m.textarea.Line() == 0 && m.textarea.LineInfo().ColumnOffset == 0
}

// isAtEditorEnd 检查是否在文本区域的最后一行最后一列位置。
func (m *UI) isAtEditorEnd() bool {
	lineCount := m.textarea.LineCount()
	if lineCount == 0 {
		return true
	}
	if m.textarea.Line() != lineCount-1 {
		return false
	}
	info := m.textarea.LineInfo()
	return info.CharOffset >= info.CharWidth-1 || info.CharWidth == 0
}
