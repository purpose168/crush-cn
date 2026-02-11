package list

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Item 表示懒加载列表中的单个项目。
type Item interface {
	// Render 返回给定宽度的项目的字符串表示。
	Render(width int) string
}

// RawRenderable 表示一个可以提供原始渲染
// 而无需额外样式的项目。
type RawRenderable interface {
	// RawRender 返回没有任何额外样式的原始渲染字符串。
	RawRender(width int) string
}

// Focusable 表示一个可以感知焦点状态变化的项目。
type Focusable interface {
	// SetFocused 设置项目的焦点状态。
	SetFocused(focused bool)
}

// Highlightable 表示一个可以高亮其内容一部分的项目。
type Highlightable interface {
	// SetHighlight 从给定的起始到结束位置高亮内容。
	// 使用-1表示不进行高亮。
	SetHighlight(startLine, startCol, endLine, endCol int)
	// Highlight 返回项目中的当前高亮位置。
	Highlight() (startLine, startCol, endLine, endCol int)
}

// MouseClickable 表示一个可以处理鼠标点击事件的项目。
type MouseClickable interface {
	// HandleMouseClick 处理给定坐标处的鼠标点击事件。
	// 如果事件被处理则返回true，否则返回false。
	HandleMouseClick(btn ansi.MouseButton, x, y int) bool
}

// SpacerItem 是一个间隔项目，它在列表中添加垂直空间。
type SpacerItem struct {
	Height int
}

// NewSpacerItem 创建一个具有指定高度的新[SpacerItem]。
func NewSpacerItem(height int) *SpacerItem {
	return &SpacerItem{
		Height: max(0, height-1),
	}
}

// Render 为[SpacerItem]实现Item接口。
func (s *SpacerItem) Render(width int) string {
	return strings.Repeat("\n", s.Height)
}
