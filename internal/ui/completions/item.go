package completions

import (
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/ui/list"
	"github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
	"github.com/sahilm/fuzzy"
)

// FileCompletionValue 表示文件路径补全值。
type FileCompletionValue struct {
	Path string
}

// ResourceCompletionValue 表示 MCP 资源补全值。
type ResourceCompletionValue struct {
	MCPName  string
	URI      string
	Title    string
	MIMEType string
}

// CompletionItem 表示补全列表中的一个项目。
type CompletionItem struct {
	text    string
	value   any
	match   fuzzy.Match
	focused bool
	cache   map[int]string

	// 样式
	normalStyle  lipgloss.Style
	focusedStyle lipgloss.Style
	matchStyle   lipgloss.Style
}

// NewCompletionItem 创建一个新的补全项目。
func NewCompletionItem(text string, value any, normalStyle, focusedStyle, matchStyle lipgloss.Style) *CompletionItem {
	return &CompletionItem{
		text:         text,
		value:        value,
		normalStyle:  normalStyle,
		focusedStyle: focusedStyle,
		matchStyle:   matchStyle,
	}
}

// Text 返回项目的显示文本。
func (c *CompletionItem) Text() string {
	return c.text
}

// Value 返回项目的值。
func (c *CompletionItem) Value() any {
	return c.value
}

// Filter 实现 [list.FilterableItem] 接口。
func (c *CompletionItem) Filter() string {
	return c.text
}

// SetMatch 实现 [list.MatchSettable] 接口。
func (c *CompletionItem) SetMatch(m fuzzy.Match) {
	c.cache = nil
	c.match = m
}

// SetFocused 实现 [list.Focusable] 接口。
func (c *CompletionItem) SetFocused(focused bool) {
	if c.focused != focused {
		c.cache = nil
	}
	c.focused = focused
}

// Render 实现 [list.Item] 接口。
func (c *CompletionItem) Render(width int) string {
	return renderItem(
		c.normalStyle,
		c.focusedStyle,
		c.matchStyle,
		c.text,
		c.focused,
		width,
		c.cache,
		&c.match,
	)
}

func renderItem(
	normalStyle, focusedStyle, matchStyle lipgloss.Style,
	text string,
	focused bool,
	width int,
	cache map[int]string,
	match *fuzzy.Match,
) string {
	if cache == nil {
		cache = make(map[int]string)
	}

	cached, ok := cache[width]
	if ok {
		return cached
	}

	innerWidth := width - 2 // 考虑内边距
	// 如果需要则截断。
	if ansi.StringWidth(text) > innerWidth {
		text = ansi.Truncate(text, innerWidth, "…")
	}

	// 选择基本样式。
	style := normalStyle
	matchStyle = matchStyle.Background(style.GetBackground())
	if focused {
		style = focusedStyle
		matchStyle = matchStyle.Background(style.GetBackground())
	}

	// 渲染带有背景的完整宽度文本。
	content := style.Padding(0, 1).Width(width).Render(text)

	// 使用 StyleRanges 应用匹配高亮。
	if len(match.MatchedIndexes) > 0 {
		var ranges []lipgloss.Range
		for _, rng := range matchedRanges(match.MatchedIndexes) {
			start, stop := bytePosToVisibleCharPos(text, rng)
			// 为内边距空格偏移 1
			ranges = append(ranges, lipgloss.NewRange(start+1, stop+2, matchStyle))
		}
		content = lipgloss.StyleRanges(content, ranges...)
	}

	cache[width] = content
	return content
}

// matchedRanges 将匹配索引列表转换为连续范围。
func matchedRanges(in []int) [][2]int {
	if len(in) == 0 {
		return [][2]int{}
	}
	current := [2]int{in[0], in[0]}
	if len(in) == 1 {
		return [][2]int{current}
	}
	var out [][2]int
	for i := 1; i < len(in); i++ {
		if in[i] == current[1]+1 {
			current[1] = in[i]
		} else {
			out = append(out, current)
			current = [2]int{in[i], in[i]}
		}
	}
	out = append(out, current)
	return out
}

// bytePosToVisibleCharPos 将字节位置转换为可见字符位置。
func bytePosToVisibleCharPos(str string, rng [2]int) (int, int) {
	bytePos, byteStart, byteStop := 0, rng[0], rng[1]
	pos, start, stop := 0, 0, 0
	gr := uniseg.NewGraphemes(str)
	for byteStart > bytePos {
		if !gr.Next() {
			break
		}
		bytePos += len(gr.Str())
		pos += max(1, gr.Width())
	}
	start = pos
	for byteStop > bytePos {
		if !gr.Next() {
			break
		}
		bytePos += len(gr.Str())
		pos += max(1, gr.Width())
	}
	stop = pos
	return start, stop
}

// 确保 CompletionItem 实现所需的接口。
var (
	_ list.Item           = (*CompletionItem)(nil)
	_ list.FilterableItem = (*CompletionItem)(nil)
	_ list.MatchSettable  = (*CompletionItem)(nil)
	_ list.Focusable      = (*CompletionItem)(nil)
)
