// Package completions 提供补全弹出组件的实现
// 该包实现了一个可过滤的补全列表,支持文件路径和 MCP 资源的补全功能
package completions

import (
	"cmp"
	"slices"
	"strings"
	"sync"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/ordered"
	"github.com/purpose168/crush-cn/internal/agent/tools/mcp"
	"github.com/purpose168/crush-cn/internal/fsext"
	"github.com/purpose168/crush-cn/internal/ui/list"
)

const (
	minHeight = 1  // 弹出窗口最小高度
	maxHeight = 10 // 弹出窗口最大高度
	minWidth  = 10 // 弹出窗口最小宽度
	maxWidth  = 100 // 弹出窗口最大宽度
)

// SelectionMsg 在选择补全时发送的消息
// T 是补全值的类型参数
type SelectionMsg[T any] struct {
	Value    T      // 选中的补全值
	KeepOpen bool   // 如果为 true,则在插入后不关闭补全窗口
}

// ClosedMsg 在补全窗口关闭时发送的消息
type ClosedMsg struct{}

// CompletionItemsLoadedMsg 在补全项目加载完成时发送的消息
// 包含从文件系统和 MCP 资源加载的补全项目
type CompletionItemsLoadedMsg struct {
	Files     []FileCompletionValue     // 文件补全项目列表
	Resources []ResourceCompletionValue // MCP 资源补全项目列表
}

// Completions 表示补全弹出组件
// 该组件提供了一个可过滤的补全列表,支持文件路径和 MCP 资源的补全
type Completions struct {
	// 弹出窗口尺寸
	width  int // 弹出窗口宽度
	height int // 弹出窗口高度

	// 组件状态
	open  bool   // 补全窗口是否打开
	query string // 当前过滤查询字符串

	// 按键绑定
	keyMap KeyMap

	// 列表组件
	list *list.FilterableList

	// 样式定义
	normalStyle  lipgloss.Style // 普通状态样式
	focusedStyle lipgloss.Style // 焦点状态样式
	matchStyle   lipgloss.Style // 匹配文本高亮样式
}

// New 创建一个新的补全组件
// 参数:
//   - normalStyle: 普通状态样式
//   - focusedStyle: 焦点状态样式
//   - matchStyle: 匹配文本高亮样式
// 返回初始化后的补全组件指针
func New(normalStyle, focusedStyle, matchStyle lipgloss.Style) *Completions {
	l := list.NewFilterableList()
	l.SetGap(0)      // 设置列表项间距为 0
	l.SetReverse(true) // 设置列表反向显示

	return &Completions{
		keyMap:       DefaultKeyMap(),
		list:         l,
		normalStyle:  normalStyle,
		focusedStyle: focusedStyle,
		matchStyle:   matchStyle,
	}
}

// IsOpen 返回补全弹出窗口是否打开
func (c *Completions) IsOpen() bool {
	return c.open
}

// Query 返回当前过滤查询字符串
func (c *Completions) Query() string {
	return c.query
}

// Size 返回弹出窗口的可见尺寸
// 返回值:
//   - width: 弹出窗口宽度
//   - height: 弹出窗口高度(根据可见项目数量计算)
func (c *Completions) Size() (width, height int) {
	visible := len(c.list.FilteredItems())
	return c.width, min(visible, c.height)
}

// KeyMap 返回按键绑定配置
func (c *Completions) KeyMap() KeyMap {
	return c.keyMap
}

// Open 使用来自文件系统的文件项目打开补全
// 参数:
//   - depth: 文件系统遍历深度
//   - limit: 文件数量限制
// 返回一个命令,用于异步加载补全项目
func (c *Completions) Open(depth, limit int) tea.Cmd {
	return func() tea.Msg {
		var msg CompletionItemsLoadedMsg
		var wg sync.WaitGroup
		// 并发加载文件
		wg.Go(func() {
			msg.Files = loadFiles(depth, limit)
		})
		// 并发加载 MCP 资源
		wg.Go(func() {
			msg.Resources = loadMCPResources()
		})
		wg.Wait()
		return msg
	}
}

// SetItems 设置文件和 MCP 资源并重建合并列表
// 参数:
//   - files: 文件补全项目列表
//   - resources: MCP 资源补全项目列表
func (c *Completions) SetItems(files []FileCompletionValue, resources []ResourceCompletionValue) {
	items := make([]list.FilterableItem, 0, len(files)+len(resources))

	// 首先添加文件项目
	for _, file := range files {
		item := NewCompletionItem(
			file.Path,
			file,
			c.normalStyle,
			c.focusedStyle,
			c.matchStyle,
		)
		items = append(items, item)
	}

	// 添加 MCP 资源项目
	for _, resource := range resources {
		item := NewCompletionItem(
			resource.MCPName+"/"+cmp.Or(resource.Title, resource.URI),
			resource,
			c.normalStyle,
			c.focusedStyle,
			c.matchStyle,
		)
		items = append(items, item)
	}

	c.open = true
	c.query = ""
	c.list.SetItems(items...)
	c.list.SetFilter("")
	c.list.Focus()

	c.width = maxWidth
	c.height = ordered.Clamp(len(items), int(minHeight), int(maxHeight))
	c.list.SetSize(c.width, c.height)
	c.list.SelectFirst()
	c.list.ScrollToSelected()

	c.updateSize()
}

