package dialog

import (
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/sahilm/fuzzy"
)

// CommandItem 包装一个 uicmd.Command 以实现 ListItem 接口。
type CommandItem struct {
	id       string
	title    string
	shortcut string
	action   Action
	t        *styles.Styles
	m        fuzzy.Match
	cache    map[int]string
	focused  bool
}

var _ ListItem = &CommandItem{}

// NewCommandItem 创建一个新的 CommandItem。
func NewCommandItem(t *styles.Styles, id, title, shortcut string, action Action) *CommandItem {
	return &CommandItem{
		id:       id,
		t:        t,
		title:    title,
		shortcut: shortcut,
		action:   action,
	}
}

// Filter 实现 ListItem 接口。
func (c *CommandItem) Filter() string {
	return c.title
}

// ID 实现 ListItem 接口。
func (c *CommandItem) ID() string {
	return c.id
}

// SetFocused 实现 ListItem 接口。
func (c *CommandItem) SetFocused(focused bool) {
	if c.focused != focused {
		c.cache = nil
	}
	c.focused = focused
}

// SetMatch 实现 ListItem 接口。
func (c *CommandItem) SetMatch(m fuzzy.Match) {
	c.cache = nil
	c.m = m
}

// Action 返回与命令项目关联的操作。
func (c *CommandItem) Action() Action {
	return c.action
}

// Shortcut 返回与命令项目关联的快捷键。
func (c *CommandItem) Shortcut() string {
	return c.shortcut
}

// Render 实现 ListItem 接口。
func (c *CommandItem) Render(width int) string {
	styles := ListItemStyles{
		ItemBlurred:     c.t.Dialog.NormalItem,
		ItemFocused:     c.t.Dialog.SelectedItem,
		InfoTextBlurred: c.t.Base,
		InfoTextFocused: c.t.Base,
	}
	return renderItem(styles, c.title, c.shortcut, c.focused, width, c.cache, &c.m)
}
