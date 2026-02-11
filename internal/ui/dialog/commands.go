package dialog

import (
	"os"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/purpose168/crush-cn/internal/commands"
	"github.com/purpose168/crush-cn/internal/config"
	"github.com/purpose168/crush-cn/internal/ui/common"
	"github.com/purpose168/crush-cn/internal/ui/list"
	"github.com/purpose168/crush-cn/internal/ui/styles"
)

// CommandsID 是命令对话框的标识符。
const CommandsID = "commands"

// CommandType 表示正在显示的命令类型。
type CommandType uint

// String 返回 CommandType 的字符串表示。
func (c CommandType) String() string { return []string{"System", "User", "MCP"}[c] }

const (
	sidebarCompactModeBreakpoint   = 120
	defaultCommandsDialogMaxHeight = 20
	defaultCommandsDialogMaxWidth  = 70
)

const (
	SystemCommands CommandType = iota
	UserCommands
	MCPPrompts
)

// Commands 表示一个显示可用命令的对话框。
type Commands struct {
	com    *common.Common
	keyMap struct {
		Select,
		UpDown,
		Next,
		Previous,
		Tab,
		ShiftTab,
		Close key.Binding
	}

	sessionID string // 对于非会话特定命令可以为空
	selected  CommandType

	spinner spinner.Model
	loading bool

	help  help.Model
	input textinput.Model
	list  *list.FilterableList

	windowWidth int

	customCommands []commands.CustomCommand
	mcpPrompts     []commands.MCPPrompt
}

var _ Dialog = (*Commands)(nil)

// NewCommands 创建一个新的命令对话框。
func NewCommands(com *common.Common, sessionID string, customCommands []commands.CustomCommand, mcpPrompts []commands.MCPPrompt) (*Commands, error) {
	c := &Commands{
		com:            com,
		selected:       SystemCommands,
		sessionID:      sessionID,
		customCommands: customCommands,
		mcpPrompts:     mcpPrompts,
	}

	help := help.New()
	help.Styles = com.Styles.DialogHelpStyles()

	c.help = help

	c.list = list.NewFilterableList()
	c.list.Focus()
	c.list.SetSelected(0)

	c.input = textinput.New()
	c.input.SetVirtualCursor(false)
	c.input.Placeholder = "输入以过滤"
	c.input.SetStyles(com.Styles.TextInput)
	c.input.Focus()

	c.keyMap.Select = key.NewBinding(
		key.WithKeys("enter", "ctrl+y"),
		key.WithHelp("enter", "确认"),
	)
	c.keyMap.UpDown = key.NewBinding(
		key.WithKeys("up", "down"),
		key.WithHelp("↑/↓", "选择"),
	)
	c.keyMap.Next = key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "下一项"),
	)
	c.keyMap.Previous = key.NewBinding(
		key.WithKeys("up", "ctrl+p"),
		key.WithHelp("↑", "上一项"),
	)
	c.keyMap.Tab = key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "切换选择"),
	)
	c.keyMap.ShiftTab = key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "切换选择（上一个）"),
	)
	closeKey := CloseKey
	closeKey.SetHelp("esc", "取消")
	c.keyMap.Close = closeKey

	// 设置初始命令
	c.setCommandItems(c.selected)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = com.Styles.Dialog.Spinner
	c.spinner = s

	return c, nil
}

// ID 实现 Dialog 接口。
func (c *Commands) ID() string {
	return CommandsID
}

