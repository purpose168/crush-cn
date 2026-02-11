package model

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/session"
	"github.com/charmbracelet/crush/internal/ui/chat"
	"github.com/charmbracelet/crush/internal/ui/styles"
)

// pillStyle 根据焦点状态返回药丸的适当样式。
func pillStyle(focused, panelFocused bool, t *styles.Styles) lipgloss.Style {
	if !panelFocused || focused {
		return t.Pills.Focused
	}
	return t.Pills.Blurred
}

const (
	// pillHeightWithBorder 是包含边框的药丸高度。
	pillHeightWithBorder = 3
	// maxTaskDisplayLength 是药丸中任务名称的最大长度。
	maxTaskDisplayLength = 40
	// maxQueueDisplayLength 是列表中队列项的最大长度。
	maxQueueDisplayLength = 60
)

// pillSection 表示药丸面板中哪个部分被聚焦。
type pillSection int

const (
	pillSectionTodos pillSection = iota
	pillSectionQueue
)

// hasIncompleteTodos 如果存在任何未完成的任务则返回true。
func hasIncompleteTodos(todos []session.Todo) bool {
	for _, todo := range todos {
		if todo.Status != session.TodoStatusCompleted {
			return true
		}
	}
	return false
}

// hasInProgressTodo 如果至少有一个进行中的任务则返回true。
func hasInProgressTodo(todos []session.Todo) bool {
	for _, todo := range todos {
		if todo.Status == session.TodoStatusInProgress {
			return true
		}
	}
	return false
}

// queuePill 渲染带有渐变三角形的队列计数药丸。
func queuePill(queue int, focused, panelFocused bool, t *styles.Styles) string {
	if queue <= 0 {
		return ""
	}
	triangles := styles.ForegroundGrad(t, "▶▶▶▶▶▶▶▶▶", false, t.RedDark, t.Secondary)
	if queue < len(triangles) {
		triangles = triangles[:queue]
	}

	text := t.Base.Render(fmt.Sprintf("%d 个排队", queue))
	content := fmt.Sprintf("%s %s", strings.Join(triangles, ""), text)
	return pillStyle(focused, panelFocused, t).Render(content)
}

// todoPill 渲染带有可选旋转器和任务名称的任务进度药丸。
func todoPill(todos []session.Todo, spinnerView string, focused, panelFocused bool, t *styles.Styles) string {
	if !hasIncompleteTodos(todos) {
		return ""
	}

	completed := 0
	var currentTodo *session.Todo
	for i := range todos {
		switch todos[i].Status {
		case session.TodoStatusCompleted:
			completed++
		case session.TodoStatusInProgress:
			if currentTodo == nil {
				currentTodo = &todos[i]
			}
		}
	}

	total := len(todos)

	label := t.Base.Render("待办")
	progress := t.Muted.Render(fmt.Sprintf("%d/%d", completed, total))

	var content string
	if panelFocused {
		content = fmt.Sprintf("%s %s", label, progress)
	} else if currentTodo != nil {
		taskText := currentTodo.Content
		if currentTodo.ActiveForm != "" {
			taskText = currentTodo.ActiveForm
		}
		if len(taskText) > maxTaskDisplayLength {
			taskText = taskText[:maxTaskDisplayLength-1] + "…"
		}
		task := t.Subtle.Render(taskText)
		content = fmt.Sprintf("%s %s %s  %s", spinnerView, label, progress, task)
	} else {
		content = fmt.Sprintf("%s %s", label, progress)
	}

	return pillStyle(focused, panelFocused, t).Render(content)
}

// todoList 渲染展开的任务列表。
func todoList(sessionTodos []session.Todo, spinnerView string, t *styles.Styles, width int) string {
	return chat.FormatTodosList(t, sessionTodos, spinnerView, width)
}

// queueList 渲染展开的队列项列表。
func queueList(queueItems []string, t *styles.Styles) string {
	if len(queueItems) == 0 {
		return ""
	}

	var lines []string
	for _, item := range queueItems {
		text := item
		if len(text) > maxQueueDisplayLength {
			text = text[:maxQueueDisplayLength-1] + "…"
		}
		prefix := t.Pills.QueueItemPrefix.Render() + " "
		lines = append(lines, prefix+t.Muted.Render(text))
	}

	return strings.Join(lines, "\n")
}

