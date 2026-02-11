package list

import (
	"strings"
)

// List 表示一个可以懒渲染的项目列表。列表总是像聊天对话一样渲染，
// 项目从上到下垂直堆叠。
type List struct {
	// 视口大小
	width, height int

	// 列表中的项目
	items []Item

	// 项目之间的间隔（0或更小表示没有间隔）
	gap int

	// 以相反顺序显示列表
	reverse bool

	// 焦点和选择状态
	focused     bool
	selectedIdx int // 当前选中的索引，-1表示没有选择

	// offsetIdx 是视口中第一个可见项目的索引。
	offsetIdx int
	// offsetLine 是offsetIdx处的项目滚动出视图（在视口上方）的行数。
	// 它必须始终 >= 0。
	offsetLine int

	// renderCallbacks 是渲染项目时要应用的回调列表。
	renderCallbacks []func(idx, selectedIdx int, item Item) Item
}

// renderedItem 保存项目的渲染内容和高度。
type renderedItem struct {
	content string
	height  int
}

// NewList 创建一个新的懒加载列表。
func NewList(items ...Item) *List {
	l := new(List)
	l.items = items
	l.selectedIdx = -1
	return l
}

// RenderCallback 定义一个可以在渲染项目之前修改项目的函数。
type RenderCallback func(idx, selectedIdx int, item Item) Item

// RegisterRenderCallback 注册一个在渲染项目时要调用的回调。
// 这可以用于在项目渲染之前修改它们。
func (l *List) RegisterRenderCallback(cb RenderCallback) {
	l.renderCallbacks = append(l.renderCallbacks, cb)
}

// SetSize 设置列表视口的大小。
func (l *List) SetSize(width, height int) {
	l.width = width
	l.height = height
}

// SetGap 设置项目之间的间隔。
func (l *List) SetGap(gap int) {
	l.gap = gap
}

// Gap 返回项目之间的间隔。
func (l *List) Gap() int {
	return l.gap
}

// AtBottom 返回列表是否在底部显示最后一个项目。
func (l *List) AtBottom() bool {
	const margin = 2

	if len(l.items) == 0 || l.offsetIdx >= len(l.items)-1 {
		return true
	}

	// 计算从offsetIdx到末尾的高度。
	var totalHeight int
	for idx := l.offsetIdx; idx < len(l.items); idx++ {
		item := l.getItem(idx)
		itemHeight := item.height
		if l.gap > 0 && idx > l.offsetIdx {
			itemHeight += l.gap
		}
		totalHeight += itemHeight
	}

	return totalHeight-l.offsetLine-margin <= l.height
}

// SetReverse 以相反顺序显示列表。
func (l *List) SetReverse(reverse bool) {
	l.reverse = reverse
}

// Width 返回列表视口的宽度。
func (l *List) Width() int {
	return l.width
}

// Height 返回列表视口的高度。
func (l *List) Height() int {
	return l.height
}

// Len 返回列表中的项目数量。
func (l *List) Len() int {
	return len(l.items)
}

// lastOffsetItem 返回可以在视口中部分可见的最后一个项目的索引和行偏移。
func (l *List) lastOffsetItem() (int, int, int) {
	var totalHeight int
	var idx int
	for idx = len(l.items) - 1; idx >= 0; idx-- {
		item := l.getItem(idx)
		itemHeight := item.height
		if l.gap > 0 && idx < len(l.items)-1 {
			itemHeight += l.gap
		}
		totalHeight += itemHeight
		if totalHeight > l.height {
			break
		}
	}

	// 计算项目内的行偏移
	lineOffset := max(totalHeight-l.height, 0)
	idx = max(idx, 0)

	return idx, lineOffset, totalHeight
}

// getItem 渲染（如果需要）并返回给定索引处的项目。
func (l *List) getItem(idx int) renderedItem {
	if idx < 0 || idx >= len(l.items) {
		return renderedItem{}
	}

	item := l.items[idx]
	if len(l.renderCallbacks) > 0 {
		for _, cb := range l.renderCallbacks {
			if it := cb(idx, l.selectedIdx, item); it != nil {
				item = it
			}
		}
	}

	rendered := item.Render(l.width)
	rendered = strings.TrimRight(rendered, "\n")
	height := strings.Count(rendered, "\n") + 1
	ri := renderedItem{
		content: rendered,
		height:  height,
	}

	return ri
}