// HandleMsg 实现 [Dialog] 接口。
func (c *Commands) HandleMsg(msg tea.Msg) Action {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if c.loading {
			var cmd tea.Cmd
			c.spinner, cmd = c.spinner.Update(msg)
			return ActionCmd{Cmd: cmd}
		}
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, c.keyMap.Close):
			return ActionClose{}
		case key.Matches(msg, c.keyMap.Previous):
			c.list.Focus()
			if c.list.IsSelectedFirst() {
				c.list.SelectLast()
				c.list.ScrollToBottom()
				break
			}
			c.list.SelectPrev()
			c.list.ScrollToSelected()
		case key.Matches(msg, c.keyMap.Next):
			c.list.Focus()
			if c.list.IsSelectedLast() {
				c.list.SelectFirst()
				c.list.ScrollToTop()
				break
			}
			c.list.SelectNext()
			c.list.ScrollToSelected()
		case key.Matches(msg, c.keyMap.Select):
			if selectedItem := c.list.SelectedItem(); selectedItem != nil {
				if item, ok := selectedItem.(*CommandItem); ok && item != nil {
					return item.Action()
				}
			}
		case key.Matches(msg, c.keyMap.Tab):
			if len(c.customCommands) > 0 || len(c.mcpPrompts) > 0 {
				c.selected = c.nextCommandType()
				c.setCommandItems(c.selected)
			}
		case key.Matches(msg, c.keyMap.ShiftTab):
			if len(c.customCommands) > 0 || len(c.mcpPrompts) > 0 {
				c.selected = c.previousCommandType()
				c.setCommandItems(c.selected)
			}
		default:
			var cmd tea.Cmd
			for _, item := range c.list.FilteredItems() {
				if item, ok := item.(*CommandItem); ok && item != nil {
					if msg.String() == item.Shortcut() {
						return item.Action()
					}
				}
			}
			c.input, cmd = c.input.Update(msg)
			value := c.input.Value()
			c.list.SetFilter(value)
			c.list.ScrollToTop()
			c.list.SetSelected(0)
			return ActionCmd{cmd}
		}
	}
	return nil
}

// Cursor 返回相对于对话框的光标位置。
func (c *Commands) Cursor() *tea.Cursor {
	return InputCursor(c.com.Styles, c.input.Cursor())
}

// commandsRadioView 生成命令类型选择器单选按钮。
func commandsRadioView(sty *styles.Styles, selected CommandType, hasUserCmds bool, hasMCPPrompts bool) string {
	if !hasUserCmds && !hasMCPPrompts {
		return ""
	}

	selectedFn := func(t CommandType) string {
		if t == selected {
			return sty.RadioOn.Padding(0, 1).Render() + sty.HalfMuted.Render(t.String())
		}
		return sty.RadioOff.Padding(0, 1).Render() + sty.HalfMuted.Render(t.String())
	}

	parts := []string{
		selectedFn(SystemCommands),
	}

	if hasUserCmds {
		parts = append(parts, selectedFn(UserCommands))
	}
	if hasMCPPrompts {
		parts = append(parts, selectedFn(MCPPrompts))
	}

	return strings.Join(parts, " ")
}

// Draw 实现 [Dialog] 接口。
func (c *Commands) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	t := c.com.Styles
	width := max(0, min(defaultCommandsDialogMaxWidth, area.Dx()-t.Dialog.View.GetHorizontalBorderSize()))
	height := max(0, min(defaultCommandsDialogMaxHeight, area.Dy()-t.Dialog.View.GetVerticalBorderSize()))
	if area.Dx() != c.windowWidth && c.selected == SystemCommands {
		c.windowWidth = area.Dx()
		// 由于列表中的某些项目依赖于宽度（例如切换侧边栏命令），
		// 我们需要在宽度变化时重置命令项目
		c.setCommandItems(c.selected)
	}

	innerWidth := width - c.com.Styles.Dialog.View.GetHorizontalFrameSize()
	heightOffset := t.Dialog.Title.GetVerticalFrameSize() + titleContentHeight +
		t.Dialog.InputPrompt.GetVerticalFrameSize() + inputContentHeight +
		t.Dialog.HelpView.GetVerticalFrameSize() +
		t.Dialog.View.GetVerticalFrameSize()

	c.input.SetWidth(max(0, innerWidth-t.Dialog.InputPrompt.GetHorizontalFrameSize()-1)) // (1) cursor padding

	c.list.SetSize(innerWidth, height-heightOffset)
	c.help.SetWidth(innerWidth)

	rc := NewRenderContext(t, width)
	rc.Title = "命令"
	rc.TitleInfo = commandsRadioView(t, c.selected, len(c.customCommands) > 0, len(c.mcpPrompts) > 0)
	inputView := t.Dialog.InputPrompt.Render(c.input.View())
	rc.AddPart(inputView)
	listView := t.Dialog.List.Height(c.list.Height()).Render(c.list.Render())
	rc.AddPart(listView)
	rc.Help = c.help.View(c)

	if c.loading {
		rc.Help = c.spinner.View() + " 正在生成提示..."
	}

	view := rc.Render()

	cur := c.Cursor()
	DrawCenterCursor(scr, area, view, cur)
	return cur
}

