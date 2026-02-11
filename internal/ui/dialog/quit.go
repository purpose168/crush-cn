package dialog

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/ui/common"
	uv "github.com/charmbracelet/ultraviolet"
)

// QuitID 是退出对话框的标识符。
const QuitID = "quit"

// Quit 表示退出应用程序的确认对话框。
type Quit struct {
	com        *common.Common
	selectedNo bool // 如果选择了"否"按钮则为 true
	keyMap     struct {
		LeftRight,
		EnterSpace,
		Yes,
		No,
		Tab,
		Close,
		Quit key.Binding
	}
}

var _ Dialog = (*Quit)(nil)

// NewQuit 创建一个新的退出确认对话框。
func NewQuit(com *common.Common) *Quit {
	q := &Quit{
		com:        com,
		selectedNo: true,
	}
	q.keyMap.LeftRight = key.NewBinding(
		key.WithKeys("left", "right"),
		key.WithHelp("←/→", "切换选项"),
	)
	q.keyMap.EnterSpace = key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter/space", "确认"),
	)
	q.keyMap.Yes = key.NewBinding(
		key.WithKeys("y", "Y", "ctrl+c"),
		key.WithHelp("y/Y/ctrl+c", "是"),
	)
	q.keyMap.No = key.NewBinding(
		key.WithKeys("n", "N"),
		key.WithHelp("n/N", "否"),
	)
	q.keyMap.Tab = key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "切换选项"),
	)
	q.keyMap.Close = CloseKey
	q.keyMap.Quit = key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "退出"),
	)
	return q
}

// ID 实现 [Model] 接口。
func (*Quit) ID() string {
	return QuitID
}

// HandleMsg 实现 [Model] 接口。
func (q *Quit) HandleMsg(msg tea.Msg) Action {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, q.keyMap.Quit):
			return ActionQuit{}
		case key.Matches(msg, q.keyMap.Close):
			return ActionClose{}
		case key.Matches(msg, q.keyMap.LeftRight, q.keyMap.Tab):
			q.selectedNo = !q.selectedNo
		case key.Matches(msg, q.keyMap.EnterSpace):
			if !q.selectedNo {
				return ActionQuit{}
			}
			return ActionClose{}
		case key.Matches(msg, q.keyMap.Yes):
			return ActionQuit{}
		case key.Matches(msg, q.keyMap.No, q.keyMap.Close):
			return ActionClose{}
		}
	}

	return nil
}

// Draw 实现 [Dialog] 接口。
func (q *Quit) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	const question = "您确定要退出吗？"
	baseStyle := q.com.Styles.Base
	buttonOpts := []common.ButtonOpts{
		{Text: "是！", Selected: !q.selectedNo, Padding: 3},
		{Text: "否", Selected: q.selectedNo, Padding: 3},
	}
	buttons := common.ButtonGroup(q.com.Styles, buttonOpts, " ")
	content := baseStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Center,
			question,
			"",
			buttons,
		),
	)

	view := q.com.Styles.BorderFocus.Render(content)
	DrawCenter(scr, area, view)
	return nil
}

// ShortHelp 实现 [help.KeyMap] 接口。
func (q *Quit) ShortHelp() []key.Binding {
	return []key.Binding{
		q.keyMap.LeftRight,
		q.keyMap.EnterSpace,
	}
}

// FullHelp 实现 [help.KeyMap] 接口。
func (q *Quit) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{q.keyMap.LeftRight, q.keyMap.EnterSpace, q.keyMap.Yes, q.keyMap.No},
		{q.keyMap.Tab, q.keyMap.Close},
	}
}