// ScrollToIndex 将列表滚动到给定的项目索引。
func (l *List) ScrollToIndex(index int) {
	if index < 0 {
		index = 0
	}
	if index >= len(l.items) {
		index = len(l.items) - 1
	}
	l.offsetIdx = index
	l.offsetLine = 0
}

// ScrollBy 按给定的行数滚动列表。
func (l *List) ScrollBy(lines int) {
	if len(l.items) == 0 || lines == 0 {
		return
	}

	if l.reverse {
		lines = -lines
	}

	if lines > 0 {
		if l.AtBottom() {
			// 已经在底部
			return
		}

		// 向下滚动
		l.offsetLine += lines
		currentItem := l.getItem(l.offsetIdx)
		for l.offsetLine >= currentItem.height {
			l.offsetLine -= currentItem.height
			if l.gap > 0 {
				l.offsetLine = max(0, l.offsetLine-l.gap)
			}

			// 移动到下一个项目
			l.offsetIdx++
			if l.offsetIdx > len(l.items)-1 {
				// 到达底部
				l.ScrollToBottom()
				return
			}
			currentItem = l.getItem(l.offsetIdx)
		}

		lastOffsetIdx, lastOffsetLine, _ := l.lastOffsetItem()
		if l.offsetIdx > lastOffsetIdx || (l.offsetIdx == lastOffsetIdx && l.offsetLine > lastOffsetLine) {
			// 限制在底部
			l.offsetIdx = lastOffsetIdx
			l.offsetLine = lastOffsetLine
		}
	} else if lines < 0 {
		// 向上滚动
		l.offsetLine += lines // lines是负数
		for l.offsetLine < 0 {
			// 移动到上一个项目
			l.offsetIdx--
			if l.offsetIdx < 0 {
				// 到达顶部
				l.ScrollToTop()
				break
			}
			prevItem := l.getItem(l.offsetIdx)
			totalHeight := prevItem.height
			if l.gap > 0 {
				totalHeight += l.gap
			}
			l.offsetLine += totalHeight
		}
	}
}

// VisibleItemIndices 查找视口中可见的项目范围。
// 这用于检查选中的项目是否在视图中。
func (l *List) VisibleItemIndices() (startIdx, endIdx int) {
	if len(l.items) == 0 {
		return 0, 0
	}

	startIdx = l.offsetIdx
	currentIdx := startIdx
	visibleHeight := -l.offsetLine

	for currentIdx < len(l.items) {
		item := l.getItem(currentIdx)
		visibleHeight += item.height
		if l.gap > 0 {
			visibleHeight += l.gap
		}

		if visibleHeight >= l.height {
			break
		}
		currentIdx++
	}

	endIdx = currentIdx
	if endIdx >= len(l.items) {
		endIdx = len(l.items) - 1
	}

	return startIdx, endIdx
}

// Render 渲染列表并返回可见行。
func (l *List) Render() string {
	if len(l.items) == 0 {
		return ""
	}

	var lines []string
	currentIdx := l.offsetIdx
	currentOffset := l.offsetLine

	linesNeeded := l.height

	for linesNeeded > 0 && currentIdx < len(l.items) {
		item := l.getItem(currentIdx)
		itemLines := strings.Split(item.content, "\n")
		itemHeight := len(itemLines)

		if currentOffset >= 0 && currentOffset < itemHeight {
			// 添加可见内容行
			lines = append(lines, itemLines[currentOffset:]...)

			// 如果不是绝对最后一个视觉元素，则添加间隔（概念上间隔在项目之间）
			// 但在循环中我们可以直接添加它并在稍后修剪
			if l.gap > 0 {
				for i := 0; i < l.gap; i++ {
					lines = append(lines, "")
				}
			}
		} else {
			// offsetLine从间隔开始
			gapOffset := currentOffset - itemHeight
			gapRemaining := l.gap - gapOffset
			if gapRemaining > 0 {
				for range gapRemaining {
					lines = append(lines, "")
				}
			}
		}

		linesNeeded = l.height - len(lines)
		currentIdx++
		currentOffset = 0 // 为后续项目重置偏移
	}

	l.height = max(l.height, 0)

	if len(lines) > l.height {
		lines = lines[:l.height]
	}

	if l.reverse {
		// 反转行，使列表从下到上渲染。
		for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
			lines[i], lines[j] = lines[j], lines[i]
		}
	}

	return strings.Join(lines, "\n")
}

// PrependItems 将项目前置到列表。
func (l *List) PrependItems(items ...Item) {
	l.items = append(items, l.items...)

	// 保持视图位置相对于可见内容
	l.offsetIdx += len(items)

	// 如果有效则更新选择索引
	if l.selectedIdx != -1 {
		l.selectedIdx += len(items)
	}
}

