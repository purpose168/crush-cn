package model

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/clipperhouse/displaywidth"
	"github.com/clipperhouse/uax29/v2/words"
	"github.com/purpose168/crush-cn/internal/ui/anim"
	"github.com/purpose168/crush-cn/internal/ui/chat"
	"github.com/purpose168/crush-cn/internal/ui/common"
	"github.com/purpose168/crush-cn/internal/ui/list"
)

// 多点击检测常量
const (
	doubleClickThreshold = 400 * time.Millisecond // 0.4秒是典型的双击阈值
	clickTolerance       = 2                      // 双击/三击的x,y坐标容差
)

// DelayedClickMsg 在双击阈值后发送，如果没有发生双击，则触发单击操作（如展开）
type DelayedClickMsg struct {
	ClickID int // 点击ID，用于区分不同的点击操作
	ItemIdx int // 点击的项索引
	X, Y    int // 点击的坐标位置
}

// Chat 表示处理聊天交互和消息的聊天UI模型
type Chat struct {
	com      *common.Common
	list     *list.List
	idInxMap map[string]int // 消息ID到列表中索引的映射

	// 动画可见性优化：跟踪因项被滚动出视图而暂停的动画
	// 当项再次可见时，重新启动它们的动画
	pausedAnimations map[string]struct{}

	// 鼠标状态
	mouseDown     bool
	mouseDownItem int // 鼠标按下的项索引
	mouseDownX    int // 项内容中的X位置（字符偏移量）
	mouseDownY    int // 项中的Y位置（行偏移量）
	mouseDragItem int // 当前被拖动的项索引
	mouseDragX    int // 当前项内容中的X位置
	mouseDragY    int // 当前项中的Y位置

	// 双击/三击点击跟踪
	lastClickTime time.Time // 上次点击时间
	lastClickX    int       // 上次点击X坐标
	lastClickY    int       // 上次点击Y坐标
	clickCount    int       // 点击次数

	// 待处理的单击操作（延迟以检测双击）
	pendingClickID int // 每次点击递增，使旧的待处理点击失效
}

// NewChat 创建一个新的[Chat]实例，用于处理聊天交互和消息
func NewChat(com *common.Common) *Chat {
	c := &Chat{
		com:              com,
		idInxMap:         make(map[string]int),
		pausedAnimations: make(map[string]struct{}),
	}
	l := list.NewList()
	l.SetGap(1)
	l.RegisterRenderCallback(c.applyHighlightRange)
	l.RegisterRenderCallback(list.FocusedRenderCallback(l))
	c.list = l
	c.mouseDownItem = -1
	c.mouseDragItem = -1
	return c
}

// Height 返回聊天视图端口的高度
func (m *Chat) Height() int {
	return m.list.Height()
}

// Draw 将聊天UI组件渲染到屏幕和指定区域
func (m *Chat) Draw(scr uv.Screen, area uv.Rectangle) {
	uv.NewStyledString(m.list.Render()).Draw(scr, area)
}

// SetSize 设置聊天视图端口的大小
func (m *Chat) SetSize(width, height int) {
	m.list.SetSize(width, height)
	// 如果之前在底部，则保持在底部
	if m.list.AtBottom() {
		m.list.ScrollToBottom()
	}
}

// Len 返回聊天列表中的项数
func (m *Chat) Len() int {
	return m.list.Len()
}

// SetMessages 将聊天消息设置为提供的消息项列表
func (m *Chat) SetMessages(msgs ...chat.MessageItem) {
	m.idInxMap = make(map[string]int)
	m.pausedAnimations = make(map[string]struct{})

	items := make([]list.Item, len(msgs))
	for i, msg := range msgs {
		m.idInxMap[msg.ID()] = i
		// 为包含嵌套工具的工具注册嵌套工具ID
		if container, ok := msg.(chat.NestedToolContainer); ok {
			for _, nested := range container.NestedTools() {
				m.idInxMap[nested.ID()] = i
			}
		}
		items[i] = msg
	}
	m.list.SetItems(items...)
	m.list.ScrollToBottom()
}

// AppendMessages 将新的消息项追加到聊天列表
func (m *Chat) AppendMessages(msgs ...chat.MessageItem) {
	items := make([]list.Item, len(msgs))
	indexOffset := m.list.Len()
	for i, msg := range msgs {
		m.idInxMap[msg.ID()] = indexOffset + i
		// 为包含嵌套工具的工具注册嵌套工具ID
		if container, ok := msg.(chat.NestedToolContainer); ok {
			for _, nested := range container.NestedTools() {
				m.idInxMap[nested.ID()] = indexOffset + i
			}
		}
		items[i] = msg
	}
	m.list.AppendItems(items...)
}

