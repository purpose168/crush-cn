package dialog

import (
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/agent/tools"
	"github.com/charmbracelet/crush/internal/fsext"
	"github.com/charmbracelet/crush/internal/permission"
	"github.com/charmbracelet/crush/internal/stringext"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/styles"
	uv "github.com/charmbracelet/ultraviolet"
)

// PermissionsID 是权限对话框的标识符。
const PermissionsID = "permissions"

// PermissionAction 表示用户对权限请求的响应。
type PermissionAction string

const (
	PermissionAllow           PermissionAction = "allow"
	PermissionAllowForSession PermissionAction = "allow_session"
	PermissionDeny            PermissionAction = "deny"
)

// Permissions 对话框大小常量。
const (
	// diffMaxWidth 是差异视图的最大宽度。
	diffMaxWidth = 180
	// diffSizeRatio 是差异视图相对于窗口的大小比例。
	diffSizeRatio = 0.8
	// simpleMaxWidth 是简单内容对话框的最大宽度。
	simpleMaxWidth = 100
	// simpleSizeRatio 是简单内容对话框的大小比例。
	simpleSizeRatio = 0.6
	// simpleHeightRatio 是简单内容对话框的高度比例。
	simpleHeightRatio = 0.5
	// splitModeMinWidth 是启用分割差异模式的最小宽度。
	splitModeMinWidth = 140
	// layoutSpacingLines 是用于布局间距的空行数。
	layoutSpacingLines = 4
	// minWindowWidth 是强制全屏之前的最小窗口宽度。
	minWindowWidth = 77
	// minWindowHeight 是强制全屏之前的最小窗口高度。
	minWindowHeight = 20
)

// Permissions 表示一个用于权限请求的对话框。
type Permissions struct {
	com          *common.Common
	windowWidth  int // 终端窗口尺寸。
	windowHeight int
	fullscreen   bool // 当对话框全屏时为 true

	permission     permission.PermissionRequest
	selectedOption int // 0: 允许, 1: 允许本次会话, 2: 拒绝

	viewport      viewport.Model
	viewportDirty bool // 当视口内容需要重新渲染时为 true
	viewportWidth int

	// 差异视图状态。
	diffSplitMode        *bool // nil 表示根据宽度使用默认值
	defaultDiffSplitMode bool  // 基于宽度的默认分割模式
	diffXOffset          int   // 差异视图的水平滚动偏移量
	unifiedDiffContent   string
	splitDiffContent     string

	help   help.Model
	keyMap permissionsKeyMap
}

type permissionsKeyMap struct {
	Left             key.Binding
	Right            key.Binding
	Tab              key.Binding
	Select           key.Binding
	Allow            key.Binding
	AllowSession     key.Binding
	Deny             key.Binding
	Close            key.Binding
	ToggleDiffMode   key.Binding
	ToggleFullscreen key.Binding
	ScrollUp         key.Binding
	ScrollDown       key.Binding
	ScrollLeft       key.Binding
	ScrollRight      key.Binding
	Choose           key.Binding
	Scroll           key.Binding
}

func defaultPermissionsKeyMap() permissionsKeyMap {
	return permissionsKeyMap{
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←", "上一个"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→", "下一个"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "下一个选项"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter", "ctrl+y"),
			key.WithHelp("enter", "确认"),
		),
		Allow: key.NewBinding(
			key.WithKeys("a", "A", "ctrl+a"),
			key.WithHelp("a", "允许"),
		),
		AllowSession: key.NewBinding(
			key.WithKeys("s", "S", "ctrl+s"),
			key.WithHelp("s", "允许本次会话"),
		),
		Deny: key.NewBinding(
			key.WithKeys("d", "D"),
			key.WithHelp("d", "拒绝"),
		),
		Close: CloseKey,
		ToggleDiffMode: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "切换差异视图"),
		),
		ToggleFullscreen: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "切换全屏"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys("shift+up", "K"),
			key.WithHelp("shift+↑", "向上滚动"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("shift+down", "J"),
			key.WithHelp("shift+↓", "向下滚动"),
		),
		ScrollLeft: key.NewBinding(
			key.WithKeys("shift+left", "H"),
			key.WithHelp("shift+←", "向左滚动"),
		),
		ScrollRight: key.NewBinding(
			key.WithKeys("shift+right", "L"),
			key.WithHelp("shift+→", "向右滚动"),
		),
		Choose: key.NewBinding(
			key.WithKeys("left", "right"),
			key.WithHelp("←/→", "选择"),
		),
		Scroll: key.NewBinding(
			key.WithKeys("shift+left", "shift+down", "shift+up", "shift+right"),
			key.WithHelp("shift+←↓↑→", "滚动"),
		),
	}
}

