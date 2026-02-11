package model

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/catwalk/pkg/catwalk"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/agent/tools/mcp"
	"github.com/charmbracelet/crush/internal/app"
	"github.com/charmbracelet/crush/internal/commands"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/fsext"
	"github.com/charmbracelet/crush/internal/history"
	"github.com/charmbracelet/crush/internal/home"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/permission"
	"github.com/charmbracelet/crush/internal/pubsub"
	"github.com/charmbracelet/crush/internal/session"
	"github.com/charmbracelet/crush/internal/ui/anim"
	"github.com/charmbracelet/crush/internal/ui/attachments"
	"github.com/charmbracelet/crush/internal/ui/chat"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/completions"
	"github.com/charmbracelet/crush/internal/ui/dialog"
	fimage "github.com/charmbracelet/crush/internal/ui/image"
	"github.com/charmbracelet/crush/internal/ui/logo"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/charmbracelet/crush/internal/ui/util"
	"github.com/charmbracelet/crush/internal/version"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/ultraviolet/layout"
	"github.com/charmbracelet/ultraviolet/screen"
	"github.com/charmbracelet/x/editor"
)

// 紧凑模式断点
const (
	compactModeWidthBreakpoint  = 120 // 紧凑模式宽度断点
	compactModeHeightBreakpoint = 30  // 紧凑模式高度断点
)

// 如果粘贴的文本超过2个换行符，则将其视为文件附件
const pasteLinesThreshold = 10

// 会话详情面板最大高度
const sessionDetailsMaxHeight = 20

// uiFocusState 表示UI当前的焦点状态
type uiFocusState uint8

// uiFocusState 的可能值
const (
	uiFocusNone   uiFocusState = iota // 无焦点
	uiFocusEditor                     // 编辑器焦点
	uiFocusMain                       // 主区域焦点
)

type uiState uint8

// uiState 的可能值
const (
	uiOnboarding uiState = iota // 引导状态
	uiInitialize                // 初始化状态
	uiLanding                   // 登陆页面状态
	uiChat                      // 聊天状态
)

type openEditorMsg struct {
	Text string
}

type (
	// cancelTimerExpiredMsg 在取消计时器过期时发送
	cancelTimerExpiredMsg struct{}
	// userCommandsLoadedMsg 在用户命令加载完成时发送
	userCommandsLoadedMsg struct {
		Commands []commands.CustomCommand
	}
	// mcpPromptsLoadedMsg 在MCP提示加载完成时发送
	mcpPromptsLoadedMsg struct {
		Prompts []commands.MCPPrompt
	}
	// mcpStateChangedMsg 在MCP客户端状态发生变化时发送
	mcpStateChangedMsg struct {
		states map[string]mcp.ClientInfo
	}
	// sendMessageMsg 用于发送消息
	// 目前仅用于MCP提示
	sendMessageMsg struct {
		Content     string
		Attachments []message.Attachment
	}

	// closeDialogMsg 用于关闭当前对话框
	closeDialogMsg struct{}

	// copyChatHighlightMsg 用于将当前聊天高亮内容复制到剪贴板
	copyChatHighlightMsg struct{}

	// sessionFilesUpdatesMsg 在当前会话的文件更新时发送
	sessionFilesUpdatesMsg struct {
		sessionFiles []SessionFile
	}
)

// UI 表示主用户界面模型
type UI struct {
	com          *common.Common
	session      *session.Session
	sessionFiles []SessionFile

	// 在没有会话ID时跟踪已读取的文件
	sessionFileReads []string

	lastUserMessageTime int64

	// 终端的宽度和高度（以单元格为单位）
	width  int
	height int
	layout uiLayout

	isTransparent bool

	focus uiFocusState
	state uiState

	keyMap KeyMap
	keyenh tea.KeyboardEnhancementsMsg

	dialog *dialog.Overlay
	status *Status

	// isCanceling 跟踪用户是否已按一次ESC键取消
	isCanceling bool

	header *header

	// sendProgressBar 指示TUI向终端发送进度条更新
	sendProgressBar    bool
	progressBarEnabled bool

	// caps 保存我们查询的不同终端能力
	caps common.Capabilities

	// 编辑器组件
	textarea textarea.Model

	// 附件列表
	attachments *attachments.Attachments

	readyPlaceholder   string
	workingPlaceholder string

	// 自动完成状态
	completions              *completions.Completions
	completionsOpen          bool
	completionsStartIndex    int
	completionsQuery         string
	completionsPositionStart image.Point // 用户输入'@'时的x,y坐标

	// 聊天组件
	chat *Chat

	// 引导状态
	onboarding struct {
		yesInitializeSelected bool
	}

	// LSP (Language Server Protocol - 语言服务器协议)
	lspStates map[string]app.LSPClientInfo

	// MCP (Model Context Protocol - 模型上下文协议)
	mcpStates map[string]mcp.ClientInfo

	// sidebarLogo 保存侧边栏logo的缓存版本
	sidebarLogo string

	// 自定义命令和MCP命令
	customCommands []commands.CustomCommand
	mcpPrompts     []commands.MCPPrompt

	// forceCompactMode 跟踪紧凑模式是否由用户切换强制启用
	forceCompactMode bool

	// isCompact 跟踪当前是否处于紧凑布局模式
	//（通过用户切换或基于窗口大小自动切换）
	isCompact bool

	// detailsOpen 跟踪详情面板是否打开（在紧凑模式下）
	detailsOpen bool

	// 药丸状态
	pillsExpanded      bool
	focusedPillSection pillSection
	promptQueue        int
	pillsView          string

	// 待办事项旋转器
	todoSpinner    spinner.Model
	todoIsSpinning bool

	// 鼠标高亮相关状态
	lastClickTime time.Time

	// 提示历史记录，用于通过上/下键导航到之前的消息
	promptHistory struct {
		messages []string
		index    int
		draft    string
	}
}

// New 创建一个新的 [UI] 模型实例
func New(com *common.Common) *UI {
	// 编辑器组件
	ta := textarea.New()
	ta.SetStyles(com.Styles.TextArea)
	ta.ShowLineNumbers = false
	ta.CharLimit = -1
	ta.SetVirtualCursor(false)
	ta.Focus()

	ch := NewChat(com)

	keyMap := DefaultKeyMap()

	// 自动完成组件
	comp := completions.New(
		com.Styles.Completions.Normal,
		com.Styles.Completions.Focused,
		com.Styles.Completions.Match,
	)

	todoSpinner := spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(com.Styles.Pills.TodoSpinner),
	)

	// 附件组件
	attachments := attachments.New(
		attachments.NewRenderer(
			com.Styles.Attachments.Normal,
			com.Styles.Attachments.Deleting,
			com.Styles.Attachments.Image,
			com.Styles.Attachments.Text,
		),
		attachments.Keymap{
			DeleteMode: keyMap.Editor.AttachmentDeleteMode,
			DeleteAll:  keyMap.Editor.DeleteAllAttachments,
			Escape:     keyMap.Editor.Escape,
		},
	)

	header := newHeader(com)

	ui := &UI{
		com:         com,
		dialog:      dialog.NewOverlay(),
		keyMap:      keyMap,
		textarea:    ta,
		chat:        ch,
		header:      header,
		completions: comp,
		attachments: attachments,
		todoSpinner: todoSpinner,
		lspStates:   make(map[string]app.LSPClientInfo),
		mcpStates:   make(map[string]mcp.ClientInfo),
	}

	status := NewStatus(com, ui)

	ui.setEditorPrompt(false)
	ui.randomizePlaceholders()
	ui.textarea.Placeholder = ui.readyPlaceholder
	ui.status = status

	// 从配置初始化紧凑模式
	ui.forceCompactMode = com.Config().Options.TUI.CompactMode

	// 设置引导状态默认值
	ui.onboarding.yesInitializeSelected = true

	desiredState := uiLanding
	desiredFocus := uiFocusEditor
	if !com.Config().IsConfigured() {
		desiredState = uiOnboarding
	} else if n, _ := config.ProjectNeedsInitialization(com.Config()); n {
		desiredState = uiInitialize
	}

	// 设置初始状态
	ui.setState(desiredState, desiredFocus)

	opts := com.Config().Options

	// 禁用不确定进度条
	ui.progressBarEnabled = opts.Progress == nil || *opts.Progress
	// 启用透明模式
	ui.isTransparent = opts.TUI.Transparent != nil && *opts.TUI.Transparent

	return ui
}