// UpdateNestedToolIDs 更新容器内嵌套工具的ID映射
// 在修改嵌套工具后调用此方法，以确保动画正常工作
func (m *Chat) UpdateNestedToolIDs(containerID string) {
	idx, ok := m.idInxMap[containerID]
	if !ok {
		return
	}

	item, ok := m.list.ItemAt(idx).(chat.MessageItem)
	if !ok {
		return
	}

	container, ok := item.(chat.NestedToolContainer)
	if !ok {
		return
	}

	// 注册所有嵌套工具ID，使其指向容器的索引
	for _, nested := range container.NestedTools() {
		m.idInxMap[nested.ID()] = idx
	}
}

// Animate 对聊天列表中的项进行动画处理
// 仅将动画消息传播到可见项以节省CPU资源
// 当项不可见时，跟踪其动画ID，以便在项再次可见时重新启动动画
func (m *Chat) Animate(msg anim.StepMsg) tea.Cmd {
	idx, ok := m.idInxMap[msg.ID]
	if !ok {
		return nil
	}

	animatable, ok := m.list.ItemAt(idx).(chat.Animatable)
	if !ok {
		return nil
	}

	// 检查项当前是否可见
	startIdx, endIdx := m.list.VisibleItemIndices()
	isVisible := idx >= startIdx && idx <= endIdx

	if !isVisible {
		// 项不可见 - 通过不传播消息来暂停动画
		// 跟踪动画ID，以便在项可见时重新启动
		m.pausedAnimations[msg.ID] = struct{}{}
		return nil
	}

	// 项可见 - 从暂停集合中移除并执行动画
	delete(m.pausedAnimations, msg.ID)
	return animatable.Animate(msg)
}