// togglePillsExpanded 切换药丸面板的展开状态。
func (m *UI) togglePillsExpanded() tea.Cmd {
	if !m.hasSession() {
		return nil
	}
	if m.layout.pills.Dy() > 0 {
		if cmd := m.chat.ScrollByAndAnimate(0); cmd != nil {
			return cmd
		}
	}
	hasPills := hasIncompleteTodos(m.session.Todos) || m.promptQueue > 0
	if !hasPills {
		return nil
	}
	m.pillsExpanded = !m.pillsExpanded
	if m.pillsExpanded {
		if hasIncompleteTodos(m.session.Todos) {
			m.focusedPillSection = pillSectionTodos
		} else {
			m.focusedPillSection = pillSectionQueue
		}
	}
	m.updateLayoutAndSize()
	return nil
}

// switchPillSection 在任务和队列部分之间切换焦点。
func (m *UI) switchPillSection(dir int) tea.Cmd {
	if !m.pillsExpanded || !m.hasSession() {
		return nil
	}
	hasIncompleteTodos := hasIncompleteTodos(m.session.Todos)
	hasQueue := m.promptQueue > 0

	if dir < 0 && m.focusedPillSection == pillSectionQueue && hasIncompleteTodos {
		m.focusedPillSection = pillSectionTodos
		m.updateLayoutAndSize()
		return nil
	}
	if dir > 0 && m.focusedPillSection == pillSectionTodos && hasQueue {
		m.focusedPillSection = pillSectionQueue
		m.updateLayoutAndSize()
		return nil
	}
	return nil
}

// pillsAreaHeight 计算药丸区域所需的总高度。
func (m *UI) pillsAreaHeight() int {
	if !m.hasSession() {
		return 0
	}
	hasIncomplete := hasIncompleteTodos(m.session.Todos)
	hasQueue := m.promptQueue > 0
	hasPills := hasIncomplete || hasQueue
	if !hasPills {
		return 0
	}

	pillsAreaHeight := pillHeightWithBorder
	if m.pillsExpanded {
		if m.focusedPillSection == pillSectionTodos && hasIncomplete {
			pillsAreaHeight += len(m.session.Todos)
		} else if m.focusedPillSection == pillSectionQueue && hasQueue {
			pillsAreaHeight += m.promptQueue
		}
	}
	return pillsAreaHeight
}

// renderPills 渲染药丸面板并将其存储在 m.pillsView 中。
func (m *UI) renderPills() {
	m.pillsView = ""
	if !m.hasSession() {
		return
	}

	width := m.layout.pills.Dx()
	if width <= 0 {
		return
	}

	paddingLeft := 3
	contentWidth := max(width-paddingLeft, 0)

	hasIncomplete := hasIncompleteTodos(m.session.Todos)
	hasQueue := m.promptQueue > 0

	if !hasIncomplete && !hasQueue {
		return
	}

	t := m.com.Styles
	todosFocused := m.pillsExpanded && m.focusedPillSection == pillSectionTodos
	queueFocused := m.pillsExpanded && m.focusedPillSection == pillSectionQueue

	inProgressIcon := t.Tool.TodoInProgressIcon.Render(styles.SpinnerIcon)
	if m.todoIsSpinning {
		inProgressIcon = m.todoSpinner.View()
	}

	var pills []string
	if hasIncomplete {
		pills = append(pills, todoPill(m.session.Todos, inProgressIcon, todosFocused, m.pillsExpanded, t))
	}
	if hasQueue {
		pills = append(pills, queuePill(m.promptQueue, queueFocused, m.pillsExpanded, t))
	}

	var expandedList string
	if m.pillsExpanded {
		if todosFocused && hasIncomplete {
			expandedList = todoList(m.session.Todos, inProgressIcon, t, contentWidth)
		} else if queueFocused && hasQueue {
			if m.com.App != nil && m.com.App.AgentCoordinator != nil {
				queueItems := m.com.App.AgentCoordinator.QueuedPromptsList(m.session.ID)
				expandedList = queueList(queueItems, t)
			}
		}
	}

	if len(pills) == 0 {
		return
	}

	pillsRow := lipgloss.JoinHorizontal(lipgloss.Top, pills...)

	helpDesc := "收起"
	if m.pillsExpanded {
		helpDesc = "展开"
	}
	helpKey := t.Pills.HelpKey.Render("ctrl+space")
	helpText := t.Pills.HelpText.Render(helpDesc)
	helpHint := lipgloss.JoinHorizontal(lipgloss.Center, helpKey, " ", helpText)
	pillsRow = lipgloss.JoinHorizontal(lipgloss.Center, pillsRow, " ", helpHint)

	pillsArea := pillsRow
	if expandedList != "" {
		pillsArea = lipgloss.JoinVertical(lipgloss.Left, pillsRow, expandedList)
	}

	m.pillsView = t.Pills.Area.MaxWidth(width).PaddingLeft(paddingLeft).Render(pillsArea)
}