// SetItems 设置列表中的项目。
func (l *List) SetItems(items ...Item) {
	l.setItems(true, items...)
}

// setItems 设置列表中的项目。如果evict为true，它清除
// 渲染的项目缓存。
func (l *List) setItems(evict bool, items ...Item) {
	l.items = items
	l.selectedIdx = min(l.selectedIdx, len(l.items)-1)
	l.offsetIdx = min(l.offsetIdx, len(l.items)-1)
	l.offsetLine = 0
}

// AppendItems 将项目追加到列表。
func (l *List) AppendItems(items ...Item) {
	l.items = append(l.items, items...)
}

// RemoveItem 从列表中移除给定索引处的项目。
func (l *List) RemoveItem(idx int) {
	if idx < 0 || idx >= len(l.items) {
		return
	}

	// 移除项目
	l.items = append(l.items[:idx], l.items[idx+1:]...)

	// 如果需要则调整选择
	if l.selectedIdx == idx {
		l.selectedIdx = -1
	} else if l.selectedIdx > idx {
		l.selectedIdx--
	}

	// 如果需要则调整偏移
	if l.offsetIdx > idx {
		l.offsetIdx--
	} else if l.offsetIdx == idx && l.offsetIdx >= len(l.items) {
		l.offsetIdx = max(0, len(l.items)-1)
		l.offsetLine = 0
	}
}

// Focused 返回列表是否聚焦。
func (l *List) Focused() bool {
	return l.focused
}

// Focus 设置列表的焦点状态。
func (l *List) Focus() {
	l.focused = true
}

// Blur 从列表中移除焦点状态。
func (l *List) Blur() {
	l.focused = false
}

// ScrollToTop 将列表滚动到顶部。
func (l *List) ScrollToTop() {
	l.offsetIdx = 0
	l.offsetLine = 0
}

// ScrollToBottom 将列表滚动到底部。
func (l *List) ScrollToBottom() {
	if len(l.items) == 0 {
		return
	}

	lastOffsetIdx, lastOffsetLine, _ := l.lastOffsetItem()
	l.offsetIdx = lastOffsetIdx
	l.offsetLine = lastOffsetLine
}

// ScrollToSelected 将列表滚动到选中的项目。
func (l *List) ScrollToSelected() {
	if l.selectedIdx < 0 || l.selectedIdx >= len(l.items) {
		return
	}

	startIdx, endIdx := l.VisibleItemIndices()
	if l.selectedIdx < startIdx {
		// 选中的项目在可见范围之上
		l.offsetIdx = l.selectedIdx
		l.offsetLine = 0
	} else if l.selectedIdx > endIdx {
		// 选中的项目在可见范围之下
		// 滚动以使选中的项目位于底部
		var totalHeight int
		for i := l.selectedIdx; i >= 0; i-- {
			item := l.getItem(i)
			totalHeight += item.height
			if l.gap > 0 && i < l.selectedIdx {
				totalHeight += l.gap
			}
			if totalHeight >= l.height {
				l.offsetIdx = i
				l.offsetLine = totalHeight - l.height
				break
			}
		}
		if totalHeight < l.height {
			// 所有项目都适合视口
			l.ScrollToTop()
		}
	}
}

// SelectedItemInView 返回选中的项目当前是否在视图中。
func (l *List) SelectedItemInView() bool {
	if l.selectedIdx < 0 || l.selectedIdx >= len(l.items) {
		return false
	}
	startIdx, endIdx := l.VisibleItemIndices()
	return l.selectedIdx >= startIdx && l.selectedIdx <= endIdx
}

// SetSelected 在列表中设置选中的项目索引。
// 如果索引超出范围，它返回-1。
func (l *List) SetSelected(index int) {
	if index < 0 || index >= len(l.items) {
		l.selectedIdx = -1
	} else {
		l.selectedIdx = index
	}
}

// Selected 返回当前选中项目的索引。如果没有
// 项目被选中，它返回-1。
func (l *List) Selected() int {
	return l.selectedIdx
}

// IsSelectedFirst 返回第一个项目是否被选中。
func (l *List) IsSelectedFirst() bool {
	return l.selectedIdx == 0
}

// IsSelectedLast 返回最后一个项目是否被选中。
func (l *List) IsSelectedLast() bool {
	return l.selectedIdx == len(l.items)-1
}

