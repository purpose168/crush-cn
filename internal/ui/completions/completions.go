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
	minHeight = 1
	maxHeight = 10
	minWidth  = 10
	maxWidth  = 100
)

// SelectionMsg 在选择补全时发送。
type SelectionMsg[T any] struct {
	Value    T
	KeepOpen bool // 如果为 true，则在插入后不关闭。
}

// ClosedMsg 在补全关闭时发送。
type ClosedMsg struct{}

// CompletionItemsLoadedMsg 在为补全加载文件时发送。
type CompletionItemsLoadedMsg struct {
	Files     []FileCompletionValue
	Resources []ResourceCompletionValue
}

// Completions 表示补全弹出组件。
type Completions struct {
	// 弹出尺寸
	width  int
	height int

	// 状态
	open  bool
	query string

	// 按键绑定
	keyMap KeyMap

	// 列表组件
	list *list.FilterableList

	// 样式
	normalStyle  lipgloss.Style
	focusedStyle lipgloss.Style
	matchStyle   lipgloss.Style
}

// New 创建一个新的补全组件。
func New(normalStyle, focusedStyle, matchStyle lipgloss.Style) *Completions {
	l := list.NewFilterableList()
	l.SetGap(0)
	l.SetReverse(true)

	return &Completions{
		keyMap:       DefaultKeyMap(),
		list:         l,
		normalStyle:  normalStyle,
		focusedStyle: focusedStyle,
		matchStyle:   matchStyle,
	}
}

// IsOpen 返回补全弹出窗口是否打开。
func (c *Completions) IsOpen() bool {
	return c.open
}

// Query 返回当前过滤查询。
func (c *Completions) Query() string {
	return c.query
}

// Size 返回弹出窗口的可见尺寸。
func (c *Completions) Size() (width, height int) {
	visible := len(c.list.FilteredItems())
	return c.width, min(visible, c.height)
}

// KeyMap 返回按键绑定。
func (c *Completions) KeyMap() KeyMap {
	return c.keyMap
}

// Open 使用来自文件系统的文件项目打开补全。
func (c *Completions) Open(depth, limit int) tea.Cmd {
	return func() tea.Msg {
		var msg CompletionItemsLoadedMsg
		var wg sync.WaitGroup
		wg.Go(func() {
			msg.Files = loadFiles(depth, limit)
		})
		wg.Go(func() {
			msg.Resources = loadMCPResources()
		})
		wg.Wait()
		return msg
	}
}

// SetItems 设置文件和 MCP 资源并重建合并列表。
func (c *Completions) SetItems(files []FileCompletionValue, resources []ResourceCompletionValue) {
	items := make([]list.FilterableItem, 0, len(files)+len(resources))

	// 首先添加文件。
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

	// 添加 MCP 资源。
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

// Close 关闭补全弹出窗口。
func (c *Completions) Close() {
	c.open = false
}

// Filter 使用给定查询过滤补全。
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

func (c *Completions) updateSize() {
	items := c.list.FilteredItems()
	start, end := c.list.VisibleItemIndices()
	width := 0
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

// HasItems 返回是否有可见项目。
func (c *Completions) HasItems() bool {
	return len(c.list.FilteredItems()) > 0
}

// Update 处理补全的按键事件。
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

// selectPrev 使用循环导航选择上一个项目。
func (c *Completions) selectPrev() {
	items := c.list.FilteredItems()
	if len(items) == 0 {
		return
	}
	if !c.list.SelectPrev() {
		c.list.WrapToEnd()
	}
	c.list.ScrollToSelected()
}

// selectNext 使用循环导航选择下一个项目。
func (c *Completions) selectNext() {
	items := c.list.FilteredItems()
	if len(items) == 0 {
		return
	}
	if !c.list.SelectNext() {
		c.list.WrapToStart()
	}
	c.list.ScrollToSelected()
}

// selectCurrent 返回一个带有当前选中项目的命令。
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

// Render 渲染补全弹出窗口。
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