var _ Dialog = (*Permissions)(nil)

// PermissionsOption 配置权限对话框。
type PermissionsOption func(*Permissions)

// WithDiffMode 设置初始差异模式（分割或统一）。
func WithDiffMode(split bool) PermissionsOption {
	return func(p *Permissions) {
		p.diffSplitMode = &split
	}
}

// NewPermissions 创建一个新的权限对话框。
func NewPermissions(com *common.Common, perm permission.PermissionRequest, opts ...PermissionsOption) *Permissions {
	h := help.New()
	h.Styles = com.Styles.DialogHelpStyles()

	km := defaultPermissionsKeyMap()

	// 使用匹配的键绑定配置视口。
	vp := viewport.New()
	vp.KeyMap = viewport.KeyMap{
		Up:    km.ScrollUp,
		Down:  km.ScrollDown,
		Left:  km.ScrollLeft,
		Right: km.ScrollRight,
		// 禁用其他视口键以避免与对话框快捷键冲突。
		PageUp:       key.NewBinding(key.WithDisabled()),
		PageDown:     key.NewBinding(key.WithDisabled()),
		HalfPageUp:   key.NewBinding(key.WithDisabled()),
		HalfPageDown: key.NewBinding(key.WithDisabled()),
	}

	p := &Permissions{
		com:            com,
		permission:     perm,
		selectedOption: 0,
		viewport:       vp,
		help:           h,
		keyMap:         km,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// 计算可用内容宽度（对话框边框 + 水平内边距）。
func (p *Permissions) calculateContentWidth(width int) int {
	t := p.com.Styles
	const dialogHorizontalPadding = 2
	return width - t.Dialog.View.GetHorizontalFrameSize() - dialogHorizontalPadding
}

// ID 实现 [Dialog] 接口。
func (*Permissions) ID() string {
	return PermissionsID
}

// HandleMsg 实现 [Dialog] 接口。
func (p *Permissions) HandleMsg(msg tea.Msg) Action {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, p.keyMap.Close):
			// Escape 拒绝权限请求。
			return p.respond(PermissionDeny)
		case key.Matches(msg, p.keyMap.Right), key.Matches(msg, p.keyMap.Tab):
			p.selectedOption = (p.selectedOption + 1) % 3
		case key.Matches(msg, p.keyMap.Left):
			// 加 2 而不是减 1 以避免负模运算。
			p.selectedOption = (p.selectedOption + 2) % 3
		case key.Matches(msg, p.keyMap.Select):
			return p.selectCurrentOption()
		case key.Matches(msg, p.keyMap.Allow):
			return p.respond(PermissionAllow)
		case key.Matches(msg, p.keyMap.AllowSession):
			return p.respond(PermissionAllowForSession)
		case key.Matches(msg, p.keyMap.Deny):
			return p.respond(PermissionDeny)
		case key.Matches(msg, p.keyMap.ToggleDiffMode):
			if p.hasDiffView() {
				newMode := !p.isSplitMode()
				p.diffSplitMode = &newMode
				p.viewportDirty = true
			}
		case key.Matches(msg, p.keyMap.ToggleFullscreen):
			if p.hasDiffView() {
				p.fullscreen = !p.fullscreen
			}
		case key.Matches(msg, p.keyMap.ScrollDown):
			p.viewport, _ = p.viewport.Update(msg)
		case key.Matches(msg, p.keyMap.ScrollUp):
			p.viewport, _ = p.viewport.Update(msg)
		case key.Matches(msg, p.keyMap.ScrollLeft):
			if p.hasDiffView() {
				p.scrollLeft()
			} else {
				p.viewport, _ = p.viewport.Update(msg)
			}
		case key.Matches(msg, p.keyMap.ScrollRight):
			if p.hasDiffView() {
				p.scrollRight()
			} else {
				p.viewport, _ = p.viewport.Update(msg)
			}
		}
	case tea.MouseWheelMsg:
		if p.hasDiffView() {
			switch msg.Button {
			case tea.MouseWheelLeft:
				p.scrollLeft()
			case tea.MouseWheelRight:
				p.scrollRight()
			default:
				p.viewport, _ = p.viewport.Update(msg)
			}
		} else {
			p.viewport, _ = p.viewport.Update(msg)
		}
	default:
		// 将未处理的键传递给视口以进行非差异内容滚动。
		if !p.hasDiffView() {
			p.viewport, _ = p.viewport.Update(msg)
			p.viewportDirty = true
		}
	}

	return nil
}