// Close 关闭补全弹出窗口
func (c *Completions) Close() {
	c.open = false
}

// Filter 使用给定查询过滤补全项目
// 参数:
//   - query: 过滤查询字符串
func (c *Completions) Filter(query string) {
	if !c.open {
		return
	}

	if query == c.query {
		return
	}

	c.query = query
	c.list.SetFilter(query)

	c.updateSize()
}

// updateSize 根据可见项目更新弹出窗口尺寸
func (c *Completions) updateSize() {
	items := c.list.FilteredItems()
	start, end := c.list.VisibleItemIndices()
	width := 0
	// 计算可见项目的最大宽度
	for i := start; i <= end; i++ {
		item := c.list.ItemAt(i)
		if item == nil {
			continue
		}
		s := item.(interface{ Text() string }).Text()
		width = max(width, ansi.StringWidth(s))
	}
	c.width = ordered.Clamp(width+2, int(minWidth), int(maxWidth))
	c.height = ordered.Clamp(len(items), int(minHeight), int(maxHeight))
	c.list.SetSize(c.width, c.height)
	c.list.SelectFirst()
	c.list.ScrollToSelected()
}

// HasItems 返回是否有可见项目
func (c *Completions) HasItems() bool {
	return len(c.list.FilteredItems()) > 0
}

// Update 处理补全组件的按键事件
// 参数:
//   - msg: 按键消息
// 返回值:
//   - tea.Msg: 生成的消息(如果有)
//   - bool: 是否处理了该按键
func (c *Completions) Update(msg tea.KeyPressMsg) (tea.Msg, bool) {
	if !c.open {
		return nil, false
	}

	switch {
	case key.Matches(msg, c.keyMap.Up):
		c.selectPrev()
		return nil, true

	case key.Matches(msg, c.keyMap.Down):
		c.selectNext()
		return nil, true

	case key.Matches(msg, c.keyMap.UpInsert):
		c.selectPrev()
		return c.selectCurrent(true), true

	case key.Matches(msg, c.keyMap.DownInsert):
		c.selectNext()
		return c.selectCurrent(true), true

	case key.Matches(msg, c.keyMap.Select):
		return c.selectCurrent(false), true

	case key.Matches(msg, c.keyMap.Cancel):
		c.Close()
		return ClosedMsg{}, true
	}

	return nil, false
}

// selectPrev 使用循环导航选择上一个项目
func (c *Completions) selectPrev() {
	items := c.list.FilteredItems()
	if len(items) == 0 {
		return
	}
	if !c.list.SelectPrev() {
		c.list.WrapToEnd() // 循环到末尾
	}
	c.list.ScrollToSelected()
}

// selectNext 使用循环导航选择下一个项目
func (c *Completions) selectNext() {
	items := c.list.FilteredItems()
	if len(items) == 0 {
		return
	}
	if !c.list.SelectNext() {
		c.list.WrapToStart() // 循环到开头
	}
	c.list.ScrollToSelected()
}

// selectCurrent 返回一个带有当前选中项目的消息
// 参数:
//   - keepOpen: 是否保持补全窗口打开
// 返回选中的补全项目消息
func (c *Completions) selectCurrent(keepOpen bool) tea.Msg {
	items := c.list.FilteredItems()
	if len(items) == 0 {
		return nil
	}

	selected := c.list.Selected()
	if selected < 0 || selected >= len(items) {
		return nil
	}

	item, ok := items[selected].(*CompletionItem)
	if !ok {
		return nil
	}

	if !keepOpen {
		c.open = false
	}

	// 根据值的类型返回相应的选择消息
	switch item := item.Value().(type) {
	case ResourceCompletionValue:
		return SelectionMsg[ResourceCompletionValue]{
			Value:    item,
			KeepOpen: keepOpen,
		}
	case FileCompletionValue:
		return SelectionMsg[FileCompletionValue]{
			Value:    item,
			KeepOpen: keepOpen,
		}
	default:
		return nil
	}
}

// Render 渲染补全弹出窗口
// 返回渲染后的字符串,如果窗口关闭或无项目则返回空字符串
func (c *Completions) Render() string {
	if !c.open {
		return ""
	}

	items := c.list.FilteredItems()
	if len(items) == 0 {
		return ""
	}

	return c.list.Render()
}

// loadFiles 从当前目录加载文件列表
// 参数:
//   - depth: 遍历深度
//   - limit: 文件数量限制
// 返回文件补全值列表
func loadFiles(depth, limit int) []FileCompletionValue {
	files, _, _ := fsext.ListDirectory(".", nil, depth, limit)
	slices.Sort(files)
	result := make([]FileCompletionValue, 0, len(files))
	for _, file := range files {
		result = append(result, FileCompletionValue{
			Path: strings.TrimPrefix(file, "./"),
		})
	}
	return result
}

// loadMCPResources 从 MCP 服务器加载资源列表
// 返回 MCP 资源补全值列表
func loadMCPResources() []ResourceCompletionValue {
	var resources []ResourceCompletionValue
	for mcpName, mcpResources := range mcp.Resources() {
		for _, r := range mcpResources {
			resources = append(resources, ResourceCompletionValue{
				MCPName:  mcpName,
				URI:      r.URI,
				Title:    r.Name,
				MIMEType: r.MIMEType,
			})
		}
	}
	return resources
}
