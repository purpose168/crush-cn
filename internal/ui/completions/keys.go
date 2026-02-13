package completions

import (
	"charm.land/bubbles/v2/key"
)

// KeyMap 定义补全组件的按键绑定
// 包含所有用于导航和选择补全项目的按键绑定
type KeyMap struct {
	Down,       // 向下移动
	Up,         // 向上移动
	Select,     // 选择当前项目
	Cancel key.Binding // 取消补全
	DownInsert, // 向下移动并插入
	UpInsert key.Binding // 向上移动并插入
}

// DefaultKeyMap 返回补全的默认按键绑定
// 定义了标准的导航和选择按键
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("down", "向下移动"),
		),
		Up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("up", "向上移动"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter", "tab", "ctrl+y"),
			key.WithHelp("enter", "选择"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc", "alt+esc"),
			key.WithHelp("esc", "取消"),
		),
		DownInsert: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("ctrl+n", "插入下一个"),
		),
		UpInsert: key.NewBinding(
			key.WithKeys("ctrl+p"),
			key.WithHelp("ctrl+p", "插入上一个"),
		),
	}
}

// KeyBindings 以切片形式返回所有按键绑定
// 用于显示按键帮助信息
func (k KeyMap) KeyBindings() []key.Binding {
	return []key.Binding{
		k.Down,
		k.Up,
		k.Select,
		k.Cancel,
	}
}

// FullHelp 返回按键绑定的完整帮助
// 将按键绑定分组显示,每组最多 4 个
func (k KeyMap) FullHelp() [][]key.Binding {
	m := [][]key.Binding{}
	slice := k.KeyBindings()
	for i := 0; i < len(slice); i += 4 {
		end := min(i+4, len(slice))
		m = append(m, slice[i:end])
	}
	return m
}

// ShortHelp 返回按键绑定的简短帮助
// 只返回上下导航按键,用于简洁显示
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Up,
		k.Down,
	}
}