func (p *Permissions) selectCurrentOption() tea.Msg {
	switch p.selectedOption {
	case 0:
		return p.respond(PermissionAllow)
	case 1:
		return p.respond(PermissionAllowForSession)
	default:
		return p.respond(PermissionDeny)
	}
}

func (p *Permissions) respond(action PermissionAction) tea.Msg {
	return ActionPermissionResponse{
		Permission: p.permission,
		Action:     action,
	}
}

func (p *Permissions) hasDiffView() bool {
	switch p.permission.ToolName {
	case tools.EditToolName, tools.WriteToolName, tools.MultiEditToolName:
		return true
	}
	return false
}

func (p *Permissions) isSplitMode() bool {
	if p.diffSplitMode != nil {
		return *p.diffSplitMode
	}
	return p.defaultDiffSplitMode
}

const horizontalScrollStep = 5

func (p *Permissions) scrollLeft() {
	p.diffXOffset = max(0, p.diffXOffset-horizontalScrollStep)
	p.viewportDirty = true
}

func (p *Permissions) scrollRight() {
	p.diffXOffset += horizontalScrollStep
	p.viewportDirty = true
}

// Draw 实现 [Dialog] 接口。
func (p *Permissions) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	t := p.com.Styles
	// 当窗口太小时强制全屏。
	forceFullscreen := area.Dx() <= minWindowWidth || area.Dy() <= minWindowHeight

	// 根据全屏状态和内容类型计算对话框尺寸。
	var width, maxHeight int
	if forceFullscreen || (p.fullscreen && p.hasDiffView()) {
		// 全屏时使用几乎整个窗口。
		width = area.Dx()
		maxHeight = area.Dy()
	} else if p.hasDiffView() {
		// 并排差异视图较宽，为了可读性进行限制。
		width = min(int(float64(area.Dx())*diffSizeRatio), diffMaxWidth)
		maxHeight = int(float64(area.Dy()) * diffSizeRatio)
	} else {
		// 简单内容（如命令/URL）较窄。
		width = min(int(float64(area.Dx())*simpleSizeRatio), simpleMaxWidth)
		maxHeight = int(float64(area.Dy()) * simpleHeightRatio)
	}

	dialogStyle := t.Dialog.View.Width(width).Padding(0, 1)

	contentWidth := p.calculateContentWidth(width)
	header := p.renderHeader(contentWidth)
	buttons := p.renderButtons(contentWidth)
	helpView := p.help.View(p)

	// 计算内容的可用高度。
	headerHeight := lipgloss.Height(header)
	buttonsHeight := lipgloss.Height(buttons)
	helpHeight := lipgloss.Height(helpView)
	frameHeight := dialogStyle.GetVerticalFrameSize() + layoutSpacingLines

	p.defaultDiffSplitMode = width >= splitModeMinWidth

	// 预渲染内容以测量其实际高度。
	renderedContent := p.renderContent(contentWidth)
	contentHeight := lipgloss.Height(renderedContent)

	// 对于非差异视图，如果内容小于最大值，则缩小对话框以适应内容。
	var availableHeight int
	if !p.hasDiffView() && !forceFullscreen {
		fixedHeight := headerHeight + buttonsHeight + helpHeight + frameHeight
		neededHeight := fixedHeight + contentHeight
		if neededHeight < maxHeight {
			availableHeight = contentHeight
		} else {
			availableHeight = maxHeight - fixedHeight
		}
		availableHeight = max(availableHeight, 3)
	} else {
		availableHeight = maxHeight - headerHeight - buttonsHeight - helpHeight - frameHeight
	}

	// 确定是否需要滚动条。
	needsScrollbar := p.hasDiffView() || contentHeight > availableHeight
	viewportWidth := contentWidth
	if needsScrollbar {
		viewportWidth = contentWidth - 1 // 为滚动条预留空间。
	}

	if p.viewport.Width() != viewportWidth {
		// 如果宽度已更改，则将内容标记为脏。
		p.viewportDirty = true
		renderedContent = p.renderContent(viewportWidth)
	}

	var content string
	var scrollbar string
	p.viewport.SetWidth(viewportWidth)
	p.viewport.SetHeight(availableHeight)
	if p.viewportDirty {
		p.viewport.SetContent(renderedContent)
		p.viewportWidth = p.viewport.Width()
		p.viewportDirty = false
	}
	content = p.viewport.View()
	if needsScrollbar {
		scrollbar = common.Scrollbar(t, availableHeight, p.viewport.TotalLineCount(), availableHeight, p.viewport.YOffset())
	}

	// 如果存在滚动条，则将内容与滚动条连接。
	if scrollbar != "" {
		content = lipgloss.JoinHorizontal(lipgloss.Top, content, scrollbar)
	}

	parts := []string{header}
	if content != "" {
		parts = append(parts, "", content)
	}
	parts = append(parts, "", buttons, "", helpView)

	innerContent := lipgloss.JoinVertical(lipgloss.Left, parts...)
	DrawCenterCursor(scr, area, dialogStyle.Render(innerContent), nil)
	return nil
}