// ShortHelp 实现 [help.KeyMap] 接口。
func (c *Commands) ShortHelp() []key.Binding {
	return []key.Binding{
		c.keyMap.Tab,
		c.keyMap.UpDown,
		c.keyMap.Select,
		c.keyMap.Close,
	}
}

// FullHelp 实现 [help.KeyMap] 接口。
func (c *Commands) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{c.keyMap.Select, c.keyMap.Next, c.keyMap.Previous, c.keyMap.Tab},
		{c.keyMap.Close},
	}
}

// nextCommandType 返回循环中的下一个命令类型。
func (c *Commands) nextCommandType() CommandType {
	switch c.selected {
	case SystemCommands:
		if len(c.customCommands) > 0 {
			return UserCommands
		}
		if len(c.mcpPrompts) > 0 {
			return MCPPrompts
		}
		fallthrough
	case UserCommands:
		if len(c.mcpPrompts) > 0 {
			return MCPPrompts
		}
		fallthrough
	case MCPPrompts:
		return SystemCommands
	default:
		return SystemCommands
	}
}

// previousCommandType 返回循环中的上一个命令类型。
func (c *Commands) previousCommandType() CommandType {
	switch c.selected {
	case SystemCommands:
		if len(c.mcpPrompts) > 0 {
			return MCPPrompts
		}
		if len(c.customCommands) > 0 {
			return UserCommands
		}
		return SystemCommands
	case UserCommands:
		return SystemCommands
	case MCPPrompts:
		if len(c.customCommands) > 0 {
			return UserCommands
		}
		return SystemCommands
	default:
		return SystemCommands
	}
}

// setCommandItems 根据指定的命令类型设置命令项目。
func (c *Commands) setCommandItems(commandType CommandType) {
	c.selected = commandType

	commandItems := []list.FilterableItem{}
	switch c.selected {
	case SystemCommands:
		for _, cmd := range c.defaultCommands() {
			commandItems = append(commandItems, cmd)
		}
	case UserCommands:
		for _, cmd := range c.customCommands {
			action := ActionRunCustomCommand{
				Content:   cmd.Content,
				Arguments: cmd.Arguments,
			}
			commandItems = append(commandItems, NewCommandItem(c.com.Styles, "custom_"+cmd.ID, cmd.Name, "", action))
		}
	case MCPPrompts:
		for _, cmd := range c.mcpPrompts {
			action := ActionRunMCPPrompt{
				Title:       cmd.Title,
				Description: cmd.Description,
				PromptID:    cmd.PromptID,
				ClientID:    cmd.ClientID,
				Arguments:   cmd.Arguments,
			}
			commandItems = append(commandItems, NewCommandItem(c.com.Styles, "mcp_"+cmd.ID, cmd.PromptID, "", action))
		}
	}

	c.list.SetItems(commandItems...)
	c.list.SetFilter("")
	c.list.ScrollToTop()
	c.list.SetSelected(0)
	c.input.SetValue("")
}