// Init 初始化UI模型
func (m *UI) Init() tea.Cmd {
	var cmds []tea.Cmd
	if m.state == uiOnboarding {
		if cmd := m.openModelsDialog(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	// 异步加载用户命令
	cmds = append(cmds, m.loadCustomCommands())
	// 异步加载提示历史记录
	cmds = append(cmds, m.loadPromptHistory())
	return tea.Batch(cmds...)
}

// setState 更改UI状态和焦点
func (m *UI) setState(state uiState, focus uiFocusState) {
	if state == uiLanding {
		// 进入登陆页面时始终关闭紧凑模式
		m.isCompact = false
	}
	m.state = state
	m.focus = focus
	// 更改状态可能会更改布局，因此更新它
	m.updateLayoutAndSize()
}

// loadCustomCommands 异步加载自定义命令
func (m *UI) loadCustomCommands() tea.Cmd {
	return func() tea.Msg {
		customCommands, err := commands.LoadCustomCommands(m.com.Config())
		if err != nil {
			slog.Error("加载自定义命令失败", "error", err)
		}
		return userCommandsLoadedMsg{Commands: customCommands}
	}
}

// loadMCPrompts 异步加载MCP提示
func (m *UI) loadMCPrompts() tea.Msg {
	prompts, err := commands.LoadMCPPrompts()
	if err != nil {
		slog.Error("加载MCP提示失败", "error", err)
	}
	if prompts == nil {
		// 即使没有提示或出错，也标记为已加载
		prompts = []commands.MCPPrompt{}
	}
	return mcpPromptsLoadedMsg{Prompts: prompts}
}

// Update 处理UI模型的更新
func (m *UI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	if m.hasSession() && m.isAgentBusy() {
		queueSize := m.com.App.AgentCoordinator.QueuedPrompts(m.session.ID)
		if queueSize != m.promptQueue {
			m.promptQueue = queueSize
			m.updateLayoutAndSize()
		}
	}
	// 更新终端能力
	m.caps.Update(msg)
	switch msg := msg.(type) {
	case tea.EnvMsg:
		// 这是Windows Terminal吗？
		if !m.sendProgressBar {
			m.sendProgressBar = slices.Contains(msg, "WT_SESSION")
		}
		cmds = append(cmds, common.QueryCmd(uv.Environ(msg)))
	case loadSessionMsg:
		if m.forceCompactMode {
			m.isCompact = true
		}
		m.setState(uiChat, m.focus)
		m.session = msg.session
		m.sessionFiles = msg.files
		cmds = append(cmds, m.startLSPs(msg.lspFilePaths()))
		msgs, err := m.com.App.Messages.List(context.Background(), m.session.ID)
		if err != nil {
			cmds = append(cmds, util.ReportError(err))
			break
		}
		if cmd := m.setSessionMessages(msgs); cmd != nil {
			cmds = append(cmds, cmd)
		}
		if hasInProgressTodo(m.session.Todos) {
			// 仅当有进行中的待办事项时才启动旋转器
			if m.isAgentBusy() {
				m.todoIsSpinning = true
				cmds = append(cmds, m.todoSpinner.Tick)
			}
			m.updateLayoutAndSize()
		}
		// 为新会话重新加载提示历史记录
		m.historyReset()
		cmds = append(cmds, m.loadPromptHistory())
		m.updateLayoutAndSize()

	case sessionFilesUpdatesMsg:
		m.sessionFiles = msg.sessionFiles
		var paths []string
		for _, f := range msg.sessionFiles {
			paths = append(paths, f.LatestVersion.Path)
		}
		cmds = append(cmds, m.startLSPs(paths))

	case sendMessageMsg:
		cmds = append(cmds, m.sendMessage(msg.Content, msg.Attachments...))

	case userCommandsLoadedMsg:
		m.customCommands = msg.Commands
		dia := m.dialog.Dialog(dialog.CommandsID)
		if dia == nil {
			break
		}

		commands, ok := dia.(*dialog.Commands)
		if ok {
			commands.SetCustomCommands(m.customCommands)
		}

	case mcpStateChangedMsg:
		m.mcpStates = msg.states
	case mcpPromptsLoadedMsg:
		m.mcpPrompts = msg.Prompts
		dia := m.dialog.Dialog(dialog.CommandsID)
		if dia == nil {
			break
		}

		commands, ok := dia.(*dialog.Commands)
		if ok {
			commands.SetMCPPrompts(m.mcpPrompts)
		}

	case promptHistoryLoadedMsg:
		m.promptHistory.messages = msg.messages
		m.promptHistory.index = -1
		m.promptHistory.draft = ""

	case closeDialogMsg:
		m.dialog.CloseFrontDialog()

	case pubsub.Event[session.Session]:
		if msg.Type == pubsub.DeletedEvent {
			if m.session != nil && m.session.ID == msg.Payload.ID {
				if cmd := m.newSession(); cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
			break
		}
		if m.session != nil && msg.Payload.ID == m.session.ID {
			prevHasInProgress := hasInProgressTodo(m.session.Todos)
			m.session = &msg.Payload
			if !prevHasInProgress && hasInProgressTodo(m.session.Todos) {
				m.todoIsSpinning = true
				cmds = append(cmds, m.todoSpinner.Tick)
				m.updateLayoutAndSize()
			}
		}
	case pubsub.Event[message.Message]:
		// 检查这是否是智能体工具的子会话消息
		if m.session == nil {
			break
		}
		if msg.Payload.SessionID != m.session.ID {
			// 这可能是来自智能体工具的子会话消息
			if cmd := m.handleChildSessionMessage(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
			break
		}
		switch msg.Type {
		case pubsub.CreatedEvent:
			cmds = append(cmds, m.appendSessionMessage(msg.Payload))
		case pubsub.UpdatedEvent:
			cmds = append(cmds, m.updateSessionMessage(msg.Payload))
		case pubsub.DeletedEvent:
			m.chat.RemoveMessage(msg.Payload.ID)
		}
		// 如果有新消息则启动旋转器
		if hasInProgressTodo(m.session.Todos) && m.isAgentBusy() && !m.todoIsSpinning {
			m.todoIsSpinning = true
			cmds = append(cmds, m.todoSpinner.Tick)
		}
		// 如果智能体不再忙碌则停止旋转器
		if m.todoIsSpinning && !m.isAgentBusy() {
			m.todoIsSpinning = false
		}
		// 这里有很多事情可能会改变药丸，所以我们想要重新渲染
		m.renderPills()
	case pubsub.Event[history.File]:
		cmds = append(cmds, m.handleFileEvent(msg.Payload))
	case pubsub.Event[app.LSPEvent]:
		m.lspStates = app.GetLSPStates()
	case pubsub.Event[mcp.Event]:
		switch msg.Payload.Type {
		case mcp.EventStateChanged:
			return m, tea.Batch(
				m.handleStateChanged(),
				m.loadMCPrompts,
			)
		case mcp.EventPromptsListChanged:
			return m, handleMCPPromptsEvent(msg.Payload.Name)
		case mcp.EventToolsListChanged:
			return m, handleMCPToolsEvent(m.com.Config(), msg.Payload.Name)
		case mcp.EventResourcesListChanged:
			return m, handleMCPResourcesEvent(msg.Payload.Name)
		}
	case pubsub.Event[permission.PermissionRequest]:
		if cmd := m.openPermissionsDialog(msg.Payload); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case pubsub.Event[permission.PermissionNotification]:
		m.handlePermissionNotification(msg.Payload)
	case cancelTimerExpiredMsg:
		m.isCanceling = false
	case tea.TerminalVersionMsg:
		termVersion := strings.ToLower(msg.Name)
		// 仅对以下终端启用进度条
		if !m.sendProgressBar {
			m.sendProgressBar = strings.Contains(termVersion, "ghostty")
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.updateLayoutAndSize()
	case tea.KeyboardEnhancementsMsg:
		m.keyenh = msg
		if msg.SupportsKeyDisambiguation() {
			m.keyMap.Models.SetHelp("ctrl+m", "模型")
			m.keyMap.Editor.Newline.SetHelp("shift+enter", "换行")
		}
	case copyChatHighlightMsg:
		cmds = append(cmds, m.copyChatHighlight())
	case DelayedClickMsg:
		// 处理延迟单击操作（例如，展开）
		m.chat.HandleDelayedClick(msg)
	case tea.MouseClickMsg:
		// 如果打开了对话框，首先将鼠标事件传递给对话框
		if m.dialog.HasDialogs() {
			m.dialog.Update(msg)
			return m, tea.Batch(cmds...)
		}

		if cmd := m.handleClickFocus(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}

		switch m.state {
		case uiChat:
			x, y := msg.X, msg.Y
			// 调整聊天区域位置
			x -= m.layout.main.Min.X
			y -= m.layout.main.Min.Y
			if !image.Pt(msg.X, msg.Y).In(m.layout.sidebar) {
				if handled, cmd := m.chat.HandleMouseDown(x, y); handled {
					m.lastClickTime = time.Now()
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			}
		}

	case tea.MouseMotionMsg:
		// 如果打开了对话框，首先将鼠标事件传递给对话框
		if m.dialog.HasDialogs() {
			m.dialog.Update(msg)
			return m, tea.Batch(cmds...)
		}

		switch m.state {
		case uiChat:
			if msg.Y <= 0 {
				if cmd := m.chat.ScrollByAndAnimate(-1); cmd != nil {
					cmds = append(cmds, cmd)
				}
				if !m.chat.SelectedItemInView() {
					m.chat.SelectPrev()
					if cmd := m.chat.ScrollToSelectedAndAnimate(); cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			} else if msg.Y >= m.chat.Height()-1 {
				if cmd := m.chat.ScrollByAndAnimate(1); cmd != nil {
					cmds = append(cmds, cmd)
				}
				if !m.chat.SelectedItemInView() {
					m.chat.SelectNext()
					if cmd := m.chat.ScrollToSelectedAndAnimate(); cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			}

			x, y := msg.X, msg.Y
			// 调整聊天区域位置
			x -= m.layout.main.Min.X
			y -= m.layout.main.Min.Y
			m.chat.HandleMouseDrag(x, y)
		}

	case tea.MouseReleaseMsg:
		// 如果打开了对话框，首先将鼠标事件传递给对话框
		if m.dialog.HasDialogs() {
			m.dialog.Update(msg)
			return m, tea.Batch(cmds...)
		}

		switch m.state {
		case uiChat:
			x, y := msg.X, msg.Y
			// 调整聊天区域位置
			x -= m.layout.main.Min.X
			y -= m.layout.main.Min.Y
			if m.chat.HandleMouseUp(x, y) && m.chat.HasHighlight() {
				cmds = append(cmds, tea.Tick(doubleClickThreshold, func(t time.Time) tea.Msg {
					if time.Since(m.lastClickTime) >= doubleClickThreshold {
						return copyChatHighlightMsg{}
					}
					return nil
				}))
			}
		}
	case tea.MouseWheelMsg:
		// 如果打开了对话框，首先将鼠标事件传递给对话框
		if m.dialog.HasDialogs() {
			m.dialog.Update(msg)
			return m, tea.Batch(cmds...)
		}

		// 否则为聊天处理鼠标滚轮
		switch m.state {
		case uiChat:
			switch msg.Button {
			case tea.MouseWheelUp:
				if cmd := m.chat.ScrollByAndAnimate(-5); cmd != nil {
					cmds = append(cmds, cmd)
				}
				if !m.chat.SelectedItemInView() {
					m.chat.SelectPrev()
					if cmd := m.chat.ScrollToSelectedAndAnimate(); cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			case tea.MouseWheelDown:
				if cmd := m.chat.ScrollByAndAnimate(5); cmd != nil {
					cmds = append(cmds, cmd)
				}
				if !m.chat.SelectedItemInView() {
					m.chat.SelectNext()
					if cmd := m.chat.ScrollToSelectedAndAnimate(); cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			}
		}
	case anim.StepMsg:
		if m.state == uiChat {
			if cmd := m.chat.Animate(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case spinner.TickMsg:
		if m.dialog.HasDialogs() {
			// 路由到对话框
			if cmd := m.handleDialogMsg(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		if m.state == uiChat && m.hasSession() && hasInProgressTodo(m.session.Todos) && m.todoIsSpinning {
			var cmd tea.Cmd
			m.todoSpinner, cmd = m.todoSpinner.Update(msg)
			if cmd != nil {
				m.renderPills()
				cmds = append(cmds, cmd)
			}
		}

	case tea.KeyPressMsg:
		if cmd := m.handleKeyPressMsg(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case tea.PasteMsg:
		if cmd := m.handlePasteMsg(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case openEditorMsg:
		var cmd tea.Cmd
		m.textarea.SetValue(msg.Text)
		m.textarea.MoveToEnd()
		m.textarea, cmd = m.textarea.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case util.InfoMsg:
		m.status.SetInfoMsg(msg)
		ttl := msg.TTL
		if ttl <= 0 {
			ttl = DefaultStatusTTL
		}
		cmds = append(cmds, clearInfoMsgCmd(ttl))
	case util.ClearStatusMsg:
		m.status.ClearInfoMsg()
	case completions.CompletionItemsLoadedMsg:
		if m.completionsOpen {
			m.completions.SetItems(msg.Files, msg.Resources)
		}
	case uv.KittyGraphicsEvent:
		if !bytes.HasPrefix(msg.Payload, []byte("OK")) {
			slog.Warn("意外的Kitty图形响应",
				"response", string(msg.Payload),
				"options", msg.Options)
		}
	default:
		if m.dialog.HasDialogs() {
			if cmd := m.handleDialogMsg(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	// 此逻辑在任何消息类型上触发，但应该触发吗？
	switch m.focus {
	case uiFocusMain:
	case uiFocusEditor:
		// 文本区域占位符逻辑
		if m.isAgentBusy() {
			m.textarea.Placeholder = m.workingPlaceholder
		} else {
			m.textarea.Placeholder = m.readyPlaceholder
		}
		if m.com.App.Permissions.SkipRequests() {
			m.textarea.Placeholder = "Yolo模式！"
		}
	}

	// 此时这只能处理 [message.Attachment] 消息，我们应该返回所有命令
	_ = m.attachments.Update(msg)
	return m, tea.Batch(cmds...)
}

// setSessionMessages 为当前会话的聊天设置消息
func (m *UI) setSessionMessages(msgs []message.Message) tea.Cmd {
	var cmds []tea.Cmd
	// 构建工具结果映射以链接工具调用及其结果
	msgPtrs := make([]*message.Message, len(msgs))
	for i := range msgs {
		msgPtrs[i] = &msgs[i]
	}
	toolResultMap := chat.BuildToolResultMap(msgPtrs)
	if len(msgPtrs) > 0 {
		m.lastUserMessageTime = msgPtrs[0].CreatedAt
	}

	// 添加消息到聊天，并链接工具结果
	items := make([]chat.MessageItem, 0, len(msgs)*2)
	for _, msg := range msgPtrs {
		switch msg.Role {
		case message.User:
			m.lastUserMessageTime = msg.CreatedAt
			items = append(items, chat.ExtractMessageItems(m.com.Styles, msg, toolResultMap)...)
		case message.Assistant:
			items = append(items, chat.ExtractMessageItems(m.com.Styles, msg, toolResultMap)...)
			if msg.FinishPart() != nil && msg.FinishPart().Reason == message.FinishReasonEndTurn {
				infoItem := chat.NewAssistantInfoItem(m.com.Styles, msg, m.com.Config(), time.Unix(m.lastUserMessageTime, 0))
				items = append(items, infoItem)
			}
		default:
			items = append(items, chat.ExtractMessageItems(m.com.Styles, msg, toolResultMap)...)
		}
	}

	// 为智能体/agentic_fetch工具加载嵌套工具调用
	m.loadNestedToolCalls(items)

	// 如果用户在智能体工作时切换会话，我们要确保显示动画
	for _, item := range items {
		if animatable, ok := item.(chat.Animatable); ok {
			if cmd := animatable.StartAnimation(); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	m.chat.SetMessages(items...)
	if cmd := m.chat.ScrollToBottomAndAnimate(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	m.chat.SelectLast()
	return tea.Batch(cmds...)
}

// loadNestedToolCalls 递归加载智能体/agentic_fetch工具的嵌套工具调用
func (m *UI) loadNestedToolCalls(items []chat.MessageItem) {
	for _, item := range items {
		nestedContainer, ok := item.(chat.NestedToolContainer)
		if !ok {
			continue
		}
		toolItem, ok := item.(chat.ToolMessageItem)
		if !ok {
			continue
		}

		tc := toolItem.ToolCall()
		messageID := toolItem.MessageID()

		// 获取智能体工具会话ID
		agentSessionID := m.com.App.Sessions.CreateAgentToolSessionID(messageID, tc.ID)

		// 获取嵌套消息
		nestedMsgs, err := m.com.App.Messages.List(context.Background(), agentSessionID)
		if err != nil || len(nestedMsgs) == 0 {
			continue
		}

		// 为嵌套消息构建工具结果映射
		nestedMsgPtrs := make([]*message.Message, len(nestedMsgs))
		for i := range nestedMsgs {
			nestedMsgPtrs[i] = &nestedMsgs[i]
		}
		nestedToolResultMap := chat.BuildToolResultMap(nestedMsgPtrs)

		// 提取嵌套工具项
		var nestedTools []chat.ToolMessageItem
		for _, nestedMsg := range nestedMsgPtrs {
			nestedItems := chat.ExtractMessageItems(m.com.Styles, nestedMsg, nestedToolResultMap)
			for _, nestedItem := range nestedItems {
				if nestedToolItem, ok := nestedItem.(chat.ToolMessageItem); ok {
					// 将嵌套工具标记为简单（紧凑）渲染
					if simplifiable, ok := nestedToolItem.(chat.Compactable); ok {
						simplifiable.SetCompact(true)
					}
					nestedTools = append(nestedTools, nestedToolItem)
				}
			}
		}

		// 递归加载任何智能体工具内的嵌套工具调用
		nestedMessageItems := make([]chat.MessageItem, len(nestedTools))
		for i, nt := range nestedTools {
			nestedMessageItems[i] = nt
		}
		m.loadNestedToolCalls(nestedMessageItems)

		// 在父项上设置嵌套工具
		nestedContainer.SetNestedTools(nestedTools)
	}
}

// appendSessionMessage 将新消息追加到当前会话的聊天中
// 如果消息是工具结果，它将更新相应的工具调用消息
func (m *UI) appendSessionMessage(msg message.Message) tea.Cmd {
	var cmds []tea.Cmd
	atBottom := m.chat.list.AtBottom()

	existing := m.chat.MessageItem(msg.ID)
	if existing != nil {
		// 消息已存在，跳过
		return nil
	}

	switch msg.Role {
	case message.User:
		m.lastUserMessageTime = msg.CreatedAt
		items := chat.ExtractMessageItems(m.com.Styles, &msg, nil)
		for _, item := range items {
			if animatable, ok := item.(chat.Animatable); ok {
				if cmd := animatable.StartAnimation(); cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}
		m.chat.AppendMessages(items...)
		if cmd := m.chat.ScrollToBottomAndAnimate(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case message.Assistant:
		items := chat.ExtractMessageItems(m.com.Styles, &msg, nil)
		for _, item := range items {
			if animatable, ok := item.(chat.Animatable); ok {
				if cmd := animatable.StartAnimation(); cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}
		m.chat.AppendMessages(items...)
		if atBottom {
			if cmd := m.chat.ScrollToBottomAndAnimate(); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		if msg.FinishPart() != nil && msg.FinishPart().Reason == message.FinishReasonEndTurn {
			infoItem := chat.NewAssistantInfoItem(m.com.Styles, &msg, m.com.Config(), time.Unix(m.lastUserMessageTime, 0))
			m.chat.AppendMessages(infoItem)
			if atBottom {
				if cmd := m.chat.ScrollToBottomAndAnimate(); cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}
	case message.Tool:
		for _, tr := range msg.ToolResults() {
			toolItem := m.chat.MessageItem(tr.ToolCallID)
			if toolItem == nil {
				// 我们应该有一个项！
				continue
			}
			if toolMsgItem, ok := toolItem.(chat.ToolMessageItem); ok {
				toolMsgItem.SetResult(&tr)
				if atBottom {
					if cmd := m.chat.ScrollToBottomAndAnimate(); cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			}
		}
	}
	return tea.Batch(cmds...)
}

func (m *UI) handleClickFocus(msg tea.MouseClickMsg) (cmd tea.Cmd) {
	switch {
	case m.state != uiChat:
		return nil
	case image.Pt(msg.X, msg.Y).In(m.layout.sidebar):
		return nil
	case m.focus != uiFocusEditor && image.Pt(msg.X, msg.Y).In(m.layout.editor):
		m.focus = uiFocusEditor
		cmd = m.textarea.Focus()
		m.chat.Blur()
	case m.focus != uiFocusMain && image.Pt(msg.X, msg.Y).In(m.layout.main):
		m.focus = uiFocusMain
		m.textarea.Blur()
		m.chat.Focus()
	}
	return cmd
}

// updateSessionMessage 更新当前会话聊天中的现有消息
// 当助手消息更新时，它可能还包括更新的工具调用
// 这就是为什么我们需要处理创建/更新每个工具调用消息
func (m *UI) updateSessionMessage(msg message.Message) tea.Cmd {
	var cmds []tea.Cmd
	existingItem := m.chat.MessageItem(msg.ID)
	atBottom := m.chat.list.AtBottom()

	if existingItem != nil {
		if assistantItem, ok := existingItem.(*chat.AssistantMessageItem); ok {
			assistantItem.SetMessage(&msg)
		}
	}

	shouldRenderAssistant := chat.ShouldRenderAssistantMessage(&msg)
	// 如果助手消息没有任何响应，只有工具调用，我们需要删除它
	if !shouldRenderAssistant && len(msg.ToolCalls()) > 0 && existingItem != nil {
		m.chat.RemoveMessage(msg.ID)
		if infoItem := m.chat.MessageItem(chat.AssistantInfoID(msg.ID)); infoItem != nil {
			m.chat.RemoveMessage(chat.AssistantInfoID(msg.ID))
		}
	}

	if shouldRenderAssistant && msg.FinishPart() != nil && msg.FinishPart().Reason == message.FinishReasonEndTurn {
		if infoItem := m.chat.MessageItem(chat.AssistantInfoID(msg.ID)); infoItem == nil {
			newInfoItem := chat.NewAssistantInfoItem(m.com.Styles, &msg, m.com.Config(), time.Unix(m.lastUserMessageTime, 0))
			m.chat.AppendMessages(newInfoItem)
		}
	}

	var items []chat.MessageItem
	for _, tc := range msg.ToolCalls() {
		existingToolItem := m.chat.MessageItem(tc.ID)
		if toolItem, ok := existingToolItem.(chat.ToolMessageItem); ok {
			existingToolCall := toolItem.ToolCall()
			// 仅在完成状态更改或输入更改时更新
			// 以避免清除缓存
			if (tc.Finished && !existingToolCall.Finished) || tc.Input != existingToolCall.Input {
				toolItem.SetToolCall(tc)
			}
		}
		if existingToolItem == nil {
			items = append(items, chat.NewToolMessageItem(m.com.Styles, msg.ID, tc, nil, false))
		}
	}

	for _, item := range items {
		if animatable, ok := item.(chat.Animatable); ok {
			if cmd := animatable.StartAnimation(); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	m.chat.AppendMessages(items...)
	if atBottom {
		if cmd := m.chat.ScrollToBottomAndAnimate(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return tea.Batch(cmds...)
}

// handleChildSessionMessage 处理来自子会话（智能体工具）的消息
func (m *UI) handleChildSessionMessage(event pubsub.Event[message.Message]) tea.Cmd {
	var cmds []tea.Cmd

	atBottom := m.chat.list.AtBottom()
	// 仅处理具有工具调用或结果的消息
	if len(event.Payload.ToolCalls()) == 0 && len(event.Payload.ToolResults()) == 0 {
		return nil
	}

	// 检查这是否是智能体工具会话并解析它
	childSessionID := event.Payload.SessionID
	_, toolCallID, ok := m.com.App.Sessions.ParseAgentToolSessionID(childSessionID)
	if !ok {
		return nil
	}

	// 查找父智能体工具项
	var agentItem chat.NestedToolContainer
	for i := 0; i < m.chat.Len(); i++ {
		item := m.chat.MessageItem(toolCallID)
		if item == nil {
			continue
		}
		if agent, ok := item.(chat.NestedToolContainer); ok {
			if toolMessageItem, ok := item.(chat.ToolMessageItem); ok {
				if toolMessageItem.ToolCall().ID == toolCallID {
					// 验证此智能体属于正确的父消息
					// 我们无法直接检查项上的parentMessageID，所以我们信任会话解析
					agentItem = agent
					break
				}
			}
		}
	}

	if agentItem == nil {
		return nil
	}

	// 获取现有的嵌套工具
	nestedTools := agentItem.NestedTools()

	// 更新或创建嵌套工具调用
	for _, tc := range event.Payload.ToolCalls() {
		found := false
		for _, existingTool := range nestedTools {
			if existingTool.ToolCall().ID == tc.ID {
				existingTool.SetToolCall(tc)
				found = true
				break
			}
		}
		if !found {
			// 创建新的嵌套工具项
			nestedItem := chat.NewToolMessageItem(m.com.Styles, event.Payload.ID, tc, nil, false)
			if simplifiable, ok := nestedItem.(chat.Compactable); ok {
				simplifiable.SetCompact(true)
			}
			if animatable, ok := nestedItem.(chat.Animatable); ok {
				if cmd := animatable.StartAnimation(); cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
			nestedTools = append(nestedTools, nestedItem)
		}
	}

	// 更新嵌套工具结果
	for _, tr := range event.Payload.ToolResults() {
		for _, nestedTool := range nestedTools {
			if nestedTool.ToolCall().ID == tr.ToolCallID {
				nestedTool.SetResult(&tr)
				break
			}
		}
	}

	// 使用新的嵌套工具更新智能体项
	agentItem.SetNestedTools(nestedTools)

	// 更新聊天，以便它更新索引映射以使动画按预期工作
	m.chat.UpdateNestedToolIDs(toolCallID)

	if atBottom {
		if cmd := m.chat.ScrollToBottomAndAnimate(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return tea.Batch(cmds...)
}

func (m *UI) handleDialogMsg(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	action := m.dialog.Update(msg)
	if action == nil {
		return tea.Batch(cmds...)
	}

	isOnboarding := m.state == uiOnboarding

	switch msg := action.(type) {
	// 通用对话框消息
	case dialog.ActionClose:
		if isOnboarding && m.dialog.ContainsDialog(dialog.ModelsID) {
			break
		}

		if m.dialog.ContainsDialog(dialog.FilePickerID) {
			defer fimage.ResetCache()
		}

		m.dialog.CloseFrontDialog()

		if isOnboarding {
			if cmd := m.openModelsDialog(); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

		if m.focus == uiFocusEditor {
			cmds = append(cmds, m.textarea.Focus())
		}
	case dialog.ActionCmd:
		if msg.Cmd != nil {
			cmds = append(cmds, msg.Cmd)
		}

	// 会话对话框消息
	case dialog.ActionSelectSession:
		m.dialog.CloseDialog(dialog.SessionsID)
		cmds = append(cmds, m.loadSession(msg.Session.ID))

	// 打开对话框消息
	case dialog.ActionOpenDialog:
		m.dialog.CloseDialog(dialog.CommandsID)
		if cmd := m.openDialog(msg.DialogID); cmd != nil {
			cmds = append(cmds, cmd)
		}

	// 命令对话框消息
	case dialog.ActionToggleYoloMode:
		yolo := !m.com.App.Permissions.SkipRequests()
		m.com.App.Permissions.SetSkipRequests(yolo)
		m.setEditorPrompt(yolo)
		m.dialog.CloseDialog(dialog.CommandsID)
	case dialog.ActionNewSession:
		if m.isAgentBusy() {
			cmds = append(cmds, util.ReportWarn("智能体忙碌，请等待后再开始新会话..."))
			break
		}
		if cmd := m.newSession(); cmd != nil {
			cmds = append(cmds, cmd)
		}
		m.dialog.CloseDialog(dialog.CommandsID)
	case dialog.ActionSummarize:
		if m.isAgentBusy() {
			cmds = append(cmds, util.ReportWarn("智能体忙碌，请等待后再总结会话..."))
			break
		}
		cmds = append(cmds, func() tea.Msg {
			err := m.com.App.AgentCoordinator.Summarize(context.Background(), msg.SessionID)
			if err != nil {
				return util.ReportError(err)()
			}
			return nil
		})
		m.dialog.CloseDialog(dialog.CommandsID)
	case dialog.ActionToggleHelp:
		m.status.ToggleHelp()
		m.dialog.CloseDialog(dialog.CommandsID)
	case dialog.ActionExternalEditor:
		if m.isAgentBusy() {
			cmds = append(cmds, util.ReportWarn("智能体正在工作，请等待..."))
			break
		}
		cmds = append(cmds, m.openEditor(m.textarea.Value()))
		m.dialog.CloseDialog(dialog.CommandsID)
	case dialog.ActionToggleCompactMode:
		cmds = append(cmds, m.toggleCompactMode())
		m.dialog.CloseDialog(dialog.CommandsID)
	case dialog.ActionToggleThinking:
		cmds = append(cmds, func() tea.Msg {
			cfg := m.com.Config()
			if cfg == nil {
				return util.ReportError(errors.New("未找到配置"))()
			}

			agentCfg, ok := cfg.Agents[config.AgentCoder]
			if !ok {
				return util.ReportError(errors.New("未找到智能体配置"))()
			}

			currentModel := cfg.Models[agentCfg.Model]
			currentModel.Think = !currentModel.Think
			if err := cfg.UpdatePreferredModel(agentCfg.Model, currentModel); err != nil {
				return util.ReportError(err)()
			}
			m.com.App.UpdateAgentModel(context.TODO())
			status := "disabled"
			if currentModel.Think {
				status = "enabled"
			}
			return util.NewInfoMsg("思考模式 " + status)
		})
		m.dialog.CloseDialog(dialog.CommandsID)
	case dialog.ActionQuit:
		cmds = append(cmds, tea.Quit)
	case dialog.ActionInitializeProject:
		if m.isAgentBusy() {
			cmds = append(cmds, util.ReportWarn("智能体忙碌，请等待后再总结会话..."))
			break
		}
		cmds = append(cmds, m.initializeProject())
		m.dialog.CloseDialog(dialog.CommandsID)

	case dialog.ActionSelectModel:
		if m.isAgentBusy() {
			cmds = append(cmds, util.ReportWarn("智能体忙碌，请等待..."))
			break
		}

		cfg := m.com.Config()
		if cfg == nil {
			cmds = append(cmds, util.ReportError(errors.New("未找到配置")))
			break
		}

		var (
			providerID   = msg.Model.Provider
			isCopilot    = providerID == string(catwalk.InferenceProviderCopilot)
			isConfigured = func() bool { _, ok := cfg.Providers.Get(providerID); return ok }
		)

		// 如果可用，尝试从VSCode导入GitHub Copilot令牌
		if isCopilot && !isConfigured() && !msg.ReAuthenticate {
			m.com.Config().ImportCopilot()
		}

		if !isConfigured() || msg.ReAuthenticate {
			m.dialog.CloseDialog(dialog.ModelsID)
			if cmd := m.openAuthenticationDialog(msg.Provider, msg.Model, msg.ModelType); cmd != nil {
				cmds = append(cmds, cmd)
			}
			break
		}

		if err := cfg.UpdatePreferredModel(msg.ModelType, msg.Model); err != nil {
			cmds = append(cmds, util.ReportError(err))
		} else if _, ok := cfg.Models[config.SelectedModelTypeSmall]; !ok {
			// 确保小模型已设置
			smallModel := m.com.App.GetDefaultSmallModel(providerID)
			if err := cfg.UpdatePreferredModel(config.SelectedModelTypeSmall, smallModel); err != nil {
				cmds = append(cmds, util.ReportError(err))
			}
		}

		cmds = append(cmds, func() tea.Msg {
			if err := m.com.App.UpdateAgentModel(context.TODO()); err != nil {
				return util.ReportError(err)
			}

			modelMsg := fmt.Sprintf("%s 模型已更改为 %s", msg.ModelType, msg.Model.Model)

			return util.NewInfoMsg(modelMsg)
		})

		m.dialog.CloseDialog(dialog.APIKeyInputID)
		m.dialog.CloseDialog(dialog.OAuthID)
		m.dialog.CloseDialog(dialog.ModelsID)

		if isOnboarding {
			m.setState(uiLanding, uiFocusEditor)
			m.com.Config().SetupAgents()
			if err := m.com.App.InitCoderAgent(context.TODO()); err != nil {
				cmds = append(cmds, util.ReportError(err))
			}
		}
	case dialog.ActionSelectReasoningEffort:
		if m.isAgentBusy() {
			cmds = append(cmds, util.ReportWarn("智能体忙碌，请等待..."))
			break
		}

		cfg := m.com.Config()
		if cfg == nil {
			cmds = append(cmds, util.ReportError(errors.New("未找到配置")))
			break
		}

		agentCfg, ok := cfg.Agents[config.AgentCoder]
		if !ok {
			cmds = append(cmds, util.ReportError(errors.New("未找到智能体配置")))
			break
		}

		currentModel := cfg.Models[agentCfg.Model]
		currentModel.ReasoningEffort = msg.Effort
		if err := cfg.UpdatePreferredModel(agentCfg.Model, currentModel); err != nil {
			cmds = append(cmds, util.ReportError(err))
			break
		}

		cmds = append(cmds, func() tea.Msg {
			m.com.App.UpdateAgentModel(context.TODO())
			return util.NewInfoMsg("推理努力设置为 " + msg.Effort)
		})
		m.dialog.CloseDialog(dialog.ReasoningID)
	case dialog.ActionPermissionResponse:
		m.dialog.CloseDialog(dialog.PermissionsID)
		switch msg.Action {
		case dialog.PermissionAllow:
			m.com.App.Permissions.Grant(msg.Permission)
		case dialog.PermissionAllowForSession:
			m.com.App.Permissions.GrantPersistent(msg.Permission)
		case dialog.PermissionDeny:
			m.com.App.Permissions.Deny(msg.Permission)
		}

	case dialog.ActionFilePickerSelected:
		cmds = append(cmds, tea.Sequence(
			msg.Cmd(),
			func() tea.Msg {
				m.dialog.CloseDialog(dialog.FilePickerID)
				return nil
			},
			func() tea.Msg {
				fimage.ResetCache()
				return nil
			},
		))

	case dialog.ActionRunCustomCommand:
		if len(msg.Arguments) > 0 && msg.Args == nil {
			m.dialog.CloseFrontDialog()
			argsDialog := dialog.NewArguments(
				m.com,
				"自定义命令参数",
				"",
				msg.Arguments,
				msg, // 将操作作为结果传递
			)
			m.dialog.OpenDialog(argsDialog)
			break
		}
		content := msg.Content
		if msg.Args != nil {
			content = substituteArgs(content, msg.Args)
		}
		cmds = append(cmds, m.sendMessage(content))
		m.dialog.CloseFrontDialog()
	case dialog.ActionRunMCPPrompt:
		if len(msg.Arguments) > 0 && msg.Args == nil {
			m.dialog.CloseFrontDialog()
			title := msg.Title
			if title == "" {
				title = "MCP提示参数"
			}
			argsDialog := dialog.NewArguments(
				m.com,
				title,
				msg.Description,
				msg.Arguments,
				msg, // 将操作作为结果传递
			)
			m.dialog.OpenDialog(argsDialog)
			break
		}
		cmds = append(cmds, m.runMCPPrompt(msg.ClientID, msg.PromptID, msg.Args))
	default:
		cmds = append(cmds, util.CmdHandler(msg))
	}

	return tea.Batch(cmds...)
}

// substituteArgs 用实际值替换内容中的$ARG_NAME占位符
func substituteArgs(content string, args map[string]string) string {
	for name, value := range args {
		placeholder := "$" + name
		content = strings.ReplaceAll(content, placeholder, value)
	}
	return content
}

func (m *UI) openAuthenticationDialog(provider catwalk.Provider, model config.SelectedModel, modelType config.SelectedModelType) tea.Cmd {
	var (
		dlg dialog.Dialog
		cmd tea.Cmd

		isOnboarding = m.state == uiOnboarding
	)

	switch provider.ID {
	case "hyper":
		dlg, cmd = dialog.NewOAuthHyper(m.com, isOnboarding, provider, model, modelType)
	case catwalk.InferenceProviderCopilot:
		dlg, cmd = dialog.NewOAuthCopilot(m.com, isOnboarding, provider, model, modelType)
	default:
		dlg, cmd = dialog.NewAPIKeyInput(m.com, isOnboarding, provider, model, modelType)
	}

	if m.dialog.ContainsDialog(dlg.ID()) {
		m.dialog.BringToFront(dlg.ID())
		return nil
	}

	m.dialog.OpenDialog(dlg)
	return cmd
}

func (m *UI) handleKeyPressMsg(msg tea.KeyPressMsg) tea.Cmd {
	var cmds []tea.Cmd

	handleGlobalKeys := func(msg tea.KeyPressMsg) bool {
		switch {
		case key.Matches(msg, m.keyMap.Help):
			m.status.ToggleHelp()
			m.updateLayoutAndSize()
			return true
		case key.Matches(msg, m.keyMap.Commands):
			if cmd := m.openCommandsDialog(); cmd != nil {
				cmds = append(cmds, cmd)
			}
			return true
		case key.Matches(msg, m.keyMap.Models):
			if cmd := m.openModelsDialog(); cmd != nil {
				cmds = append(cmds, cmd)
			}
			return true
		case key.Matches(msg, m.keyMap.Sessions):
			if cmd := m.openSessionsDialog(); cmd != nil {
				cmds = append(cmds, cmd)
			}
			return true
		case key.Matches(msg, m.keyMap.Chat.Details) && m.isCompact:
			m.detailsOpen = !m.detailsOpen
			m.updateLayoutAndSize()
			return true
		case key.Matches(msg, m.keyMap.Chat.TogglePills):
			if m.state == uiChat && m.hasSession() {
				if cmd := m.togglePillsExpanded(); cmd != nil {
					cmds = append(cmds, cmd)
				}
				return true
			}
		case key.Matches(msg, m.keyMap.Chat.PillLeft):
			if m.state == uiChat && m.hasSession() && m.pillsExpanded && m.focus != uiFocusEditor {
				if cmd := m.switchPillSection(-1); cmd != nil {
					cmds = append(cmds, cmd)
				}
				return true
			}
		case key.Matches(msg, m.keyMap.Chat.PillRight):
			if m.state == uiChat && m.hasSession() && m.pillsExpanded && m.focus != uiFocusEditor {
				if cmd := m.switchPillSection(1); cmd != nil {
					cmds = append(cmds, cmd)
				}
				return true
			}
		case key.Matches(msg, m.keyMap.Suspend):
			if m.isAgentBusy() {
				cmds = append(cmds, util.ReportWarn("智能体忙碌，请等待..."))
				return true
			}
			cmds = append(cmds, tea.Suspend)
			return true
		}
		return false
	}

	if key.Matches(msg, m.keyMap.Quit) && !m.dialog.ContainsDialog(dialog.QuitID) {
		// 始终先处理退出键
		if cmd := m.openQuitDialog(); cmd != nil {
			cmds = append(cmds, cmd)
		}

		return tea.Batch(cmds...)
	}

	// 如果对话框打开，将所有消息路由到对话框
	if m.dialog.HasDialogs() {
		return m.handleDialogMsg(msg)
	}

	// 当智能体忙碌时处理取消键
	if key.Matches(msg, m.keyMap.Chat.Cancel) {
		if m.isAgentBusy() {
			if cmd := m.cancelAgent(); cmd != nil {
				cmds = append(cmds, cmd)
			}
			return tea.Batch(cmds...)
		}
	}

	switch m.state {
	case uiOnboarding:
		return tea.Batch(cmds...)
	case uiInitialize:
		cmds = append(cmds, m.updateInitializeView(msg)...)
		return tea.Batch(cmds...)
	case uiChat, uiLanding:
		switch m.focus {
		case uiFocusEditor:
			// 如果自动完成打开，则处理
			if m.completionsOpen {
				if msg, ok := m.completions.Update(msg); ok {
					switch msg := msg.(type) {
					case completions.SelectionMsg[completions.FileCompletionValue]:
						cmds = append(cmds, m.insertFileCompletion(msg.Value.Path))
						if !msg.KeepOpen {
							m.closeCompletions()
						}
					case completions.SelectionMsg[completions.ResourceCompletionValue]:
						cmds = append(cmds, m.insertMCPResourceCompletion(msg.Value))
						if !msg.KeepOpen {
							m.closeCompletions()
						}
					case completions.ClosedMsg:
						m.completionsOpen = false
					}
					return tea.Batch(cmds...)
				}
			}

			if ok := m.attachments.Update(msg); ok {
				return tea.Batch(cmds...)
			}

			switch {
			case key.Matches(msg, m.keyMap.Editor.AddImage):
				if cmd := m.openFilesDialog(); cmd != nil {
					cmds = append(cmds, cmd)
				}

			case key.Matches(msg, m.keyMap.Editor.PasteImage):
				cmds = append(cmds, m.pasteImageFromClipboard)

			case key.Matches(msg, m.keyMap.Editor.SendMessage):
				value := m.textarea.Value()
				if before, ok := strings.CutSuffix(value, "\\"); ok {
					// 如果最后一个字符是反斜杠，则删除它并添加换行符
					m.textarea.SetValue(before)
					break
				}

				// 否则，发送消息
				m.textarea.Reset()

				value = strings.TrimSpace(value)
				if value == "exit" || value == "quit" {
					return m.openQuitDialog()
				}

				attachments := m.attachments.List()
				m.attachments.Reset()
				if len(value) == 0 && !message.ContainsTextAttachment(attachments) {
					return nil
				}

				m.randomizePlaceholders()
				m.historyReset()

				return tea.Batch(m.sendMessage(value, attachments...), m.loadPromptHistory())
			case key.Matches(msg, m.keyMap.Chat.NewSession):
				if !m.hasSession() {
					break
				}
				if m.isAgentBusy() {
					cmds = append(cmds, util.ReportWarn("智能体忙碌，请等待后再开始新会话..."))
					break
				}
				if cmd := m.newSession(); cmd != nil {
					cmds = append(cmds, cmd)
				}
			case key.Matches(msg, m.keyMap.Tab):
				if m.state != uiLanding {
					m.setState(m.state, uiFocusMain)
					m.textarea.Blur()
					m.chat.Focus()
					m.chat.SetSelected(m.chat.Len() - 1)
				}
			case key.Matches(msg, m.keyMap.Editor.OpenEditor):
				if m.isAgentBusy() {
					cmds = append(cmds, util.ReportWarn("智能体正在工作，请等待..."))
					break
				}
				cmds = append(cmds, m.openEditor(m.textarea.Value()))
			case key.Matches(msg, m.keyMap.Editor.Newline):
				m.textarea.InsertRune('\n')
				m.closeCompletions()
				ta, cmd := m.textarea.Update(msg)
				m.textarea = ta
				cmds = append(cmds, cmd)
			case key.Matches(msg, m.keyMap.Editor.HistoryPrev):
				cmd := m.handleHistoryUp(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			case key.Matches(msg, m.keyMap.Editor.HistoryNext):
				cmd := m.handleHistoryDown(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			case key.Matches(msg, m.keyMap.Editor.Escape):
				cmd := m.handleHistoryEscape(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			case key.Matches(msg, m.keyMap.Editor.Commands) && m.textarea.Value() == "":
				if cmd := m.openCommandsDialog(); cmd != nil {
					cmds = append(cmds, cmd)
				}
			default:
				if handleGlobalKeys(msg) {
					// 在传递给文本区域之前先处理全局键
					break
				}

				// 在传递给文本区域之前检查@触发器
				curValue := m.textarea.Value()
				curIdx := len(curValue)

				// 在@上触发自动完成
				if msg.String() == "@" && !m.completionsOpen {
					// 仅在提示开头或空白字符之后显示
					if curIdx == 0 || (curIdx > 0 && isWhitespace(curValue[curIdx-1])) {
						m.completionsOpen = true
						m.completionsQuery = ""
						m.completionsStartIndex = curIdx
						m.completionsPositionStart = m.completionsPosition()
						depth, limit := m.com.Config().Options.TUI.Completions.Limits()
						cmds = append(cmds, m.completions.Open(depth, limit))
					}
				}

				// 如果用户开始输入时详情打开，则移除详情
				if m.detailsOpen {
					m.detailsOpen = false
					m.updateLayoutAndSize()
				}

				ta, cmd := m.textarea.Update(msg)
				m.textarea = ta
				cmds = append(cmds, cmd)

				// 任何文本修改都成为当前草稿
				m.updateHistoryDraft(curValue)

				// 更新文本区域后，检查是否需要过滤自动完成
				// 跳过初始@按键的过滤，因为项目正在异步加载
				if m.completionsOpen && msg.String() != "@" {
					newValue := m.textarea.Value()
					newIdx := len(newValue)

					// 如果光标在开始之前移动，则关闭自动完成
					if newIdx <= m.completionsStartIndex {
						m.closeCompletions()
					} else if msg.String() == "space" {
						// 在空格时关闭
						m.closeCompletions()
					} else {
						// 提取当前单词并过滤
						word := m.textareaWord()
						if strings.HasPrefix(word, "@") {
							m.completionsQuery = word[1:]
							m.completions.Filter(m.completionsQuery)
						} else if m.completionsOpen {
							m.closeCompletions()
						}
					}
				}
			}
		case uiFocusMain:
			switch {
			case key.Matches(msg, m.keyMap.Tab):
				m.focus = uiFocusEditor
				cmds = append(cmds, m.textarea.Focus())
				m.chat.Blur()
			case key.Matches(msg, m.keyMap.Chat.NewSession):
				if !m.hasSession() {
					break
				}
				if m.isAgentBusy() {
					cmds = append(cmds, util.ReportWarn("智能体忙碌，请等待后再开始新会话..."))
					break
				}
				m.focus = uiFocusEditor
				if cmd := m.newSession(); cmd != nil {
					cmds = append(cmds, cmd)
				}
			case key.Matches(msg, m.keyMap.Chat.Expand):
				m.chat.ToggleExpandedSelectedItem()
			case key.Matches(msg, m.keyMap.Chat.Up):
				if cmd := m.chat.ScrollByAndAnimate(-1); cmd != nil {
					cmds = append(cmds, cmd)
				}
				if !m.chat.SelectedItemInView() {
					m.chat.SelectPrev()
					if cmd := m.chat.ScrollToSelectedAndAnimate(); cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			case key.Matches(msg, m.keyMap.Chat.Down):
				if cmd := m.chat.ScrollByAndAnimate(1); cmd != nil {
					cmds = append(cmds, cmd)
				}
				if !m.chat.SelectedItemInView() {
					m.chat.SelectNext()
					if cmd := m.chat.ScrollToSelectedAndAnimate(); cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			case key.Matches(msg, m.keyMap.Chat.UpOneItem):
				m.chat.SelectPrev()
				if cmd := m.chat.ScrollToSelectedAndAnimate(); cmd != nil {
					cmds = append(cmds, cmd)
				}
			case key.Matches(msg, m.keyMap.Chat.DownOneItem):
				m.chat.SelectNext()
				if cmd := m.chat.ScrollToSelectedAndAnimate(); cmd != nil {
					cmds = append(cmds, cmd)
				}
			case key.Matches(msg, m.keyMap.Chat.HalfPageUp):
				if cmd := m.chat.ScrollByAndAnimate(-m.chat.Height() / 2); cmd != nil {
					cmds = append(cmds, cmd)
				}
				m.chat.SelectFirstInView()
			case key.Matches(msg, m.keyMap.Chat.HalfPageDown):
				if cmd := m.chat.ScrollByAndAnimate(m.chat.Height() / 2); cmd != nil {
					cmds = append(cmds, cmd)
				}
				m.chat.SelectLastInView()
			case key.Matches(msg, m.keyMap.Chat.PageUp):
				if cmd := m.chat.ScrollByAndAnimate(-m.chat.Height()); cmd != nil {
					cmds = append(cmds, cmd)
				}
				m.chat.SelectFirstInView()
			case key.Matches(msg, m.keyMap.Chat.PageDown):
				if cmd := m.chat.ScrollByAndAnimate(m.chat.Height()); cmd != nil {
					cmds = append(cmds, cmd)
				}
				m.chat.SelectLastInView()
			case key.Matches(msg, m.keyMap.Chat.Home):
				if cmd := m.chat.ScrollToTopAndAnimate(); cmd != nil {
					cmds = append(cmds, cmd)
				}
				m.chat.SelectFirst()
			case key.Matches(msg, m.keyMap.Chat.End):
				if cmd := m.chat.ScrollToBottomAndAnimate(); cmd != nil {
					cmds = append(cmds, cmd)
				}
				m.chat.SelectLast()
			default:
				if ok, cmd := m.chat.HandleKeyMsg(msg); ok {
					cmds = append(cmds, cmd)
				} else {
					handleGlobalKeys(msg)
				}
			}
		default:
			handleGlobalKeys(msg)
		}
	default:
		handleGlobalKeys(msg)
	}

	return tea.Batch(cmds...)
}

// drawHeader 绘制UI的头部区域
func (m *UI) drawHeader(scr uv.Screen, area uv.Rectangle) {
	m.header.drawHeader(
		scr,
		area,
		m.session,
		m.isCompact,
		m.detailsOpen,
		m.width,
	)
}

// Draw 实现 [uv.Drawable] 并绘制UI模型
func (m *UI) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	layout := m.generateLayout(area.Dx(), area.Dy())

	if m.layout != layout {
		m.layout = layout
		m.updateSize()
	}

	// 首先清除屏幕
	screen.Clear(scr)

	switch m.state {
	case uiOnboarding:
		m.drawHeader(scr, layout.header)

		// 注意：引导流程将在下面作为对话框渲染，
		// 但位于屏幕的左下角

	case uiInitialize:
		m.drawHeader(scr, layout.header)

		main := uv.NewStyledString(m.initializeView())
		main.Draw(scr, layout.main)

	case uiLanding:
		m.drawHeader(scr, layout.header)
		main := uv.NewStyledString(m.landingView())
		main.Draw(scr, layout.main)

		editor := uv.NewStyledString(m.renderEditorView(scr.Bounds().Dx()))
		editor.Draw(scr, layout.editor)

	case uiChat:
		if m.isCompact {
			m.drawHeader(scr, layout.header)
		} else {
			m.drawSidebar(scr, layout.sidebar)
		}

		m.chat.Draw(scr, layout.main)
		if layout.pills.Dy() > 0 && m.pillsView != "" {
			uv.NewStyledString(m.pillsView).Draw(scr, layout.pills)
		}

		editorWidth := scr.Bounds().Dx()
		if !m.isCompact {
			editorWidth -= layout.sidebar.Dx()
		}
		editor := uv.NewStyledString(m.renderEditorView(editorWidth))
		editor.Draw(scr, layout.editor)

		// 在紧凑模式下打开时绘制详情覆盖层
		if m.isCompact && m.detailsOpen {
			m.drawSessionDetails(scr, layout.sessionDetails)
		}
	}

	isOnboarding := m.state == uiOnboarding

	// 添加状态和帮助层
	m.status.SetHideHelp(isOnboarding)
	m.status.Draw(scr, layout.status)

	// 如果打开，绘制自动完成弹出窗口
	if !isOnboarding && m.completionsOpen && m.completions.HasItems() {
		w, h := m.completions.Size()
		x := m.completionsPositionStart.X
		y := m.completionsPositionStart.Y - h

		screenW := area.Dx()
		if x+w > screenW {
			x = screenW - w
		}
		x = max(0, x)
		y = max(0, y)

		completionsView := uv.NewStyledString(m.completions.Render())
		completionsView.Draw(scr, image.Rectangle{
			Min: image.Pt(x, y),
			Max: image.Pt(x+w, y+h),
		})
	}

	// 调试渲染（视觉上查看tui何时重新渲染）
	if os.Getenv("CRUSH_UI_DEBUG") == "true" {
		debugView := lipgloss.NewStyle().Background(lipgloss.ANSIColor(rand.Intn(256))).Width(4).Height(2)
		debug := uv.NewStyledString(debugView.String())
		debug.Draw(scr, image.Rectangle{
			Min: image.Pt(4, 1),
			Max: image.Pt(8, 3),
		})
	}

	// 这需要放在最后以覆盖在所有内容之上。我们始终传递
	// 完整的屏幕边界，因为对话框会相应地定位自己
	if m.dialog.HasDialogs() {
		return m.dialog.Draw(scr, scr.Bounds())
	}

	switch m.focus {
	case uiFocusEditor:
		if m.layout.editor.Dy() <= 0 {
			// 如果编辑器不可见，不显示光标
			return nil
		}
		if m.detailsOpen && m.isCompact {
			// 如果详情覆盖层打开，不显示光标
			return nil
		}

		if m.textarea.Focused() {
			cur := m.textarea.Cursor()
			cur.X++ // 调整应用程序边距
			cur.Y += m.layout.editor.Min.Y
			// 如果存在附件行，则偏移
			if len(m.attachments.List()) > 0 {
				cur.Y++
			}
			return cur
		}
	}
	return nil
}

// View 渲染UI模型的视图
func (m *UI) View() tea.View {
	var v tea.View
	v.AltScreen = true
	if !m.isTransparent {
		v.BackgroundColor = m.com.Styles.Background
	}
	v.MouseMode = tea.MouseModeCellMotion
	v.WindowTitle = "crush " + home.Short(m.com.Config().WorkingDir())

	canvas := uv.NewScreenBuffer(m.width, m.height)
	v.Cursor = m.Draw(canvas, canvas.Bounds())

	content := strings.ReplaceAll(canvas.Render(), "\r\n", "\n") // 规范化换行符
	contentLines := strings.Split(content, "\n")
	for i, line := range contentLines {
		// 去除尾随空格以简洁渲染
		contentLines[i] = strings.TrimRight(line, " ")
	}

	content = strings.Join(contentLines, "\n")

	v.Content = content
	if m.progressBarEnabled && m.sendProgressBar && m.isAgentBusy() {
		// HACK: 使用随机百分比以防止ghostty在超时后隐藏它
		v.ProgressBar = tea.NewProgressBar(tea.ProgressBarIndeterminate, rand.Intn(100))
	}

	return v
}

// ShortHelp 实现 [help.KeyMap]
func (m *UI) ShortHelp() []key.Binding {
	var binds []key.Binding
	k := &m.keyMap
	tab := k.Tab
	commands := k.Commands
	if m.focus == uiFocusEditor && m.textarea.Value() == "" {
		commands.SetHelp("/ or ctrl+p", "命令")
	}

	switch m.state {
	case uiInitialize:
		binds = append(binds, k.Quit)
	case uiChat:
		// 如果智能体忙碌，显示取消绑定
		if m.isAgentBusy() {
			cancelBinding := k.Chat.Cancel
			if m.isCanceling {
				cancelBinding.SetHelp("esc", "再次按下取消")
			} else if m.com.App.AgentCoordinator.QueuedPrompts(m.session.ID) > 0 {
				cancelBinding.SetHelp("esc", "清除队列")
			}
			binds = append(binds, cancelBinding)
		}

		if m.focus == uiFocusEditor {
			tab.SetHelp("tab", "聚焦聊天")
		} else {
			tab.SetHelp("tab", "聚焦编辑器")
		}

		binds = append(binds,
			tab,
			commands,
			k.Models,
		)

		switch m.focus {
		case uiFocusEditor:
			binds = append(binds,
				k.Editor.Newline,
			)
		case uiFocusMain:
			binds = append(binds,
				k.Chat.UpDown,
				k.Chat.UpDownOneItem,
				k.Chat.PageUp,
				k.Chat.PageDown,
				k.Chat.Copy,
			)
			if m.pillsExpanded && hasIncompleteTodos(m.session.Todos) && m.promptQueue > 0 {
				binds = append(binds, k.Chat.PillLeft)
			}
		}
	default:
		// TODO: 其他状态
		// if m.session == nil {
		// no session selected
		binds = append(binds,
			commands,
			k.Models,
			k.Editor.Newline,
		)
	}

	binds = append(binds,
		k.Quit,
		k.Help,
	)

	return binds
}

// FullHelp 实现 [help.KeyMap]
func (m *UI) FullHelp() [][]key.Binding {
	var binds [][]key.Binding
	k := &m.keyMap
	help := k.Help
	help.SetHelp("ctrl+g", "less")
	hasAttachments := len(m.attachments.List()) > 0
	hasSession := m.hasSession()
	commands := k.Commands
	if m.focus == uiFocusEditor && m.textarea.Value() == "" {
		commands.SetHelp("/ or ctrl+p", "命令")
	}

	switch m.state {
	case uiInitialize:
		binds = append(binds,
			[]key.Binding{
				k.Quit,
			})
	case uiChat:
		// 如果智能体忙碌，显示取消绑定
		if m.isAgentBusy() {
			cancelBinding := k.Chat.Cancel
			if m.isCanceling {
				cancelBinding.SetHelp("esc", "再次按下取消")
			} else if m.com.App.AgentCoordinator.QueuedPrompts(m.session.ID) > 0 {
				cancelBinding.SetHelp("esc", "清除队列")
			}
			binds = append(binds, []key.Binding{cancelBinding})
		}

		mainBinds := []key.Binding{}
		tab := k.Tab
		if m.focus == uiFocusEditor {
			tab.SetHelp("tab", "聚焦聊天")
		} else {
			tab.SetHelp("tab", "聚焦编辑器")
		}

		mainBinds = append(mainBinds,
			tab,
			commands,
			k.Models,
			k.Sessions,
		)
		if hasSession {
			mainBinds = append(mainBinds, k.Chat.NewSession)
		}

		binds = append(binds, mainBinds)

		switch m.focus {
		case uiFocusEditor:
			binds = append(binds,
				[]key.Binding{
					k.Editor.Newline,
					k.Editor.AddImage,
					k.Editor.PasteImage,
					k.Editor.MentionFile,
					k.Editor.OpenEditor,
				},
			)
			if hasAttachments {
				binds = append(binds,
					[]key.Binding{
						k.Editor.AttachmentDeleteMode,
						k.Editor.DeleteAllAttachments,
						k.Editor.Escape,
					},
				)
			}
		case uiFocusMain:
			binds = append(binds,
				[]key.Binding{
					k.Chat.UpDown,
					k.Chat.UpDownOneItem,
					k.Chat.PageUp,
					k.Chat.PageDown,
				},
				[]key.Binding{
					k.Chat.HalfPageUp,
					k.Chat.HalfPageDown,
					k.Chat.Home,
					k.Chat.End,
				},
				[]key.Binding{
					k.Chat.Copy,
					k.Chat.ClearHighlight,
				},
			)
			if m.pillsExpanded && hasIncompleteTodos(m.session.Todos) && m.promptQueue > 0 {
				binds = append(binds, []key.Binding{k.Chat.PillLeft})
			}
		}
	default:
		if m.session == nil {
			// no session selected
			binds = append(binds,
				[]key.Binding{
					commands,
					k.Models,
					k.Sessions,
				},
				[]key.Binding{
					k.Editor.Newline,
					k.Editor.AddImage,
					k.Editor.PasteImage,
					k.Editor.MentionFile,
					k.Editor.OpenEditor,
				},
			)
			if hasAttachments {
				binds = append(binds,
					[]key.Binding{
						k.Editor.AttachmentDeleteMode,
						k.Editor.DeleteAllAttachments,
						k.Editor.Escape,
					},
				)
			}
			binds = append(binds,
				[]key.Binding{
					help,
				},
			)
		}
	}

	binds = append(binds,
		[]key.Binding{
			help,
			k.Quit,
		},
	)

	return binds
}

// toggleCompactMode 在uiChat和uiChatCompact状态之间切换紧凑模式
func (m *UI) toggleCompactMode() tea.Cmd {
	m.forceCompactMode = !m.forceCompactMode

	err := m.com.Config().SetCompactMode(m.forceCompactMode)
	if err != nil {
		return util.ReportError(err)
	}

	m.updateLayoutAndSize()

	return nil
}

// updateLayoutAndSize 更新UI组件的布局和大小
func (m *UI) updateLayoutAndSize() {
	// 确定我们是否应该处于紧凑模式
	if m.state == uiChat {
		if m.forceCompactMode {
			m.isCompact = true
			return
		}
		if m.width < compactModeWidthBreakpoint || m.height < compactModeHeightBreakpoint {
			m.isCompact = true
		} else {
			m.isCompact = false
		}
	}

	m.layout = m.generateLayout(m.width, m.height)
	m.updateSize()
}

// updateSize 根据当前布局更新UI组件的大小
func (m *UI) updateSize() {
	// 设置状态宽度
	m.status.SetWidth(m.layout.status.Dx())

	m.chat.SetSize(m.layout.main.Dx(), m.layout.main.Dy())
	m.textarea.SetWidth(m.layout.editor.Dx())
	m.textarea.SetHeight(m.layout.editor.Dy())
	m.renderPills()

	// 处理不同的应用程序状态
	switch m.state {
	case uiChat:
		if !m.isCompact {
			m.cacheSidebarLogo(m.layout.sidebar.Dx())
		}
	}
}

// generateLayout 根据当前UI状态和终端尺寸
// 计算所有UI组件的布局矩形
func (m *UI) generateLayout(w, h int) uiLayout {
	// 我们正在使用的屏幕区域
	area := image.Rect(0, 0, w, h)

	// 帮助高度
	helpHeight := 1
	// 编辑器高度
	editorHeight := 5
	// 侧边栏宽度
	sidebarWidth := 30
	// 头部高度
	const landingHeaderHeight = 4

	var helpKeyMap help.KeyMap = m
	if m.status != nil && m.status.ShowingAll() {
		for _, row := range helpKeyMap.FullHelp() {
			helpHeight = max(helpHeight, len(row))
		}
	}

	// 添加应用程序边距
	appRect, helpRect := layout.SplitVertical(area, layout.Fixed(area.Dy()-helpHeight))
	appRect.Min.Y += 1
	appRect.Max.Y -= 1
	helpRect.Min.Y -= 1
	appRect.Min.X += 1
	appRect.Max.X -= 1

	if slices.Contains([]uiState{uiOnboarding, uiInitialize, uiLanding}, m.state) {
		// 为这些状态在左右两侧添加额外的填充
		appRect.Min.X += 1
		appRect.Max.X -= 1
	}

	uiLayout := uiLayout{
		area:   area,
		status: helpRect,
	}

	// 处理不同的应用程序状态
	switch m.state {
	case uiOnboarding, uiInitialize:
		// 布局
		//
		// header
		// ------
		// main
		// ------
		// help

		headerRect, mainRect := layout.SplitVertical(appRect, layout.Fixed(landingHeaderHeight))
		uiLayout.header = headerRect
		uiLayout.main = mainRect

	case uiLanding:
		// 布局
		//
		// 头部
		// ------
		// 主体
		// ------
		// 编辑器
		// ------
		// 帮助
		headerRect, mainRect := layout.SplitVertical(appRect, layout.Fixed(landingHeaderHeight))
		mainRect, editorRect := layout.SplitVertical(mainRect, layout.Fixed(mainRect.Dy()-editorHeight))
		// 从编辑器中移除额外填充（但为头部和主体保留）
		editorRect.Min.X -= 1
		editorRect.Max.X += 1
		uiLayout.header = headerRect
		uiLayout.main = mainRect
		uiLayout.editor = editorRect

	case uiChat:
		if m.isCompact {
			// 布局
			//
			// 紧凑头部
			// ------
			// 主体
			// ------
			// 编辑器
			// ------
			// 帮助
			const compactHeaderHeight = 1
			headerRect, mainRect := layout.SplitVertical(appRect, layout.Fixed(compactHeaderHeight))
			detailsHeight := min(sessionDetailsMaxHeight, area.Dy()-1) // 头部的一行
			sessionDetailsArea, _ := layout.SplitVertical(appRect, layout.Fixed(detailsHeight))
			uiLayout.sessionDetails = sessionDetailsArea
			uiLayout.sessionDetails.Min.Y += compactHeaderHeight // 调整头部
			// 在头部和主体内容之间添加一行间隙
			mainRect.Min.Y += 1
			mainRect, editorRect := layout.SplitVertical(mainRect, layout.Fixed(mainRect.Dy()-editorHeight))
			mainRect.Max.X -= 1 // 添加右侧填充
			uiLayout.header = headerRect
			pillsHeight := m.pillsAreaHeight()
			if pillsHeight > 0 {
				pillsHeight = min(pillsHeight, mainRect.Dy())
				chatRect, pillsRect := layout.SplitVertical(mainRect, layout.Fixed(mainRect.Dy()-pillsHeight))
				uiLayout.main = chatRect
				uiLayout.pills = pillsRect
			} else {
				uiLayout.main = mainRect
			}
			// 为主体添加底部边距
			uiLayout.main.Max.Y -= 1
			uiLayout.editor = editorRect
		} else {
			// 布局
			//
			// ------|---
			// 主体  |
			// ------| 侧边栏
			// 编辑器|
			// ----------
			// 帮助

			mainRect, sideRect := layout.SplitHorizontal(appRect, layout.Fixed(appRect.Dx()-sidebarWidth))
			// 添加左侧填充
			sideRect.Min.X += 1
			mainRect, editorRect := layout.SplitVertical(mainRect, layout.Fixed(mainRect.Dy()-editorHeight))
			mainRect.Max.X -= 1 // 添加右侧填充
			uiLayout.sidebar = sideRect
			pillsHeight := m.pillsAreaHeight()
			if pillsHeight > 0 {
				pillsHeight = min(pillsHeight, mainRect.Dy())
				chatRect, pillsRect := layout.SplitVertical(mainRect, layout.Fixed(mainRect.Dy()-pillsHeight))
				uiLayout.main = chatRect
				uiLayout.pills = pillsRect
			} else {
				uiLayout.main = mainRect
			}
			// 为主体添加底部边距
			uiLayout.main.Max.Y -= 1
			uiLayout.editor = editorRect
		}
	}

	if !uiLayout.editor.Empty() {
		// 添加编辑器上下边距1
		if len(m.attachments.List()) == 0 {
			uiLayout.editor.Min.Y += 1
		}
		uiLayout.editor.Max.Y -= 1
	}

	return uiLayout
}

// uiLayout 定义UI元素的定位
type uiLayout struct {
	// area 是整体可用区域
	area uv.Rectangle

	// header 是在特殊情况下显示的头部
	// 例如当侧边栏折叠时
	// 或在着陆页面时
	// 或在初始化/配置时
	header uv.Rectangle

	// main 是主面板的区域（例如聊天、配置、着陆页面）
	main uv.Rectangle

	// pills 是药丸面板的区域
	pills uv.Rectangle

	// editor 是编辑器面板的区域
	editor uv.Rectangle

	// sidebar 是侧边栏的区域
	sidebar uv.Rectangle

	// status 是状态视图的区域
	status uv.Rectangle

	// session details 是紧凑模式下会话详情覆盖层的区域
	sessionDetails uv.Rectangle
}

func (m *UI) openEditor(value string) tea.Cmd {
	tmpfile, err := os.CreateTemp("", "msg_*.md")
	if err != nil {
		return util.ReportError(err)
	}
	defer tmpfile.Close() //nolint:errcheck
	if _, err := tmpfile.WriteString(value); err != nil {
		return util.ReportError(err)
	}
	cmd, err := editor.Command(
		"crush",
		tmpfile.Name(),
		editor.AtPosition(
			m.textarea.Line()+1,
			m.textarea.Column()+1,
		),
	)
	if err != nil {
		return util.ReportError(err)
	}
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return util.ReportError(err)
		}
		content, err := os.ReadFile(tmpfile.Name())
		if err != nil {
			return util.ReportError(err)
		}
		if len(content) == 0 {
			return util.ReportWarn("消息为空")
		}
		os.Remove(tmpfile.Name())
		return openEditorMsg{
			Text: strings.TrimSpace(string(content)),
		}
	})
}

// setEditorPrompt 根据是否启用了yolo模式配置文本区域提示函数
func (m *UI) setEditorPrompt(yolo bool) {
	if yolo {
		m.textarea.SetPromptFunc(4, m.yoloPromptFunc)
		return
	}
	m.textarea.SetPromptFunc(4, m.normalPromptFunc)
}

// normalPromptFunc 返回正常的编辑器提示样式（第一行"  > "，
// 后续行"::: "）
func (m *UI) normalPromptFunc(info textarea.PromptInfo) string {
	t := m.com.Styles
	if info.LineNumber == 0 {
		if info.Focused {
			return "  > "
		}
		return "::: "
	}
	if info.Focused {
		return t.EditorPromptNormalFocused.Render()
	}
	return t.EditorPromptNormalBlurred.Render()
}

// yoloPromptFunc 返回yolo模式编辑器提示样式，带有警告图标
// 和彩色点
func (m *UI) yoloPromptFunc(info textarea.PromptInfo) string {
	t := m.com.Styles
	if info.LineNumber == 0 {
		if info.Focused {
			return t.EditorPromptYoloIconFocused.Render()
		} else {
			return t.EditorPromptYoloIconBlurred.Render()
		}
	}
	if info.Focused {
		return t.EditorPromptYoloDotsFocused.Render()
	}
	return t.EditorPromptYoloDotsBlurred.Render()
}

// closeCompletions 关闭自动完成弹出窗口并重置状态
func (m *UI) closeCompletions() {
	m.completionsOpen = false
	m.completionsQuery = ""
	m.completionsStartIndex = 0
	m.completions.Close()
}

// insertCompletionText 用给定文本替换文本区域中的@query
// 如果无法执行替换，则返回false
func (m *UI) insertCompletionText(text string) bool {
	value := m.textarea.Value()
	if m.completionsStartIndex > len(value) {
		return false
	}

	word := m.textareaWord()
	endIdx := min(m.completionsStartIndex+len(word), len(value))
	newValue := value[:m.completionsStartIndex] + text + value[endIdx:]
	m.textarea.SetValue(newValue)
	m.textarea.MoveToEnd()
	m.textarea.InsertRune(' ')
	return true
}

// insertFileCompletion 将选定的文件路径插入到文本区域中，
// 替换@query，并将文件添加为附件
func (m *UI) insertFileCompletion(path string) tea.Cmd {
	if !m.insertCompletionText(path) {
		return nil
	}

	return func() tea.Msg {
		absPath, _ := filepath.Abs(path)

		if m.hasSession() {
			// 如果文件已被读取且未被修改，则跳过附件
			lastRead := m.com.App.FileTracker.LastReadTime(context.Background(), m.session.ID, absPath)
			if !lastRead.IsZero() {
				if info, err := os.Stat(path); err == nil && !info.ModTime().After(lastRead) {
					return nil
				}
			}
		} else if slices.Contains(m.sessionFileReads, absPath) {
			return nil
		}

		m.sessionFileReads = append(m.sessionFileReads, absPath)

		// 将文件添加为附件
		content, err := os.ReadFile(path)
		if err != nil {
			// 如果失败，让LLM稍后处理
			return nil
		}

		return message.Attachment{
			FilePath: path,
			FileName: filepath.Base(path),
			MimeType: mimeOf(content),
			Content:  content,
		}
	}
}

// insertMCPResourceCompletion 将选定的资源插入到文本区域中，
// 替换@query，并将资源添加为附件
func (m *UI) insertMCPResourceCompletion(item completions.ResourceCompletionValue) tea.Cmd {
	displayText := item.Title
	if displayText == "" {
		displayText = item.URI
	}

	if !m.insertCompletionText(displayText) {
		return nil
	}

	return func() tea.Msg {
		contents, err := mcp.ReadResource(
			context.Background(),
			m.com.Config(),
			item.MCPName,
			item.URI,
		)
		if err != nil {
			slog.Warn("读取MCP资源失败", "uri", item.URI, "error", err)
			return nil
		}
		if len(contents) == 0 {
			return nil
		}

		content := contents[0]
		var data []byte
		if content.Text != "" {
			data = []byte(content.Text)
		} else if len(content.Blob) > 0 {
			data = content.Blob
		}
		if len(data) == 0 {
			return nil
		}

		mimeType := item.MIMEType
		if mimeType == "" && content.MIMEType != "" {
			mimeType = content.MIMEType
		}
		if mimeType == "" {
			mimeType = "text/plain"
		}

		return message.Attachment{
			FilePath: item.URI,
			FileName: displayText,
			MimeType: mimeType,
			Content:  data,
		}
	}
}

// completionsPosition 返回自动完成弹出窗口的X和Y位置
func (m *UI) completionsPosition() image.Point {
	cur := m.textarea.Cursor()
	if cur == nil {
		return image.Point{
			X: m.layout.editor.Min.X,
			Y: m.layout.editor.Min.Y,
		}
	}
	return image.Point{
		X: cur.X + m.layout.editor.Min.X,
		Y: m.layout.editor.Min.Y + cur.Y,
	}
}

// textareaWord 返回光标位置处的当前单词
func (m *UI) textareaWord() string {
	return m.textarea.Word()
}

// isWhitespace 如果字节是空白字符，则返回true
func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// isAgentBusy 如果智能体协调器存在且当前
// 忙于处理请求，则返回true
func (m *UI) isAgentBusy() bool {
	return m.com.App != nil &&
		m.com.App.AgentCoordinator != nil &&
		m.com.App.AgentCoordinator.IsBusy()
}

// hasSession 如果存在具有有效ID的活动会话，则返回true
func (m *UI) hasSession() bool {
	return m.session != nil && m.session.ID != ""
}

// mimeOf 检测给定内容的MIME类型
func mimeOf(content []byte) string {
	mimeBufferSize := min(512, len(content))
	return http.DetectContentType(content[:mimeBufferSize])
}

var readyPlaceholders = [...]string{
	"就绪！",
	"就绪...",
	"就绪？",
	"准备接收指令",
}

var workingPlaceholders = [...]string{
	"工作中！",
	"工作中...",
	"Brrrrr...",
	"Prrrrrrrr...",
	"处理中...",
	"思考中...",
}

// randomizePlaceholders 为文本区域的就绪和工作状态
// 选择随机占位符文本
func (m *UI) randomizePlaceholders() {
	m.workingPlaceholder = workingPlaceholders[rand.Intn(len(workingPlaceholders))]
	m.readyPlaceholder = readyPlaceholders[rand.Intn(len(readyPlaceholders))]
}

// renderEditorView 渲染编辑器视图，如果有附件则包含附件
func (m *UI) renderEditorView(width int) string {
	if len(m.attachments.List()) == 0 {
		return m.textarea.View()
	}
	return lipgloss.JoinVertical(
		lipgloss.Top,
		m.attachments.Render(width),
		m.textarea.View(),
	)
}

// cacheSidebarLogo 渲染并缓存指定宽度的侧边栏logo
func (m *UI) cacheSidebarLogo(width int) {
	m.sidebarLogo = renderLogo(m.com.Styles, true, width)
}

// sendMessage 发送具有给定内容和附件的消息
func (m *UI) sendMessage(content string, attachments ...message.Attachment) tea.Cmd {
	if m.com.App.AgentCoordinator == nil {
		return util.ReportError(fmt.Errorf("编码器智能体未初始化"))
	}

	var cmds []tea.Cmd
	if !m.hasSession() {
		newSession, err := m.com.App.Sessions.Create(context.Background(), "新会话")
		if err != nil {
			return util.ReportError(err)
		}
		if m.forceCompactMode {
			m.isCompact = true
		}
		if newSession.ID != "" {
			m.session = &newSession
			cmds = append(cmds, m.loadSession(newSession.ID))
		}
		m.setState(uiChat, m.focus)
	}

	ctx := context.Background()
	cmds = append(cmds, func() tea.Msg {
		for _, path := range m.sessionFileReads {
			m.com.App.FileTracker.RecordRead(ctx, m.session.ID, path)
			m.com.App.LSPManager.Start(ctx, path)
		}
		return nil
	})

	// 捕获会话ID以避免与主goroutine更新m.session竞争
	sessionID := m.session.ID
	cmds = append(cmds, func() tea.Msg {
		_, err := m.com.App.AgentCoordinator.Run(context.Background(), sessionID, content, attachments...)
		if err != nil {
			isCancelErr := errors.Is(err, context.Canceled)
			isPermissionErr := errors.Is(err, permission.ErrorPermissionDenied)
			if isCancelErr || isPermissionErr {
				return nil
			}
			return util.InfoMsg{
				Type: util.InfoTypeError,
				Msg:  err.Error(),
			}
		}
		return nil
	})
	return tea.Batch(cmds...)
}

const cancelTimerDuration = 2 * time.Second

// cancelTimerCmd 创建一个使取消计时器过期的命令
func cancelTimerCmd() tea.Cmd {
	return tea.Tick(cancelTimerDuration, func(time.Time) tea.Msg {
		return cancelTimerExpiredMsg{}
	})
}

// cancelAgent 处理取消键按下。第一次按下将isCanceling设置为true
// 并启动计时器。第二次按下（在计时器过期之前）实际
// 取消智能体
func (m *UI) cancelAgent() tea.Cmd {
	if !m.hasSession() {
		return nil
	}

	coordinator := m.com.App.AgentCoordinator
	if coordinator == nil {
		return nil
	}

	if m.isCanceling {
		// 第二次按下escape键 - 实际取消智能体
		m.isCanceling = false
		coordinator.Cancel(m.session.ID)
		// 停止旋转的todo指示器
		m.todoIsSpinning = false
		m.renderPills()
		return nil
	}

	// 检查是否有排队的提示 - 如果有，则清除队列
	if coordinator.QueuedPrompts(m.session.ID) > 0 {
		coordinator.ClearQueue(m.session.ID)
		return nil
	}

	// 第一次按下escape键 - 设置取消状态并启动计时器
	m.isCanceling = true
	return cancelTimerCmd()
}

// openDialog 通过ID打开对话框
func (m *UI) openDialog(id string) tea.Cmd {
	var cmds []tea.Cmd
	switch id {
	case dialog.SessionsID:
		if cmd := m.openSessionsDialog(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case dialog.ModelsID:
		if cmd := m.openModelsDialog(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case dialog.CommandsID:
		if cmd := m.openCommandsDialog(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case dialog.ReasoningID:
		if cmd := m.openReasoningDialog(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case dialog.QuitID:
		if cmd := m.openQuitDialog(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	default:
		// 未知对话框
		break
	}
	return tea.Batch(cmds...)
}

// openQuitDialog 打开退出确认对话框
func (m *UI) openQuitDialog() tea.Cmd {
	if m.dialog.ContainsDialog(dialog.QuitID) {
		// 带到前面
		m.dialog.BringToFront(dialog.QuitID)
		return nil
	}

	quitDialog := dialog.NewQuit(m.com)
	m.dialog.OpenDialog(quitDialog)
	return nil
}

// openModelsDialog 打开模型对话框
func (m *UI) openModelsDialog() tea.Cmd {
	if m.dialog.ContainsDialog(dialog.ModelsID) {
		// 带到前面
		m.dialog.BringToFront(dialog.ModelsID)
		return nil
	}

	isOnboarding := m.state == uiOnboarding
	modelsDialog, err := dialog.NewModels(m.com, isOnboarding)
	if err != nil {
		return util.ReportError(err)
	}

	m.dialog.OpenDialog(modelsDialog)

	return nil
}

// openCommandsDialog 打开命令对话框
func (m *UI) openCommandsDialog() tea.Cmd {
	if m.dialog.ContainsDialog(dialog.CommandsID) {
		// 带到前面
		m.dialog.BringToFront(dialog.CommandsID)
		return nil
	}

	sessionID := ""
	if m.session != nil {
		sessionID = m.session.ID
	}

	commands, err := dialog.NewCommands(m.com, sessionID, m.customCommands, m.mcpPrompts)
	if err != nil {
		return util.ReportError(err)
	}

	m.dialog.OpenDialog(commands)

	return nil
}

// openReasoningDialog 打开推理努力对话框
func (m *UI) openReasoningDialog() tea.Cmd {
	if m.dialog.ContainsDialog(dialog.ReasoningID) {
		m.dialog.BringToFront(dialog.ReasoningID)
		return nil
	}

	reasoningDialog, err := dialog.NewReasoning(m.com)
	if err != nil {
		return util.ReportError(err)
	}

	m.dialog.OpenDialog(reasoningDialog)
	return nil
}

// openSessionsDialog 打开会话对话框。如果对话框已经打开，
// 它会将其带到前面。否则，它将列出所有会话并打开
// 对话框
func (m *UI) openSessionsDialog() tea.Cmd {
	if m.dialog.ContainsDialog(dialog.SessionsID) {
		// 带到前面
		m.dialog.BringToFront(dialog.SessionsID)
		return nil
	}

	selectedSessionID := ""
	if m.session != nil {
		selectedSessionID = m.session.ID
	}

	dialog, err := dialog.NewSessions(m.com, selectedSessionID)
	if err != nil {
		return util.ReportError(err)
	}

	m.dialog.OpenDialog(dialog)
	return nil
}

// openFilesDialog 打开文件选择器对话框
func (m *UI) openFilesDialog() tea.Cmd {
	if m.dialog.ContainsDialog(dialog.FilePickerID) {
		// 带到前面
		m.dialog.BringToFront(dialog.FilePickerID)
		return nil
	}

	filePicker, cmd := dialog.NewFilePicker(m.com)
	filePicker.SetImageCapabilities(&m.caps)
	m.dialog.OpenDialog(filePicker)

	return cmd
}

// openPermissionsDialog 为权限请求打开权限对话框
func (m *UI) openPermissionsDialog(perm permission.PermissionRequest) tea.Cmd {
	// 首先关闭任何现有的权限对话框
	m.dialog.CloseDialog(dialog.PermissionsID)

	// 从配置获取差异模式
	var opts []dialog.PermissionsOption
	if diffMode := m.com.Config().Options.TUI.DiffMode; diffMode != "" {
		opts = append(opts, dialog.WithDiffMode(diffMode == "split"))
	}

	permDialog := dialog.NewPermissions(m.com, perm, opts...)
	m.dialog.OpenDialog(permDialog)
	return nil
}

// handlePermissionNotification 当权限状态改变时更新工具项
func (m *UI) handlePermissionNotification(notification permission.PermissionNotification) {
	toolItem := m.chat.MessageItem(notification.ToolCallID)
	if toolItem == nil {
		return
	}

	if permItem, ok := toolItem.(chat.ToolMessageItem); ok {
		if notification.Granted {
			permItem.SetStatus(chat.ToolStatusRunning)
		} else {
			permItem.SetStatus(chat.ToolStatusAwaitingPermission)
		}
	}
}

// newSession 清除当前会话状态并准备新会话
// 实际的会话创建发生在用户发送第一条消息时
// 返回重新加载提示历史的命令
func (m *UI) newSession() tea.Cmd {
	if !m.hasSession() {
		return nil
	}

	m.session = nil
	m.sessionFiles = nil
	m.sessionFileReads = nil
	m.setState(uiLanding, uiFocusEditor)
	m.textarea.Focus()
	m.chat.Blur()
	m.chat.ClearMessages()
	m.pillsExpanded = false
	m.promptQueue = 0
	m.pillsView = ""
	m.historyReset()
	return tea.Batch(
		func() tea.Msg {
			m.com.App.LSPManager.StopAll(context.Background())
			return nil
		},
		m.loadPromptHistory(),
	)
}

// handlePasteMsg 处理粘贴消息
func (m *UI) handlePasteMsg(msg tea.PasteMsg) tea.Cmd {
	if m.dialog.HasDialogs() {
		return m.handleDialogMsg(msg)
	}

	if m.focus != uiFocusEditor {
		return nil
	}

	if strings.Count(msg.Content, "\n") > pasteLinesThreshold {
		return func() tea.Msg {
			content := []byte(msg.Content)
			if int64(len(content)) > common.MaxAttachmentSize {
				return util.ReportWarn("粘贴内容过大（>5MB）")
			}
			name := fmt.Sprintf("paste_%d.txt", m.pasteIdx())
			mimeBufferSize := min(512, len(content))
			mimeType := http.DetectContentType(content[:mimeBufferSize])
			return message.Attachment{
				FileName: name,
				FilePath: name,
				MimeType: mimeType,
				Content:  content,
			}
		}
	}

	// 尝试将粘贴的内容解析为文件路径。如果可以解析，
	// 所有文件都存在且有效，则添加为附件
	// 否则，作为文本粘贴
	paths := fsext.ParsePastedFiles(msg.Content)
	allExistsAndValid := func() bool {
		if len(paths) == 0 {
			return false
		}
		for _, path := range paths {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return false
			}

			lowerPath := strings.ToLower(path)
			isValid := false
			for _, ext := range common.AllowedImageTypes {
				if strings.HasSuffix(lowerPath, ext) {
					isValid = true
					break
				}
			}
			if !isValid {
				return false
			}
		}
		return true
	}
	if !allExistsAndValid() {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return cmd
	}

	var cmds []tea.Cmd
	for _, path := range paths {
		cmds = append(cmds, m.handleFilePathPaste(path))
	}
	return tea.Batch(cmds...)
}

// handleFilePathPaste 处理粘贴的文件路径
func (m *UI) handleFilePathPaste(path string) tea.Cmd {
	return func() tea.Msg {
		fileInfo, err := os.Stat(path)
		if err != nil {
			return util.ReportError(err)
		}
		if fileInfo.IsDir() {
			return util.ReportWarn("不能附加目录")
		}
		if fileInfo.Size() > common.MaxAttachmentSize {
			return util.ReportWarn("文件过大（>5MB）")
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return util.ReportError(err)
		}

		mimeBufferSize := min(512, len(content))
		mimeType := http.DetectContentType(content[:mimeBufferSize])
		fileName := filepath.Base(path)
		return message.Attachment{
			FilePath: path,
			FileName: fileName,
			MimeType: mimeType,
			Content:  content,
		}
	}
}

// pasteImageFromClipboard 从系统剪贴板读取图像数据并
// 创建附件。如果未找到图像数据，则回退到
// 将剪贴板文本解释为文件路径
func (m *UI) pasteImageFromClipboard() tea.Msg {
	imageData, err := readClipboard(clipboardFormatImage)
	if int64(len(imageData)) > common.MaxAttachmentSize {
		return util.InfoMsg{
			Type: util.InfoTypeError,
			Msg:  "文件过大，最大5MB",
		}
	}
	name := fmt.Sprintf("paste_%d.png", m.pasteIdx())
	if err == nil {
		return message.Attachment{
			FilePath: name,
			FileName: name,
			MimeType: mimeOf(imageData),
			Content:  imageData,
		}
	}

	textData, textErr := readClipboard(clipboardFormatText)
	if textErr != nil || len(textData) == 0 {
		return util.NewInfoMsg("剪贴板为空或不包含图像")
	}

	path := strings.TrimSpace(string(textData))
	path = strings.ReplaceAll(path, "\\ ", " ")
	if _, statErr := os.Stat(path); statErr != nil {
		return util.NewInfoMsg("剪贴板不包含图像或有效的文件路径")
	}

	lowerPath := strings.ToLower(path)
	isAllowed := false
	for _, ext := range common.AllowedImageTypes {
		if strings.HasSuffix(lowerPath, ext) {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		return util.NewInfoMsg("文件类型不是支持的图像格式")
	}

	fileInfo, statErr := os.Stat(path)
	if statErr != nil {
		return util.InfoMsg{
			Type: util.InfoTypeError,
			Msg:  fmt.Sprintf("无法读取文件: %v", statErr),
		}
	}
	if fileInfo.Size() > common.MaxAttachmentSize {
		return util.InfoMsg{
			Type: util.InfoTypeError,
			Msg:  "文件过大，最大5MB",
		}
	}

	content, readErr := os.ReadFile(path)
	if readErr != nil {
		return util.InfoMsg{
			Type: util.InfoTypeError,
			Msg:  fmt.Sprintf("无法读取文件: %v", readErr),
		}
	}

	return message.Attachment{
		FilePath: path,
		FileName: filepath.Base(path),
		MimeType: mimeOf(content),
		Content:  content,
	}
}

var pasteRE = regexp.MustCompile(`paste_(\d+).txt`)

func (m *UI) pasteIdx() int {
	result := 0
	for _, at := range m.attachments.List() {
		found := pasteRE.FindStringSubmatch(at.FileName)
		if len(found) == 0 {
			continue
		}
		idx, err := strconv.Atoi(found[1])
		if err == nil {
			result = max(result, idx)
		}
	}
	return result + 1
}

// drawSessionDetails 在紧凑模式下绘制会话详情
func (m *UI) drawSessionDetails(scr uv.Screen, area uv.Rectangle) {
	if m.session == nil {
		return
	}

	s := m.com.Styles

	width := area.Dx() - s.CompactDetails.View.GetHorizontalFrameSize()
	height := area.Dy() - s.CompactDetails.View.GetVerticalFrameSize()

	title := s.CompactDetails.Title.Width(width).MaxHeight(2).Render(m.session.Title)
	blocks := []string{
		title,
		"",
		m.modelInfo(width),
		"",
	}

	detailsHeader := lipgloss.JoinVertical(
		lipgloss.Left,
		blocks...,
	)

	version := s.CompactDetails.Version.Foreground(s.Border).Width(width).AlignHorizontal(lipgloss.Right).Render(version.Version)

	remainingHeight := height - lipgloss.Height(detailsHeader) - lipgloss.Height(version)

	const maxSectionWidth = 50
	sectionWidth := min(maxSectionWidth, width/3-2) // account for 2 spaces
	maxItemsPerSection := remainingHeight - 3       // 为节标题和间距预留空间

	lspSection := m.lspInfo(sectionWidth, maxItemsPerSection, false)
	mcpSection := m.mcpInfo(sectionWidth, maxItemsPerSection, false)
	filesSection := m.filesInfo(m.com.Config().WorkingDir(), sectionWidth, maxItemsPerSection, false)
	sections := lipgloss.JoinHorizontal(lipgloss.Top, filesSection, " ", lspSection, " ", mcpSection)
	uv.NewStyledString(
		s.CompactDetails.View.
			Width(area.Dx()).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					detailsHeader,
					sections,
					version,
				),
			),
	).Draw(scr, area)
}

func (m *UI) runMCPPrompt(clientID, promptID string, arguments map[string]string) tea.Cmd {
	load := func() tea.Msg {
		prompt, err := commands.GetMCPPrompt(m.com.Config(), clientID, promptID, arguments)
		if err != nil {
			// TODO: 改进这个
			return util.ReportError(err)()
		}

		if prompt == "" {
			return nil
		}
		return sendMessageMsg{
			Content: prompt,
		}
	}

	var cmds []tea.Cmd
	if cmd := m.dialog.StartLoading(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	cmds = append(cmds, load, func() tea.Msg {
		return closeDialogMsg{}
	})

	return tea.Sequence(cmds...)
}

func (m *UI) handleStateChanged() tea.Cmd {
	return func() tea.Msg {
		m.com.App.UpdateAgentModel(context.Background())
		return mcpStateChangedMsg{
			states: mcp.GetStates(),
		}
	}
}

func handleMCPPromptsEvent(name string) tea.Cmd {
	return func() tea.Msg {
		mcp.RefreshPrompts(context.Background(), name)
		return nil
	}
}

func handleMCPToolsEvent(cfg *config.Config, name string) tea.Cmd {
	return func() tea.Msg {
		mcp.RefreshTools(
			context.Background(),
			cfg,
			name,
		)
		return nil
	}
}

func handleMCPResourcesEvent(name string) tea.Cmd {
	return func() tea.Msg {
		mcp.RefreshResources(context.Background(), name)
		return nil
	}
}

func (m *UI) copyChatHighlight() tea.Cmd {
	text := m.chat.HighlightContent()
	return common.CopyToClipboardWithCallback(
		text,
		"选中的文本已复制到剪贴板",
		func() tea.Msg {
			m.chat.ClearMouse()
			return nil
		},
	)
}

// renderLogo 使用给定的样式和尺寸渲染Crush logo
func renderLogo(t *styles.Styles, compact bool, width int) string {
	return logo.Render(t, version.Version, compact, logo.Opts{
		FieldColor:   t.LogoFieldColor,
		TitleColorA:  t.LogoTitleColorA,
		TitleColorB:  t.LogoTitleColorB,
		CharmColor:   t.LogoCharmColor,
		VersionColor: t.LogoVersionColor,
		Width:        width,
	})
}