func (p *Permissions) renderHeader(contentWidth int) string {
	t := p.com.Styles

	title := common.DialogTitle(t, "需要权限", contentWidth-t.Dialog.Title.GetHorizontalFrameSize(), t.Primary, t.Secondary)
	title = t.Dialog.Title.Render(title)

	// 工具信息。
	toolLine := p.renderToolName(contentWidth)
	pathLine := p.renderKeyValue("路径", fsext.PrettyPath(p.permission.Path), contentWidth)

	lines := []string{title, "", toolLine, pathLine}

	// 添加工具特定的头部信息。
	switch p.permission.ToolName {
	case tools.BashToolName:
		if params, ok := p.permission.Params.(tools.BashPermissionsParams); ok {
			lines = append(lines, p.renderKeyValue("描述", params.Description, contentWidth))
		}
	case tools.DownloadToolName:
		if params, ok := p.permission.Params.(tools.DownloadPermissionsParams); ok {
			lines = append(lines, p.renderKeyValue("URL", params.URL, contentWidth))
			lines = append(lines, p.renderKeyValue("文件", fsext.PrettyPath(params.FilePath), contentWidth))
		}
	case tools.EditToolName, tools.WriteToolName, tools.MultiEditToolName, tools.ViewToolName:
		var filePath string
		switch params := p.permission.Params.(type) {
		case tools.EditPermissionsParams:
			filePath = params.FilePath
		case tools.WritePermissionsParams:
			filePath = params.FilePath
		case tools.MultiEditPermissionsParams:
			filePath = params.FilePath
		case tools.ViewPermissionsParams:
			filePath = params.FilePath
		}
		if filePath != "" {
			lines = append(lines, p.renderKeyValue("文件", fsext.PrettyPath(filePath), contentWidth))
		}
	case tools.LSToolName:
		if params, ok := p.permission.Params.(tools.LSPermissionsParams); ok {
			lines = append(lines, p.renderKeyValue("目录", fsext.PrettyPath(params.Path), contentWidth))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (p *Permissions) renderKeyValue(key, value string, width int) string {
	t := p.com.Styles
	keyStyle := t.Muted
	valueStyle := t.Base

	keyStr := keyStyle.Render(key)
	valueStr := valueStyle.Width(width - lipgloss.Width(keyStr) - 1).Render(" " + value)

	return lipgloss.JoinHorizontal(lipgloss.Left, keyStr, valueStr)
}

func (p *Permissions) renderToolName(width int) string {
	toolName := p.permission.ToolName

	// 检查这是否是 MCP 工具（格式：mcp_<mcpname>_<toolname>）。
	if strings.HasPrefix(toolName, "mcp_") {
		parts := strings.SplitN(toolName, "_", 3)
		if len(parts) == 3 {
			mcpName := prettyName(parts[1])
			toolPart := prettyName(parts[2])
			toolName = fmt.Sprintf("%s %s %s", mcpName, styles.ArrowRightIcon, toolPart)
		}
	}

	return p.renderKeyValue("工具", toolName, width)
}

// prettyName 将 snake_case 或 kebab-case 转换为标题大小写。
func prettyName(name string) string {
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	return stringext.Capitalize(name)
}

func (p *Permissions) renderContent(width int) string {
	switch p.permission.ToolName {
	case tools.BashToolName:
		return p.renderBashContent(width)
	case tools.EditToolName:
		return p.renderEditContent(width)
	case tools.WriteToolName:
		return p.renderWriteContent(width)
	case tools.MultiEditToolName:
		return p.renderMultiEditContent(width)
	case tools.DownloadToolName:
		return p.renderDownloadContent(width)
	case tools.FetchToolName:
		return p.renderFetchContent(width)
	case tools.AgenticFetchToolName:
		return p.renderAgenticFetchContent(width)
	case tools.ViewToolName:
		return p.renderViewContent(width)
	case tools.LSToolName:
		return p.renderLSContent(width)
	default:
		return p.renderDefaultContent(width)
	}
}

func (p *Permissions) renderBashContent(width int) string {
	params, ok := p.permission.Params.(tools.BashPermissionsParams)
	if !ok {
		return ""
	}

	return p.renderContentPanel(params.Command, width)
}

func (p *Permissions) renderEditContent(contentWidth int) string {
	params, ok := p.permission.Params.(tools.EditPermissionsParams)
	if !ok {
		return ""
	}
	return p.renderDiff(params.FilePath, params.OldContent, params.NewContent, contentWidth)
}

func (p *Permissions) renderWriteContent(contentWidth int) string {
	params, ok := p.permission.Params.(tools.WritePermissionsParams)
	if !ok {
		return ""
	}
	return p.renderDiff(params.FilePath, params.OldContent, params.NewContent, contentWidth)
}

func (p *Permissions) renderMultiEditContent(contentWidth int) string {
	params, ok := p.permission.Params.(tools.MultiEditPermissionsParams)
	if !ok {
		return ""
	}
	return p.renderDiff(params.FilePath, params.OldContent, params.NewContent, contentWidth)
}

func (p *Permissions) renderDiff(filePath, oldContent, newContent string, contentWidth int) string {
	if !p.viewportDirty {
		if p.isSplitMode() {
			return p.splitDiffContent
		}
		return p.unifiedDiffContent
	}

	isSplitMode := p.isSplitMode()
	formatter := common.DiffFormatter(p.com.Styles).
		Before(fsext.PrettyPath(filePath), oldContent).
		After(fsext.PrettyPath(filePath), newContent).
		XOffset(p.diffXOffset).
		Width(contentWidth)

	var result string
	if isSplitMode {
		formatter = formatter.Split()
		p.splitDiffContent = formatter.String()
		result = p.splitDiffContent
	} else {
		formatter = formatter.Unified()
		p.unifiedDiffContent = formatter.String()
		result = p.unifiedDiffContent
	}

	return result
}

func (p *Permissions) renderDownloadContent(width int) string {
	params, ok := p.permission.Params.(tools.DownloadPermissionsParams)
	if !ok {
		return ""
	}

	content := fmt.Sprintf("URL: %s\n文件: %s", params.URL, fsext.PrettyPath(params.FilePath))
	if params.Timeout > 0 {
		content += fmt.Sprintf("\n超时: %ds", params.Timeout)
	}

	return p.renderContentPanel(content, width)
}

func (p *Permissions) renderFetchContent(width int) string {
	params, ok := p.permission.Params.(tools.FetchPermissionsParams)
	if !ok {
		return ""
	}

	return p.renderContentPanel(params.URL, width)
}

func (p *Permissions) renderAgenticFetchContent(width int) string {
	params, ok := p.permission.Params.(tools.AgenticFetchPermissionsParams)
	if !ok {
		return ""
	}

	var content string
	if params.URL != "" {
		content = fmt.Sprintf("URL: %s\n\n提示: %s", params.URL, params.Prompt)
	} else {
		content = fmt.Sprintf("提示: %s", params.Prompt)
	}

	return p.renderContentPanel(content, width)
}

func (p *Permissions) renderViewContent(width int) string {
	params, ok := p.permission.Params.(tools.ViewPermissionsParams)
	if !ok {
		return ""
	}

	content := fmt.Sprintf("文件: %s", fsext.PrettyPath(params.FilePath))
	if params.Offset > 0 {
		content += fmt.Sprintf("\n起始行: %d", params.Offset+1)
	}
	if params.Limit > 0 && params.Limit != 2000 {
		content += fmt.Sprintf("\n读取行数: %d", params.Limit)
	}

	return p.renderContentPanel(content, width)
}

func (p *Permissions) renderLSContent(width int) string {
	params, ok := p.permission.Params.(tools.LSPermissionsParams)
	if !ok {
		return ""
	}

	content := fmt.Sprintf("目录: %s", fsext.PrettyPath(params.Path))
	if len(params.Ignore) > 0 {
		content += fmt.Sprintf("\n忽略模式: %s", strings.Join(params.Ignore, ", "))
	}

	return p.renderContentPanel(content, width)
}

func (p *Permissions) renderDefaultContent(width int) string {
	t := p.com.Styles
	var content string
	// 不要为 mcp 工具添加描述
	if !strings.HasPrefix(p.permission.ToolName, "mcp_") {
		content = p.permission.Description
	}

	// 如果可用，则漂亮地打印 JSON 参数。
	if p.permission.Params != nil {
		var paramStr string
		if str, ok := p.permission.Params.(string); ok {
			paramStr = str
		} else {
			paramStr = fmt.Sprintf("%v", p.permission.Params)
		}

		var parsed any
		if err := json.Unmarshal([]byte(paramStr), &parsed); err == nil {
			if b, err := json.MarshalIndent(parsed, "", "  "); err == nil {
				jsonContent := string(b)
				highlighted, err := common.SyntaxHighlight(t, jsonContent, "params.json", t.BgSubtle)
				if err == nil {
					jsonContent = highlighted
				}
				if content != "" {
					content += "\n\n"
				}
				content += jsonContent
			}
		} else if paramStr != "" {
			if content != "" {
				content += "\n\n"
			}
			content += paramStr
		}
	}

	if content == "" {
		return ""
	}

	return p.renderContentPanel(strings.TrimSpace(content), width)
}

// renderContentPanel 在具有全宽度的面板中渲染内容。
func (p *Permissions) renderContentPanel(content string, width int) string {
	panelStyle := p.com.Styles.Dialog.ContentPanel
	return panelStyle.Width(width).Render(content)
}

func (p *Permissions) renderButtons(contentWidth int) string {
	buttons := []common.ButtonOpts{
		{Text: "允许", UnderlineIndex: 0, Selected: p.selectedOption == 0},
		{Text: "允许本次会话", UnderlineIndex: 10, Selected: p.selectedOption == 1},
		{Text: "拒绝", UnderlineIndex: 0, Selected: p.selectedOption == 2},
	}

	content := common.ButtonGroup(p.com.Styles, buttons, "  ")

	// 如果按钮太宽，则垂直堆叠它们。
	if lipgloss.Width(content) > contentWidth {
		content = common.ButtonGroup(p.com.Styles, buttons, "\n")
		return lipgloss.NewStyle().
			Width(contentWidth).
			Align(lipgloss.Center).
			Render(content)
	}

	return lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Right).
		Render(content)
}

func (p *Permissions) canScroll() bool {
	if p.hasDiffView() {
		// 差异视图始终可以滚动。
		return true
	}
	// 对于非差异内容，检查视口是否具有可滚动内容。
	return !p.viewport.AtTop() || !p.viewport.AtBottom()
}

// ShortHelp 实现 [help.KeyMap] 接口。
func (p *Permissions) ShortHelp() []key.Binding {
	bindings := []key.Binding{
		p.keyMap.Choose,
		p.keyMap.Select,
		p.keyMap.Close,
	}

	if p.canScroll() {
		bindings = append(bindings, p.keyMap.Scroll)
	}

	if p.hasDiffView() {
		bindings = append(bindings,
			p.keyMap.ToggleDiffMode,
			p.keyMap.ToggleFullscreen,
		)
	}

	return bindings
}

// FullHelp 实现 [help.KeyMap] 接口。
func (p *Permissions) FullHelp() [][]key.Binding {
	return [][]key.Binding{p.ShortHelp()}
}
