package model

import "charm.land/bubbles/v2/key"

// KeyMap 表示应用程序的按键映射结构
type KeyMap struct {
	// Editor 编辑器相关按键映射
	Editor struct {
		AddFile     key.Binding // 添加文件
		SendMessage key.Binding // 发送消息
		OpenEditor  key.Binding // 打开编辑器
		Newline     key.Binding // 换行
		AddImage    key.Binding // 添加图片
		PasteImage  key.Binding // 粘贴图片
		MentionFile key.Binding // 提及文件
		Commands    key.Binding // 命令

		// Attachments key maps 附件相关按键映射
		AttachmentDeleteMode key.Binding // 附件删除模式
		Escape               key.Binding // 退出
		DeleteAllAttachments key.Binding // 删除所有附件

		// History navigation 历史记录导航
		HistoryPrev key.Binding // 上一条历史记录
		HistoryNext key.Binding // 下一条历史记录
	}

	// Chat 聊天相关按键映射
	Chat struct {
		NewSession     key.Binding // 新建会话
		AddAttachment  key.Binding // 添加附件
		Cancel         key.Binding // 取消
		Tab            key.Binding // 切换
		Details        key.Binding // 详情
		TogglePills    key.Binding // 切换药丸视图
		PillLeft       key.Binding // 药丸左移
		PillRight      key.Binding // 药丸右移
		Down           key.Binding // 向下
		Up             key.Binding // 向上
		UpDown         key.Binding // 上下移动
		DownOneItem    key.Binding // 向下移动一项
		UpOneItem      key.Binding // 向上移动一项
		UpDownOneItem  key.Binding // 上下移动一项
		PageDown       key.Binding // 向下翻页
		PageUp         key.Binding // 向上翻页
		HalfPageDown   key.Binding // 向下翻半页
		HalfPageUp     key.Binding // 向上翻半页
		Home           key.Binding // 首页
		End            key.Binding // 末页
		Copy           key.Binding // 复制
		ClearHighlight key.Binding // 清除高亮
		Expand         key.Binding // 展开
	}

	// Initialize 初始化相关按键映射
	Initialize struct {
		Yes,
		No,
		Enter,
		Switch key.Binding // 切换
	}

	// Global key maps 全局按键映射
	Quit     key.Binding // 退出
	Help     key.Binding // 帮助
	Commands key.Binding // 命令
	Models   key.Binding // 模型
	Suspend  key.Binding // 挂起
	Sessions key.Binding // 会话
	Tab      key.Binding // 切换焦点
}

