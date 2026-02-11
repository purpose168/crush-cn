package dialog

import (
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/purpose168/crush-cn/internal/commands"
	"github.com/purpose168/crush-cn/internal/ui/common"
	"github.com/purpose168/crush-cn/internal/ui/util"
)

// ArgumentsID 是参数对话框的标识符。
const ArgumentsID = "arguments"

// 参数对话框的尺寸。
const (
	maxInputWidth        = 120
	minInputWidth        = 30
	maxViewportHeight    = 20
	argumentsFieldHeight = 3 // 每个字段的标签 + 输入 + 间距
)

// Arguments 表示一个用于收集命令参数的对话框。
type Arguments struct {
	com       *common.Common
	title     string
	arguments []commands.Argument
	inputs    []textinput.Model
	focused   int
	spinner   spinner.Model
	loading   bool

	description  string
	resultAction Action

	help   help.Model
	keyMap struct {
		Confirm,
		Next,
		Previous,
		ScrollUp,
		ScrollDown,
		Close key.Binding
	}

	viewport viewport.Model
}

var _ Dialog = (*Arguments)(nil)

// NewArguments 创建一个新的参数对话框。
func NewArguments(com *common.Common, title, description string, arguments []commands.Argument, resultAction Action) *Arguments {
	a := &Arguments{
		com:          com,
		title:        title,
		description:  description,
		arguments:    arguments,
		resultAction: resultAction,
	}

	a.help = help.New()
	a.help.Styles = com.Styles.DialogHelpStyles()

	a.keyMap.Confirm = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "确认"),
	)
	a.keyMap.Next = key.NewBinding(
		key.WithKeys("down", "tab"),
		key.WithHelp("↓/tab", "下一个"),
	)
	a.keyMap.Previous = key.NewBinding(
		key.WithKeys("up", "shift+tab"),
		key.WithHelp("↑/shift+tab", "上一个"),
	)
	a.keyMap.Close = CloseKey

	// 为每个参数创建输入字段。
	a.inputs = make([]textinput.Model, len(arguments))
	for i, arg := range arguments {
		input := textinput.New()
		input.SetVirtualCursor(false)
		input.SetStyles(com.Styles.TextInput)
		input.Prompt = "> "
		// 如果有描述，则使用描述作为占位符，否则使用标题
		if arg.Description != "" {
			input.Placeholder = arg.Description
		} else {
			input.Placeholder = arg.Title
		}

		if i == 0 {
			input.Focus()
		} else {
			input.Blur()
		}

		a.inputs[i] = input
	}
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = com.Styles.Dialog.Spinner
	a.spinner = s

	return a
}

// ID 实现 Dialog 接口。
func (a *Arguments) ID() string {
	return ArgumentsID
}

// focusInput 将焦点更改为新的输入字段，带有环绕功能。
func (a *Arguments) focusInput(newIndex int) {
	a.inputs[a.focused].Blur()

	// 环绕：Go 的取模可以返回负数，所以先加上长度。
	n := len(a.inputs)
	a.focused = ((newIndex % n) + n) % n

	a.inputs[a.focused].Focus()

	// 确保新聚焦的字段在视口中可见
	a.ensureFieldVisible(a.focused)
}

// isFieldVisible 检查给定索引处的字段是否在视口中可见。
func (a *Arguments) isFieldVisible(fieldIndex int) bool {
	fieldStart := fieldIndex * argumentsFieldHeight
	fieldEnd := fieldStart + argumentsFieldHeight - 1
	viewportTop := a.viewport.YOffset()
	viewportBottom := viewportTop + a.viewport.Height() - 1

	return fieldStart >= viewportTop && fieldEnd <= viewportBottom
}

// ensureFieldVisible 滚动视口以使字段可见。
func (a *Arguments) ensureFieldVisible(fieldIndex int) {
	if a.isFieldVisible(fieldIndex) {
		return
	}

	fieldStart := fieldIndex * argumentsFieldHeight
	fieldEnd := fieldStart + argumentsFieldHeight - 1
	viewportTop := a.viewport.YOffset()
	viewportHeight := a.viewport.Height()

	// 如果字段在视口上方，向上滚动以在顶部显示它
	if fieldStart < viewportTop {
		a.viewport.SetYOffset(fieldStart)
		return
	}

	// 如果字段在视口下方，向下滚动以在底部显示它
	if fieldEnd > viewportTop+viewportHeight-1 {
		a.viewport.SetYOffset(fieldEnd - viewportHeight + 1)
	}
}