// defaultCommands 返回默认系统命令列表。
func (c *Commands) defaultCommands() []*CommandItem {
	commands := []*CommandItem{
		NewCommandItem(c.com.Styles, "new_session", "新建会话", "ctrl+n", ActionNewSession{}),
		NewCommandItem(c.com.Styles, "switch_session", "会话", "ctrl+s", ActionOpenDialog{SessionsID}),
		NewCommandItem(c.com.Styles, "switch_model", "切换模型", "ctrl+l", ActionOpenDialog{ModelsID}),
	}

	// 仅在有活动会话时显示摘要命令
	if c.sessionID != "" {
		commands = append(commands, NewCommandItem(c.com.Styles, "summarize", "摘要会话", "", ActionSummarize{SessionID: c.sessionID}))
	}

	// 为支持推理的模型添加推理切换
	cfg := c.com.Config()
	if agentCfg, ok := cfg.Agents[config.AgentCoder]; ok {
		providerCfg := cfg.GetProviderForModel(agentCfg.Model)
		model := cfg.GetModelByType(agentCfg.Model)
		if providerCfg != nil && model != nil && model.CanReason {
			selectedModel := cfg.Models[agentCfg.Model]

			// Anthropic 模型：思考切换
			if model.CanReason && len(model.ReasoningLevels) == 0 {
				status := "启用"
				if selectedModel.Think {
					status = "禁用"
				}
				commands = append(commands, NewCommandItem(c.com.Styles, "toggle_thinking", status+" 思考模式", "", ActionToggleThinking{}))
			}

			// OpenAI 模型：推理强度对话框
			if len(model.ReasoningLevels) > 0 {
				commands = append(commands, NewCommandItem(c.com.Styles, "select_reasoning_effort", "选择推理强度", "", ActionOpenDialog{
					DialogID: ReasoningID,
				}))
			}
		}
	}
	// 仅在窗口宽度大于紧凑断点（120）时显示切换紧凑模式命令
	if c.windowWidth >= sidebarCompactModeBreakpoint && c.sessionID != "" {
		commands = append(commands, NewCommandItem(c.com.Styles, "toggle_sidebar", "切换侧边栏", "", ActionToggleCompactMode{}))
	}
	if c.sessionID != "" {
		cfg := c.com.Config()
		agentCfg := cfg.Agents[config.AgentCoder]
		model := cfg.GetModelByType(agentCfg.Model)
		if model != nil && model.SupportsImages {
			commands = append(commands, NewCommandItem(c.com.Styles, "file_picker", "打开文件选择器", "ctrl+f", ActionOpenDialog{
				// TODO: 传入文件选择器对话框 ID
			}))
		}
	}

	// 如果 $EDITOR 可用，则添加外部编辑器命令
	// TODO: 使用 [tea.EnvMsg] 获取环境变量而不是 os.Getenv
	if os.Getenv("EDITOR") != "" {
		commands = append(commands, NewCommandItem(c.com.Styles, "open_external_editor", "打开外部编辑器", "ctrl+o", ActionExternalEditor{}))
	}

	return append(commands,
		NewCommandItem(c.com.Styles, "toggle_yolo", "切换 Yolo 模式", "", ActionToggleYoloMode{}),
		NewCommandItem(c.com.Styles, "toggle_help", "切换帮助", "ctrl+g", ActionToggleHelp{}),
		NewCommandItem(c.com.Styles, "init", "初始化项目", "", ActionInitializeProject{}),
		NewCommandItem(c.com.Styles, "quit", "退出", "ctrl+c", tea.QuitMsg{}),
	)
}

// SetCustomCommands 设置自定义命令，如果当前显示用户命令则刷新视图。
func (c *Commands) SetCustomCommands(customCommands []commands.CustomCommand) {
	c.customCommands = customCommands
	if c.selected == UserCommands {
		c.setCommandItems(c.selected)
	}
}

// SetMCPPrompts 设置 MCP 提示，如果当前显示 MCP 提示则刷新视图。
func (c *Commands) SetMCPPrompts(mcpPrompts []commands.MCPPrompt) {
	c.mcpPrompts = mcpPrompts
	if c.selected == MCPPrompts {
		c.setCommandItems(c.selected)
	}
}

// StartLoading 实现 [LoadingDialog] 接口。
func (a *Commands) StartLoading() tea.Cmd {
	if a.loading {
		return nil
	}
	a.loading = true
	return a.spinner.Tick
}

// StopLoading 实现 [LoadingDialog] 接口。
func (a *Commands) StopLoading() {
	a.loading = false
}