// DefaultKeyMap 返回默认的按键映射
func DefaultKeyMap() KeyMap {
	km := KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "退出"),
		),
		Help: key.NewBinding(
			key.WithKeys("ctrl+g"),
			key.WithHelp("ctrl+g", "更多"),
		),
		Commands: key.NewBinding(
			key.WithKeys("ctrl+p"),
			key.WithHelp("ctrl+p", "命令"),
		),
		Models: key.NewBinding(
			key.WithKeys("ctrl+m", "ctrl+l"),
			key.WithHelp("ctrl+l", "模型"),
		),
		Suspend: key.NewBinding(
			key.WithKeys("ctrl+z"),
			key.WithHelp("ctrl+z", "挂起"),
		),
		Sessions: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "会话"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "切换焦点"),
		),
	}

	km.Editor.AddFile = key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "添加文件"),
	)
	km.Editor.SendMessage = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "发送"),
	)
	km.Editor.OpenEditor = key.NewBinding(
		key.WithKeys("ctrl+o"),
		key.WithHelp("ctrl+o", "打开编辑器"),
	)
	km.Editor.Newline = key.NewBinding(
		key.WithKeys("shift+enter", "ctrl+j"),
		// "ctrl+j" 是许多编辑器中常见的换行快捷键。如果
		// 终端支持 "shift+enter"，我们会替换帮助文本
		// 以反映这一点。
		key.WithHelp("ctrl+j", "换行"),
	)
	km.Editor.AddImage = key.NewBinding(
		key.WithKeys("ctrl+f"),
		key.WithHelp("ctrl+f", "添加图片"),
	)
	km.Editor.PasteImage = key.NewBinding(
		key.WithKeys("ctrl+v"),
		key.WithHelp("ctrl+v", "从剪贴板粘贴图片"),
	)
	km.Editor.MentionFile = key.NewBinding(
		key.WithKeys("@"),
		key.WithHelp("@", "提及文件"),
	)
	km.Editor.Commands = key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "命令"),
	)
	km.Editor.AttachmentDeleteMode = key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r+{i}", "删除索引i处的附件"),
	)
	km.Editor.Escape = key.NewBinding(
		key.WithKeys("esc", "alt+esc"),
		key.WithHelp("esc", "取消删除模式"),
	)
	km.Editor.DeleteAllAttachments = key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("ctrl+r+r", "删除所有附件"),
	)
	km.Editor.HistoryPrev = key.NewBinding(
		key.WithKeys("up"),
	)
	km.Editor.HistoryNext = key.NewBinding(
		key.WithKeys("down"),
	)

	km.Chat.NewSession = key.NewBinding(
		key.WithKeys("ctrl+n"),
		key.WithHelp("ctrl+n", "新建会话"),
	)
	km.Chat.AddAttachment = key.NewBinding(
		key.WithKeys("ctrl+f"),
		key.WithHelp("ctrl+f", "添加附件"),
	)
	km.Chat.Cancel = key.NewBinding(
		key.WithKeys("esc", "alt+esc"),
		key.WithHelp("esc", "取消"),
	)
	km.Chat.Tab = key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "切换焦点"),
	)
	km.Chat.Details = key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "切换详情"),
	)
	km.Chat.TogglePills = key.NewBinding(
		key.WithKeys("ctrl+space"),
		key.WithHelp("ctrl+space", "切换任务"),
	)
	km.Chat.PillLeft = key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("←/→", "切换章节"),
	)
	km.Chat.PillRight = key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("←/→", "切换章节"),
	)

	km.Chat.Down = key.NewBinding(
		key.WithKeys("down", "ctrl+j", "j"),
		key.WithHelp("↓", "向下"),
	)
	km.Chat.Up = key.NewBinding(
		key.WithKeys("up", "ctrl+k", "k"),
		key.WithHelp("↑", "向上"),
	)
	km.Chat.UpDown = key.NewBinding(
		key.WithKeys("up", "down"),
		key.WithHelp("↑↓", "滚动"),
	)
	km.Chat.UpOneItem = key.NewBinding(
		key.WithKeys("shift+up", "K"),
		key.WithHelp("shift+↑", "向上移动一项"),
	)
	km.Chat.DownOneItem = key.NewBinding(
		key.WithKeys("shift+down", "J"),
		key.WithHelp("shift+↓", "向下移动一项"),
	)
	km.Chat.UpDownOneItem = key.NewBinding(
		key.WithKeys("shift+up", "shift+down"),
		key.WithHelp("shift+↑↓", "滚动一项"),
	)
	km.Chat.HalfPageDown = key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "向下翻半页"),
	)
	km.Chat.PageDown = key.NewBinding(
		key.WithKeys("pgdown", " ", "f"),
		key.WithHelp("f/pgdn", "向下翻页"),
	)
	km.Chat.PageUp = key.NewBinding(
		key.WithKeys("pgup", "b"),
		key.WithHelp("b/pgup", "向上翻页"),
	)
	km.Chat.HalfPageUp = key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "向上翻半页"),
	)
	km.Chat.Home = key.NewBinding(
		key.WithKeys("g", "home"),
		key.WithHelp("g", "首页"),
	)
	km.Chat.End = key.NewBinding(
		key.WithKeys("G", "end"),
		key.WithHelp("G", "末页"),
	)
	km.Chat.Copy = key.NewBinding(
		key.WithKeys("c", "y", "C", "Y"),
		key.WithHelp("c/y", "复制"),
	)
	km.Chat.ClearHighlight = key.NewBinding(
		key.WithKeys("esc", "alt+esc"),
		key.WithHelp("esc", "清除选择"),
	)
	km.Chat.Expand = key.NewBinding(
		key.WithKeys("space"),
		key.WithHelp("space", "展开/折叠"),
	)
	km.Initialize.Yes = key.NewBinding(
		key.WithKeys("y", "Y"),
		key.WithHelp("y", "是"),
	)
	km.Initialize.No = key.NewBinding(
		key.WithKeys("n", "N", "esc", "alt+esc"),
		key.WithHelp("n", "否"),
	)
	km.Initialize.Switch = key.NewBinding(
		key.WithKeys("left", "right", "tab"),
		key.WithHelp("tab", "切换"),
	)
	km.Initialize.Enter = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "选择"),
	)

	return km
}
