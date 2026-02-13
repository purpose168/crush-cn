package completions

import (
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/purpose168/crush-cn/internal/ui/list"
	"github.com/rivo/uniseg"
	"github.com/sahilm/fuzzy"
)

// FileCompletionValue 表示文件路径补全值
// 用于存储文件路径信息,作为补全列表中的项目值
type FileCompletionValue struct {
	Path string // 文件路径
}

// ResourceCompletionValue 表示 MCP 资源补全值
// 用于存储 MCP (Model Context Protocol) 资源信息
type ResourceCompletionValue struct {
	MCPName  string // MCP 服务器名称
	URI      string // 资源 URI
	Title    string // 资源标题
	MIMEType string // 资源的 MIME 类型
}

// CompletionItem 表示补全列表中的一个项目
// 实现了 list.Item, list.FilterableItem, list.MatchSettable 和 list.Focusable 接口
type CompletionItem struct {
	text    string       // 显示文本
	value   any          // 项目值(可以是 FileCompletionValue 或 ResourceCompletionValue)
	match   fuzzy.Match  // 模糊匹配结果
	focused bool         // 是否获得焦点
	cache   map[int]string // 渲染缓存,按宽度缓存渲染结果

	// 样式定义
	normalStyle  lipgloss.Style // 普通状态样式
	focusedStyle lipgloss.Style // 焦点状态样式
	matchStyle   lipgloss.Style // 匹配文本高亮样式
}

// NewCompletionItem 创建一个新的补全项目
// 参数:
//   - text: 显示文本
//   - value: 项目值
//   - normalStyle: 普通状态样式
//   - focusedStyle: 焦点状态样式
//   - matchStyle: 匹配文本高亮样式
// 返回初始化后的补全项目指针
func NewCompletionItem(text string, value any, normalStyle, focusedStyle, matchStyle lipgloss.Style) *CompletionItem {
	return &CompletionItem{
		text:         text,
		value:        value,
		normalStyle:  normalStyle,
		focusedStyle: focusedStyle,
		matchStyle:   matchStyle,
	}
}

// Text 返回项目的显示文本
func (c *CompletionItem) Text() string {
	return c.text
}

// Value 返回项目的值
func (c *CompletionItem) Value() any {
	return c.value
}

// Filter 实现 [list.FilterableItem] 接口
// 返回用于过滤的字符串
func (c *CompletionItem) Filter() string {
	return c.text
}

// SetMatch 实现 [list.MatchSettable] 接口
// 设置模糊匹配结果,并清除缓存
func (c *CompletionItem) SetMatch(m fuzzy.Match) {
	c.cache = nil
	c.match = m
}

// SetFocused 实现 [list.Focusable] 接口
// 设置焦点状态,状态改变时清除缓存
func (c *CompletionItem) SetFocused(focused bool) {
	if c.focused != focused {
		c.cache = nil
	}
	c.focused = focused
}

// Render 实现 [list.Item] 接口
// 渲染项目到指定宽度
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

// renderItem 渲染补全项目
// 参数:
//   - normalStyle: 普通状态样式
//   - focusedStyle: 焦点状态样式
//   - matchStyle: 匹配文本高亮样式
//   - text: 显示文本
//   - focused: 是否获得焦点
//   - width: 渲染宽度
//   - cache: 渲染缓存
//   - match: 模糊匹配结果
// 返回渲染后的字符串
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

	// 检查缓存
	cached, ok := cache[width]
	if ok {
		return cached
	}

	innerWidth := width - 2 // 考虑内边距
	// 如果需要则截断文本
	if ansi.StringWidth(text) > innerWidth {
		text = ansi.Truncate(text, innerWidth, "…")
	}

	// 选择基本样式
	style := normalStyle
	matchStyle = matchStyle.Background(style.GetBackground())
	if focused {
		style = focusedStyle
		matchStyle = matchStyle.Background(style.GetBackground())
	}

	// 渲染带有背景的完整宽度文本
	content := style.Padding(0, 1).Width(width).Render(text)

	// 使用 StyleRanges 应用匹配高亮
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

// matchedRanges 将匹配索引列表转换为连续范围
// 参数:
//   - in: 匹配索引列表
// 返回连续的范围列表,每个范围包含起始和结束索引
func matchedRanges(in []int) [][2]int {
	if len(in) == 0 {
		return [][2]int{}
	}
	current := [2]int{in[0], in[0]}
	if len(in) == 1 {
		return [][2]int{current}
	}
	var out [][2]int
	// 合并连续的索引为范围
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

// bytePosToVisibleCharPos 将字节位置转换为可见字符位置
// 用于正确处理多字节字符(如中文)的位置计算
// 参数:
//   - str: 源字符串
//   - rng: 字节范围 [起始, 结束]
// 返回可见字符的起始和结束位置
func bytePosToVisibleCharPos(str string, rng [2]int) (int, int) {
	bytePos, byteStart, byteStop := 0, rng[0], rng[1]
	pos, start, stop := 0, 0, 0
	gr := uniseg.NewGraphemes(str)
	// 查找起始位置
	for byteStart > bytePos {
		if !gr.Next() {
			break
		}
		bytePos += len(gr.Str())
		pos += max(1, gr.Width())
	}
	start = pos
	// 查找结束位置
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

// 确保 CompletionItem 实现所需的接口
var (
	_ list.Item           = (*CompletionItem)(nil) // 列表项目接口
	_ list.FilterableItem = (*CompletionItem)(nil) // 可过滤项目接口
	_ list.MatchSettable  = (*CompletionItem)(nil) // 可设置匹配接口
	_ list.Focusable      = (*CompletionItem)(nil) // 可聚焦接口
)