// SelectPrev 选择视觉上的上一个项目（向视觉顶部移动）。
// 它返回选择是否已更改。
func (l *List) SelectPrev() bool {
	if l.reverse {
		// 在反向模式下，视觉向上=更高的索引
		if l.selectedIdx < len(l.items)-1 {
			l.selectedIdx++
			return true
		}
	} else {
		// 正常模式：视觉向上=更低的索引
		if l.selectedIdx > 0 {
			l.selectedIdx--
			return true
		}
	}
	return false
}

// SelectNext 选择列表中的下一个项目。
// 它返回选择是否已更改。
func (l *List) SelectNext() bool {
	if l.reverse {
		// 在反向模式下，视觉向下=更低的索引
		if l.selectedIdx > 0 {
			l.selectedIdx--
			return true
		}
	} else {
		// 正常模式：视觉向下=更高的索引
		if l.selectedIdx < len(l.items)-1 {
			l.selectedIdx++
			return true
		}
	}
	return false
}

// SelectFirst 选择列表中的第一个项目。
// 它返回选择是否已更改。
func (l *List) SelectFirst() bool {
	if len(l.items) == 0 {
		return false
	}
	l.selectedIdx = 0
	return true
}

// SelectLast 选择列表中的最后一个项目（最高索引）。
// 它返回选择是否已更改。
func (l *List) SelectLast() bool {
	if len(l.items) == 0 {
		return false
	}
	l.selectedIdx = len(l.items) - 1
	return true
}

// WrapToStart 将选择包装到视觉开始（用于循环导航）。
// 在正常模式下，这是索引0。在反向模式下，这是最高索引。
func (l *List) WrapToStart() bool {
	if len(l.items) == 0 {
		return false
	}
	if l.reverse {
		l.selectedIdx = len(l.items) - 1
	} else {
		l.selectedIdx = 0
	}
	return true
}

// WrapToEnd 将选择包装到视觉结束（用于循环导航）。
// 在正常模式下，这是最高索引。在反向模式下，这是索引0。
func (l *List) WrapToEnd() bool {
	if len(l.items) == 0 {
		return false
	}
	if l.reverse {
		l.selectedIdx = 0
	} else {
		l.selectedIdx = len(l.items) - 1
	}
	return true
}

// SelectedItem 返回当前选中的项目。如果没有
// 项目被选中，它可能为nil。
func (l *List) SelectedItem() Item {
	if l.selectedIdx < 0 || l.selectedIdx >= len(l.items) {
		return nil
	}
	return l.items[l.selectedIdx]
}

// SelectFirstInView 选择当前在视图中可见的第一个项目。
func (l *List) SelectFirstInView() {
	startIdx, _ := l.VisibleItemIndices()
	l.selectedIdx = startIdx
}

// SelectLastInView 选择当前在视图中可见的最后一个项目。
func (l *List) SelectLastInView() {
	_, endIdx := l.VisibleItemIndices()
	l.selectedIdx = endIdx
}

// ItemAt 返回给定索引处的项目。
func (l *List) ItemAt(index int) Item {
	if index < 0 || index >= len(l.items) {
		return nil
	}
	return l.items[index]
}

// ItemIndexAtPosition 返回给定视口相对y坐标处的项目。
// 返回项目索引和该项目内的y偏移。如果
// 没有找到项目，它返回-1, -1。
func (l *List) ItemIndexAtPosition(x, y int) (itemIdx int, itemY int) {
	return l.findItemAtY(x, y)
}

// findItemAtY 查找给定视口y坐标处的项目。
// 返回项目索引和该项目内的y偏移。如果
// 没有找到项目，它返回-1, -1。
func (l *List) findItemAtY(_, y int) (itemIdx int, itemY int) {
	if y < 0 || y >= l.height {
		return -1, -1
	}

	// 遍历可见项目以找到包含此y的项目
	currentIdx := l.offsetIdx
	currentLine := -l.offsetLine // 负数，因为offsetLine是隐藏的行数

	for currentIdx < len(l.items) && currentLine < l.height {
		item := l.getItem(currentIdx)
		itemEndLine := currentLine + item.height

		// 检查y是否在此项目的可见范围内
		if y >= currentLine && y < itemEndLine {
			// 找到项目，计算itemY（项目内的偏移）
			itemY = y - currentLine
			return currentIdx, itemY
		}

		// 移动到下一个项目
		currentLine = itemEndLine
		if l.gap > 0 {
			currentLine += l.gap
		}
		currentIdx++
	}

	return -1, -1
}