// findVisibleFieldByOffset 返回最接近给定视口偏移量的字段索引。
func (a *Arguments) findVisibleFieldByOffset(fromTop bool) int {
	offset := a.viewport.YOffset()
	if !fromTop {
		offset += a.viewport.Height() - 1
	}

	fieldIndex := offset / argumentsFieldHeight
	if fieldIndex >= len(a.inputs) {
		return len(a.inputs) - 1
	}
	return fieldIndex
}

// HandleMsg 实现 Dialog 接口。
func (a *Arguments) HandleMsg(msg tea.Msg) Action {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if a.loading {
			var cmd tea.Cmd
			a.spinner, cmd = a.spinner.Update(msg)
			return ActionCmd{Cmd: cmd}
		}
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, a.keyMap.Close):
			return ActionClose{}
		case key.Matches(msg, a.keyMap.Confirm):
			// 如果我们在最后一个输入或只有一个输入，则提交。
			if a.focused == len(a.inputs)-1 || len(a.inputs) == 1 {
				args := make(map[string]string)
				var warning tea.Cmd
				for i, arg := range a.arguments {
					args[arg.ID] = a.inputs[i].Value()
					if arg.Required && strings.TrimSpace(a.inputs[i].Value()) == "" {
						warning = util.ReportWarn("必需参数 '" + arg.Title + "' 缺失。")
						break
					}
				}
				if warning != nil {
					return ActionCmd{Cmd: warning}
				}

				switch action := a.resultAction.(type) {
				case ActionRunCustomCommand:
					action.Args = args
					return action
				case ActionRunMCPPrompt:
					action.Args = args
					return action
				}
			}
			a.focusInput(a.focused + 1)
		case key.Matches(msg, a.keyMap.Next):
			a.focusInput(a.focused + 1)
		case key.Matches(msg, a.keyMap.Previous):
			a.focusInput(a.focused - 1)
		default:
			var cmd tea.Cmd
			a.inputs[a.focused], cmd = a.inputs[a.focused].Update(msg)
			return ActionCmd{Cmd: cmd}
		}
	case tea.MouseWheelMsg:
		a.viewport, _ = a.viewport.Update(msg)
		// 如果聚焦字段滚动出视图，聚焦可见字段
		if !a.isFieldVisible(a.focused) {
			a.focusInput(a.findVisibleFieldByOffset(msg.Button == tea.MouseWheelDown))
		}
	case tea.PasteMsg:
		var cmd tea.Cmd
		a.inputs[a.focused], cmd = a.inputs[a.focused].Update(msg)
		return ActionCmd{Cmd: cmd}
	}
	return nil
}

// Cursor 返回相对于对话框的光标位置。
// 我们传递描述高度以正确偏移光标。
func (a *Arguments) Cursor(descriptionHeight int) *tea.Cursor {
	cursor := InputCursor(a.com.Styles, a.inputs[a.focused].Cursor())
	if cursor == nil {
		return nil
	}
	cursor.Y += descriptionHeight + a.focused*argumentsFieldHeight - a.viewport.YOffset() + 1
	return cursor
}