// RestartPausedVisibleAnimations 重新启动因滚动出视图而暂停但现在再次可见的项的动画
func (m *Chat) RestartPausedVisibleAnimations() tea.Cmd {
	if len(m.pausedAnimations) == 0 {
		return nil
	}

	startIdx, endIdx := m.list.VisibleItemIndices()
	var cmds []tea.Cmd

	for id := range m.pausedAnimations {
		idx, ok := m.idInxMap[id]
		if !ok {
			// 项已不存在
			delete(m.pausedAnimations, id)
			continue
		}

		if idx >= startIdx && idx <= endIdx {
			// 项现在可见 - 重新启动其动画
			if animatable, ok := m.list.ItemAt(idx).(chat.Animatable); ok {
				if cmd := animatable.StartAnimation(); cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
			delete(m.pausedAnimations, id)
		}
	}

	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// Focus 设置聊天组件的聚焦状态
func (m *Chat) Focus() {
	m.list.Focus()
}

// Blur 移除聊天组件的聚焦状态
func (m *Chat) Blur() {
	m.list.Blur()
}

// ScrollToTopAndAnimate 将聊天视图滚动到顶部，并返回一个命令以重新启动现在可见的任何暂停动画
func (m *Chat) ScrollToTopAndAnimate() tea.Cmd {
	m.list.ScrollToTop()
	return m.RestartPausedVisibleAnimations()
}

// ScrollToBottomAndAnimate 将聊天视图滚动到底部，并返回一个命令以重新启动现在可见的任何暂停动画
func (m *Chat) ScrollToBottomAndAnimate() tea.Cmd {
	m.list.ScrollToBottom()
	return m.RestartPausedVisibleAnimations()
}

// ScrollByAndAnimate 将聊天视图滚动指定行数，并返回一个命令以重新启动现在可见的任何暂停动画
func (m *Chat) ScrollByAndAnimate(lines int) tea.Cmd {
	m.list.ScrollBy(lines)
	return m.RestartPausedVisibleAnimations()
}

// ScrollToSelectedAndAnimate 将聊天视图滚动到选中项，并返回一个命令以重新启动现在可见的任何暂停动画
func (m *Chat) ScrollToSelectedAndAnimate() tea.Cmd {
	m.list.ScrollToSelected()
	return m.RestartPausedVisibleAnimations()
}

// SelectedItemInView 返回选中项当前是否在视图中
func (m *Chat) SelectedItemInView() bool {
	return m.list.SelectedItemInView()
}

// isSelectable 判断指定索引的项是否可选中
func (m *Chat) isSelectable(index int) bool {
	item := m.list.ItemAt(index)
	if item == nil {
		return false
	}
	_, ok := item.(list.Focusable)
	return ok
}

// SetSelected 设置聊天列表中选中的消息索引
func (m *Chat) SetSelected(index int) {
	m.list.SetSelected(index)
	if index < 0 || index >= m.list.Len() {
		return
	}
	for {
		if m.isSelectable(m.list.Selected()) {
			return
		}
		if m.list.SelectNext() {
			continue
		}
		// 如果我们在末尾且最后一项不可选中，则向后查找最近的可选中项
		for {
			if !m.list.SelectPrev() {
				return
			}
			if m.isSelectable(m.list.Selected()) {
				return
			}
		}
	}
}

// SelectPrev 选择聊天列表中的上一条消息
func (m *Chat) SelectPrev() {
	for {
		if !m.list.SelectPrev() {
			return
		}
		if m.isSelectable(m.list.Selected()) {
			return
		}
	}
}

// SelectNext 选择聊天列表中的下一条消息
func (m *Chat) SelectNext() {
	for {
		if !m.list.SelectNext() {
			return
		}
		if m.isSelectable(m.list.Selected()) {
			return
		}
	}
}

// SelectFirst 选择聊天列表中的第一条消息
func (m *Chat) SelectFirst() {
	if !m.list.SelectFirst() {
		return
	}
	if m.isSelectable(m.list.Selected()) {
		return
	}
	for {
		if !m.list.SelectNext() {
			return
		}
		if m.isSelectable(m.list.Selected()) {
			return
		}
	}
}

// SelectLast 选择聊天列表中的最后一条消息
func (m *Chat) SelectLast() {
	if !m.list.SelectLast() {
		return
	}
	if m.isSelectable(m.list.Selected()) {
		return
	}
	for {
		if !m.list.SelectPrev() {
			return
		}
		if m.isSelectable(m.list.Selected()) {
			return
		}
	}
}

// SelectFirstInView 选择当前视图中的第一条消息
func (m *Chat) SelectFirstInView() {
	startIdx, endIdx := m.list.VisibleItemIndices()
	for i := startIdx; i <= endIdx; i++ {
		if m.isSelectable(i) {
			m.list.SetSelected(i)
			return
		}
	}
}

// SelectLastInView 选择当前视图中的最后一条消息
func (m *Chat) SelectLastInView() {
	startIdx, endIdx := m.list.VisibleItemIndices()
	for i := endIdx; i >= startIdx; i-- {
		if m.isSelectable(i) {
			m.list.SetSelected(i)
			return
		}
	}
}

// ClearMessages 清除聊天列表中的所有消息
func (m *Chat) ClearMessages() {
	m.idInxMap = make(map[string]int)
	m.pausedAnimations = make(map[string]struct{})
	m.list.SetItems()
	m.ClearMouse()
}

// RemoveMessage 根据ID从聊天列表中删除消息
func (m *Chat) RemoveMessage(id string) {
	idx, ok := m.idInxMap[id]
	if !ok {
		return
	}

	// 从列表中删除
	m.list.RemoveItem(idx)

	// 从索引映射中删除
	delete(m.idInxMap, id)

	// 重建删除项之后所有项的索引映射
	for i := idx; i < m.list.Len(); i++ {
		if item, ok := m.list.ItemAt(i).(chat.MessageItem); ok {
			m.idInxMap[item.ID()] = i
		}
	}

	// 清理此消息的任何暂停动画
	delete(m.pausedAnimations, id)
}

// MessageItem 返回具有给定ID的消息项，如果未找到则返回nil
func (m *Chat) MessageItem(id string) chat.MessageItem {
	idx, ok := m.idInxMap[id]
	if !ok {
		return nil
	}
	item, ok := m.list.ItemAt(idx).(chat.MessageItem)
	if !ok {
		return nil
	}
	return item
}

// ToggleExpandedSelectedItem 如果选中的消息项可展开，则切换其展开状态
func (m *Chat) ToggleExpandedSelectedItem() {
	if expandable, ok := m.list.SelectedItem().(chat.Expandable); ok {
		if !expandable.ToggleExpanded() {
			m.list.ScrollToIndex(m.list.Selected())
		}
		if m.list.AtBottom() {
			m.list.ScrollToBottom()
		}
	}
}

// HandleKeyMsg 处理聊天组件的键盘事件
func (m *Chat) HandleKeyMsg(key tea.KeyMsg) (bool, tea.Cmd) {
	if m.list.Focused() {
		if handler, ok := m.list.SelectedItem().(chat.KeyEventHandler); ok {
			return handler.HandleKeyEvent(key)
		}
	}
	return false, nil
}

// HandleMouseDown 处理聊天组件的鼠标按下事件
// 它检测单击、双击和三击以进行文本选择
// 返回是否处理了点击以及用于延迟单击操作的可选命令
func (m *Chat) HandleMouseDown(x, y int) (bool, tea.Cmd) {
	if m.list.Len() == 0 {
		return false, nil
	}

	itemIdx, itemY := m.list.ItemIndexAtPosition(x, y)
	if itemIdx < 0 {
		return false, nil
	}
	if !m.isSelectable(itemIdx) {
		return false, nil
	}

	// 递增待处理点击ID，使任何先前的待处理点击失效
	m.pendingClickID++
	clickID := m.pendingClickID

	// 检测多点击（双击/三击）
	now := time.Now()
	if now.Sub(m.lastClickTime) <= doubleClickThreshold &&
		abs(x-m.lastClickX) <= clickTolerance &&
		abs(y-m.lastClickY) <= clickTolerance {
		m.clickCount++
	} else {
		m.clickCount = 1
	}
	m.lastClickTime = now
	m.lastClickX = x
	m.lastClickY = y

	// 选择被点击的项
	m.list.SetSelected(itemIdx)

	var cmd tea.Cmd

	switch m.clickCount {
	case 1:
		// 单击 - 开始选择并安排延迟点击操作
		m.mouseDown = true
		m.mouseDownItem = itemIdx
		m.mouseDownX = x
		m.mouseDownY = itemY
		m.mouseDragItem = itemIdx
		m.mouseDragX = x
		m.mouseDragY = itemY

		// 安排延迟点击操作（如展开）在短延迟后执行
		// 如果发生双击，clickID将失效
		cmd = tea.Tick(doubleClickThreshold, func(t time.Time) tea.Msg {
			return DelayedClickMsg{
				ClickID: clickID,
				ItemIdx: itemIdx,
				X:       x,
				Y:       itemY,
			}
		})
	case 2:
		// 双击 - 选择单词（无延迟操作）
		m.selectWord(itemIdx, x, itemY)
	case 3:
		// 三击 - 选择行（无延迟操作）
		m.selectLine(itemIdx, itemY)
		m.clickCount = 0 // 三击后重置
	}

	return true, cmd
}

// HandleDelayedClick 处理延迟的单击操作（如展开）
// 仅在点击ID匹配（即未发生双击）且未进行文本选择（拖动选择）时执行
func (m *Chat) HandleDelayedClick(msg DelayedClickMsg) bool {
	// 如果此点击被较新的点击（双击/三击）取代，则忽略
	if msg.ClickID != m.pendingClickID {
		return false
	}

	// 如果用户拖动选择了文本，则不展开
	if m.HasHighlight() {
		return false
	}

	// 执行点击操作（如展开）
	selectedItem := m.list.SelectedItem()
	if clickable, ok := selectedItem.(list.MouseClickable); ok {
		handled := clickable.HandleMouseClick(ansi.MouseButton1, msg.X, msg.Y)
		// 如果适用，切换展开状态
		if expandable, ok := selectedItem.(chat.Expandable); ok {
			if !expandable.ToggleExpanded() {
				m.list.ScrollToIndex(m.list.Selected())
			}
		}
		if m.list.AtBottom() {
			m.list.ScrollToBottom()
		}
		return handled
	}

	return false
}

// HandleMouseUp 处理聊天组件的鼠标释放事件
func (m *Chat) HandleMouseUp(x, y int) bool {
	if !m.mouseDown {
		return false
	}

	m.mouseDown = false
	return true
}

// HandleMouseDrag 处理聊天组件的鼠标拖动事件
func (m *Chat) HandleMouseDrag(x, y int) bool {
	if !m.mouseDown {
		return false
	}

	if m.list.Len() == 0 {
		return false
	}

	itemIdx, itemY := m.list.ItemIndexAtPosition(x, y)
	if itemIdx < 0 {
		return false
	}

	m.mouseDragItem = itemIdx
	m.mouseDragX = x
	m.mouseDragY = itemY

	return true
}

// HasHighlight 返回当前是否有高亮内容
func (m *Chat) HasHighlight() bool {
	startItemIdx, startLine, startCol, endItemIdx, endLine, endCol := m.getHighlightRange()
	return startItemIdx >= 0 && endItemIdx >= 0 && (startLine != endLine || startCol != endCol)
}

// HighlightContent 根据鼠标选择返回当前高亮的内容
// 如果没有内容被高亮，则返回空字符串
func (m *Chat) HighlightContent() string {
	startItemIdx, startLine, startCol, endItemIdx, endLine, endCol := m.getHighlightRange()
	if startItemIdx < 0 || endItemIdx < 0 || startLine == endLine && startCol == endCol {
		return ""
	}

	var sb strings.Builder
	for i := startItemIdx; i <= endItemIdx; i++ {
		item := m.list.ItemAt(i)
		if hi, ok := item.(list.Highlightable); ok {
			startLine, startCol, endLine, endCol := hi.Highlight()
			listWidth := m.list.Width()
			var rendered string
			if rr, ok := item.(list.RawRenderable); ok {
				rendered = rr.RawRender(listWidth)
			} else {
				rendered = item.Render(listWidth)
			}
			sb.WriteString(list.HighlightContent(
				rendered,
				uv.Rect(0, 0, listWidth, lipgloss.Height(rendered)),
				startLine,
				startCol,
				endLine,
				endCol,
			))
			sb.WriteString(strings.Repeat("\n", m.list.Gap()))
		}
	}

	return strings.TrimSpace(sb.String())
}

// ClearMouse 清除当前鼠标交互状态
func (m *Chat) ClearMouse() {
	m.mouseDown = false
	m.mouseDownItem = -1
	m.mouseDragItem = -1
	m.lastClickTime = time.Time{}
	m.lastClickX = 0
	m.lastClickY = 0
	m.clickCount = 0
	m.pendingClickID++ // 使任何待处理的延迟点击失效
}

// applyHighlightRange 将当前高亮范围应用于聊天项
func (m *Chat) applyHighlightRange(idx, selectedIdx int, item list.Item) list.Item {
	if hi, ok := item.(list.Highlightable); ok {
		// 应用高亮
		startItemIdx, startLine, startCol, endItemIdx, endLine, endCol := m.getHighlightRange()
		sLine, sCol, eLine, eCol := -1, -1, -1, -1
		if idx >= startItemIdx && idx <= endItemIdx {
			if idx == startItemIdx && idx == endItemIdx {
				// 单项选择
				sLine = startLine
				sCol = startCol
				eLine = endLine
				eCol = endCol
			} else if idx == startItemIdx {
				// 第一项 - 从起始位置到项末尾
				sLine = startLine
				sCol = startCol
				eLine = -1
				eCol = -1
			} else if idx == endItemIdx {
				// 最后一项 - 从项开头到结束位置
				sLine = 0
				sCol = 0
				eLine = endLine
				eCol = endCol
			} else {
				// 中间项 - 完全高亮
				sLine = 0
				sCol = 0
				eLine = -1
				eCol = -1
			}
		}

		hi.SetHighlight(sLine, sCol, eLine, eCol)
		return hi.(list.Item)
	}

	return item
}

// getHighlightRange 返回当前高亮范围
func (m *Chat) getHighlightRange() (startItemIdx, startLine, startCol, endItemIdx, endLine, endCol int) {
	if m.mouseDownItem < 0 {
		return -1, -1, -1, -1, -1, -1
	}

	downItemIdx := m.mouseDownItem
	dragItemIdx := m.mouseDragItem

	// 确定选择方向
	draggingDown := dragItemIdx > downItemIdx ||
		(dragItemIdx == downItemIdx && m.mouseDragY > m.mouseDownY) ||
		(dragItemIdx == downItemIdx && m.mouseDragY == m.mouseDownY && m.mouseDragX >= m.mouseDownX)

	if draggingDown {
		// 正常正向选择
		startItemIdx = downItemIdx
		startLine = m.mouseDownY
		startCol = m.mouseDownX
		endItemIdx = dragItemIdx
		endLine = m.mouseDragY
		endCol = m.mouseDragX
	} else {
		// 反向选择（向上拖动）
		startItemIdx = dragItemIdx
		startLine = m.mouseDragY
		startCol = m.mouseDragX
		endItemIdx = downItemIdx
		endLine = m.mouseDownY
		endCol = m.mouseDownX
	}

	return startItemIdx, startLine, startCol, endItemIdx, endLine, endCol
}

// selectWord 选择项中指定位置的单词
func (m *Chat) selectWord(itemIdx, x, itemY int) {
	item := m.list.ItemAt(itemIdx)
	if item == nil {
		return
	}

	// 获取此项目的渲染内容
	var rendered string
	if rr, ok := item.(list.RawRenderable); ok {
		rendered = rr.RawRender(m.list.Width())
	} else {
		rendered = item.Render(m.list.Width())
	}

	lines := strings.Split(rendered, "\n")
	if itemY < 0 || itemY >= len(lines) {
		return
	}

	// 调整x坐标以考虑项的左侧填充（边框+内边距），获取内容列
	// 鼠标x坐标在视口空间中，但我们需要内容空间进行边界检测
	offset := chat.MessageLeftPaddingTotal
	contentX := x - offset
	if contentX < 0 {
		contentX = 0
	}

	line := ansi.Strip(lines[itemY])
	startCol, endCol := findWordBoundaries(line, contentX)
	if startCol == endCol {
		// 在该位置未找到单词，回退到单击行为
		m.mouseDown = true
		m.mouseDownItem = itemIdx
		m.mouseDownX = x
		m.mouseDownY = itemY
		m.mouseDragItem = itemIdx
		m.mouseDragX = x
		m.mouseDragY = itemY
		return
	}

	// 将选择设置为单词边界（转换回视口空间）
	// 保持mouseDown为true，以便HandleMouseUp触发复制操作
	m.mouseDown = true
	m.mouseDownItem = itemIdx
	m.mouseDownX = startCol + offset
	m.mouseDownY = itemY
	m.mouseDragItem = itemIdx
	m.mouseDragX = endCol + offset
	m.mouseDragY = itemY
}

// selectLine 选择项中指定位置的整行
func (m *Chat) selectLine(itemIdx, itemY int) {
	item := m.list.ItemAt(itemIdx)
	if item == nil {
		return
	}

	// 获取此项目的渲染内容
	var rendered string
	if rr, ok := item.(list.RawRenderable); ok {
		rendered = rr.RawRender(m.list.Width())
	} else {
		rendered = item.Render(m.list.Width())
	}

	lines := strings.Split(rendered, "\n")
	if itemY < 0 || itemY >= len(lines) {
		return
	}

	// 获取行长度（去除ANSI代码）并考虑填充
	// SetHighlight会减去偏移量，因此我们需要在这里添加它
	offset := chat.MessageLeftPaddingTotal
	lineLen := ansi.StringWidth(lines[itemY])

	// 将选择设置为整行
	// 保持mouseDown为true，以便HandleMouseUp触发复制操作
	m.mouseDown = true
	m.mouseDownItem = itemIdx
	m.mouseDownX = 0
	m.mouseDownY = itemY
	m.mouseDragItem = itemIdx
	m.mouseDragX = lineLen + offset
	m.mouseDragY = itemY
}

// findWordBoundaries 查找给定列中单词的起始和结束列
// 返回 (startCol, endCol)，其中 endCol 是排他的
func findWordBoundaries(line string, col int) (startCol, endCol int) {
	if line == "" || col < 0 {
		return 0, 0
	}

	i := displaywidth.StringGraphemes(line)
	for i.Next() {
	}

	// 使用UAX#29将行分割为单词
	lineCol := 0 // 跟踪已访问的列宽度
	lastCol := 0 // 跟踪当前令牌的起始位置
	iter := words.FromString(line)
	for iter.Next() {
		token := iter.Value()
		tokenWidth := displaywidth.String(token)

		graphemeStart := lineCol
		graphemeEnd := lineCol + tokenWidth
		lineCol += tokenWidth

		// 如果在此令牌之前点击，返回前一个令牌的边界
		if col < graphemeStart {
			return lastCol, lastCol
		}

		// 更新lastCol为此令牌的末尾，用于下一次迭代
		lastCol = graphemeEnd

		// 如果在此令牌内点击，返回其边界
		if col >= graphemeStart && col < graphemeEnd {
			// 如果点击在空白区域，返回空选择
			if strings.TrimSpace(token) == "" {
				return col, col
			}
			return graphemeStart, graphemeEnd
		}
	}

	return col, col
}

// abs 返回整数的绝对值
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