// Draw 实现 Dialog 接口。
func (a *Arguments) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	s := a.com.Styles

	dialogContentStyle := s.Dialog.Arguments.Content
	possibleWidth := area.Dx() - s.Dialog.View.GetHorizontalFrameSize() - dialogContentStyle.GetHorizontalFrameSize()
	// 构建带有标签和输入的字段。
	caser := cases.Title(language.English)

	var fields []string
	for i, arg := range a.arguments {
		isFocused := i == a.focused

		// 尝试美化标题作为标签。
		title := strings.ReplaceAll(arg.Title, "_", " ")
		title = strings.ReplaceAll(title, "-", " ")
		titleParts := strings.Fields(title)
		for i, part := range titleParts {
			titleParts[i] = caser.String(strings.ToLower(part))
		}
		labelText := strings.Join(titleParts, " ")

		markRequiredStyle := s.Dialog.Arguments.InputRequiredMarkBlurred

		labelStyle := s.Dialog.Arguments.InputLabelBlurred
		if isFocused {
			labelStyle = s.Dialog.Arguments.InputLabelFocused
			markRequiredStyle = s.Dialog.Arguments.InputRequiredMarkFocused
		}
		if arg.Required {
			labelText += markRequiredStyle.String()
		}
		label := labelStyle.Render(labelText)

		labelWidth := lipgloss.Width(labelText)
		placeholderWidth := lipgloss.Width(a.inputs[i].Placeholder)

		inputWidth := max(placeholderWidth, labelWidth, minInputWidth)
		inputWidth = min(inputWidth, min(possibleWidth, maxInputWidth))
		a.inputs[i].SetWidth(inputWidth)

		inputLine := a.inputs[i].View()

		field := lipgloss.JoinVertical(lipgloss.Left, label, inputLine, "")
		fields = append(fields, field)
	}

	renderedFields := lipgloss.JoinVertical(lipgloss.Left, fields...)

	// 将宽度锚定到最长的字段，上限为 maxInputWidth。
	const scrollbarWidth = 1
	width := lipgloss.Width(renderedFields)
	height := lipgloss.Height(renderedFields)

	// 使用标准标题
	titleStyle := s.Dialog.Title

	titleText := a.title
	if titleText == "" {
		titleText = "参数"
	}

	header := common.DialogTitle(s, titleText, width, s.Primary, s.Secondary)

	// 如果有描述则添加。
	var description string
	if a.description != "" {
		descStyle := s.Dialog.Arguments.Description.Width(width)
		description = descStyle.Render(a.description)
	}

	helpView := s.Dialog.HelpView.Width(width).Render(a.help.View(a))
	if a.loading {
		helpView = s.Dialog.HelpView.Width(width).Render(a.spinner.View() + " 正在生成提示...")
	}

	availableHeight := area.Dy() - s.Dialog.View.GetVerticalFrameSize() - dialogContentStyle.GetVerticalFrameSize() - lipgloss.Height(header) - lipgloss.Height(description) - lipgloss.Height(helpView) - 2 // 额外间距
	viewportHeight := min(height, maxViewportHeight, availableHeight)

	a.viewport.SetWidth(width) // -1 用于滚动条
	a.viewport.SetHeight(viewportHeight)
	a.viewport.SetContent(renderedFields)

	scrollbar := common.Scrollbar(s, viewportHeight, a.viewport.TotalLineCount(), viewportHeight, a.viewport.YOffset())
	content := a.viewport.View()
	if scrollbar != "" {
		content = lipgloss.JoinHorizontal(lipgloss.Top, content, scrollbar)
	}
	var contentParts []string
	if description != "" {
		contentParts = append(contentParts, description)
	}
	contentParts = append(contentParts, content)

	view := lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render(header),
		dialogContentStyle.Render(lipgloss.JoinVertical(lipgloss.Left, contentParts...)),
		helpView,
	)

	dialog := s.Dialog.View.Render(view)

	descriptionHeight := 0
	if a.description != "" {
		descriptionHeight = lipgloss.Height(description)
	}
	cur := a.Cursor(descriptionHeight)

	DrawCenterCursor(scr, area, dialog, cur)
	return cur
}

// StartLoading 实现 [LoadingDialog] 接口。
func (a *Arguments) StartLoading() tea.Cmd {
	if a.loading {
		return nil
	}
	a.loading = true
	return a.spinner.Tick
}

// StopLoading 实现 [LoadingDialog] 接口。
func (a *Arguments) StopLoading() {
	a.loading = false
}

// ShortHelp 实现 help.KeyMap 接口。
func (a *Arguments) ShortHelp() []key.Binding {
	return []key.Binding{
		a.keyMap.Confirm,
		a.keyMap.Next,
		a.keyMap.Close,
	}
}

// FullHelp 实现 help.KeyMap 接口。
func (a *Arguments) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{a.keyMap.Confirm, a.keyMap.Next, a.keyMap.Previous},
		{a.keyMap.Close},
	}
}
