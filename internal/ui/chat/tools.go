package chat

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/tree"
	"github.com/charmbracelet/x/ansi"
	"github.com/purpose168/crush-cn/internal/agent"
	"github.com/purpose168/crush-cn/internal/agent/tools"
	"github.com/purpose168/crush-cn/internal/diff"
	"github.com/purpose168/crush-cn/internal/fsext"
	"github.com/purpose168/crush-cn/internal/message"
	"github.com/purpose168/crush-cn/internal/stringext"
	"github.com/purpose168/crush-cn/internal/ui/anim"
	"github.com/purpose168/crush-cn/internal/ui/common"
	"github.com/purpose168/crush-cn/internal/ui/styles"
)

// responseContextHeight 限制工具输出中显示的行数
const responseContextHeight = 10

// toolBodyLeftPaddingTotal 表示应用于每个工具主体的左侧填充值
const toolBodyLeftPaddingTotal = 2

// ToolStatus 表示工具调用的当前状态
type ToolStatus int

const (
	ToolStatusAwaitingPermission ToolStatus = iota // 等待权限许可
	ToolStatusRunning                              // 正在运行
	ToolStatusSuccess                              // 执行成功
	ToolStatusError                                // 执行错误
	ToolStatusCanceled                             // 已取消
)

// ToolMessageItem 表示聊天界面中的工具调用消息
type ToolMessageItem interface {
	MessageItem

	ToolCall() message.ToolCall
	SetToolCall(tc message.ToolCall)
	SetResult(res *message.ToolResult)
	MessageID() string
	SetMessageID(id string)
	SetStatus(status ToolStatus)
	Status() ToolStatus
}

// Compactable 是可以在紧凑模式下渲染的工具项接口
// 当启用紧凑模式时，工具渲染为紧凑的单行标题
type Compactable interface {
	SetCompact(compact bool)
}

// SpinningState 包含传递给 SpinningFunc 用于自定义旋转逻辑的状态
type SpinningState struct {
	ToolCall message.ToolCall
	Result   *message.ToolResult
	Status   ToolStatus
}

// IsCanceled 返回工具状态是否为已取消
func (s *SpinningState) IsCanceled() bool {
	return s.Status == ToolStatusCanceled
}

// HasResult 返回结果是否不为 nil
func (s *SpinningState) HasResult() bool {
	return s.Result != nil
}

// SpinningFunc 是自定义旋转逻辑的函数类型
// 如果工具应该显示旋转动画，则返回 true
type SpinningFunc func(state SpinningState) bool

// DefaultToolRenderContext 实现默认的 [ToolRenderer] 接口
type DefaultToolRenderContext struct{}

// RenderTool 实现 [ToolRenderer] 接口
func (d *DefaultToolRenderContext) RenderTool(sty *styles.Styles, width int, opts *ToolRenderOpts) string {
	return "待实现：为以下工具渲染器实现渲染功能: " + opts.ToolCall.Name
}

// ToolRenderOpts 包含渲染工具调用所需的数据
type ToolRenderOpts struct {
	ToolCall        message.ToolCall
	Result          *message.ToolResult
	Anim            *anim.Anim
	ExpandedContent bool
	Compact         bool
	IsSpinning      bool
	Status          ToolStatus
}

// IsPending 返回工具调用是否仍在等待中（未完成且未取消）
func (o *ToolRenderOpts) IsPending() bool {
	return !o.ToolCall.Finished && !o.IsCanceled()
}

// IsCanceled 返回工具状态是否为已取消
func (o *ToolRenderOpts) IsCanceled() bool {
	return o.Status == ToolStatusCanceled
}

// HasResult 返回结果是否不为 nil
func (o *ToolRenderOpts) HasResult() bool {
	return o.Result != nil
}

// HasEmptyResult 返回结果是否为 nil 或内容为空
func (o *ToolRenderOpts) HasEmptyResult() bool {
	return o.Result == nil || o.Result.Content == ""
}

// ToolRenderer 表示渲染工具调用的接口
type ToolRenderer interface {
	RenderTool(sty *styles.Styles, width int, opts *ToolRenderOpts) string
}

// ToolRendererFunc 是实现 [ToolRenderer] 接口的函数类型
type ToolRendererFunc func(sty *styles.Styles, width int, opts *ToolRenderOpts) string

// RenderTool 实现 ToolRenderer 接口
func (f ToolRendererFunc) RenderTool(sty *styles.Styles, width int, opts *ToolRenderOpts) string {
	return f(sty, width, opts)
}

// baseToolMessageItem 表示可以在界面中显示的工具调用消息
type baseToolMessageItem struct {
	*highlightableMessageItem
	*cachedMessageItem
	*focusableMessageItem

	toolRenderer ToolRenderer
	toolCall     message.ToolCall
	result       *message.ToolResult
	messageID    string
	status       ToolStatus
	// 用于高效缓存具有宽度限制的工具（例如 bash 和其他工具）
	hasCappedWidth bool
	// isCompact 表示此工具应以紧凑模式渲染
	isCompact bool
	// spinningFunc 允许工具覆盖默认的旋转逻辑
	// 如果为 nil，则使用默认逻辑：!toolCall.Finished && !canceled
	spinningFunc SpinningFunc

	sty             *styles.Styles
	anim            *anim.Anim
	expandedContent bool
}

var _ Expandable = (*baseToolMessageItem)(nil)

// newBaseToolMessageItem 是基础工具消息项的内部构造函数
func newBaseToolMessageItem(
	sty *styles.Styles,
	toolCall message.ToolCall,
	result *message.ToolResult,
	toolRenderer ToolRenderer,
	canceled bool,
) *baseToolMessageItem {
	// 目前只为 diff 工具使用全宽显示（据我所知）
	hasCappedWidth := toolCall.Name != tools.EditToolName && toolCall.Name != tools.MultiEditToolName

	status := ToolStatusRunning
	if canceled {
		status = ToolStatusCanceled
	}

	t := &baseToolMessageItem{
		highlightableMessageItem: defaultHighlighter(sty),
		cachedMessageItem:        &cachedMessageItem{},
		focusableMessageItem:     &focusableMessageItem{},
		sty:                      sty,
		toolRenderer:             toolRenderer,
		toolCall:                 toolCall,
		result:                   result,
		status:                   status,
		hasCappedWidth:           hasCappedWidth,
	}
	t.anim = anim.New(anim.Settings{
		ID:          toolCall.ID,
		Size:        15,
		GradColorA:  sty.Primary,
		GradColorB:  sty.Secondary,
		LabelColor:  sty.FgBase,
		CycleColors: true,
	})

	return t
}

// NewToolMessageItem 根据工具调用名称创建新的 [ToolMessageItem]
//
// 如果已实现特定工具消息项类型，则返回该类型，
// 否则返回通用工具消息项。messageID 是包含此工具调用的助手消息的 ID。
func NewToolMessageItem(
	sty *styles.Styles,
	messageID string,
	toolCall message.ToolCall,
	result *message.ToolResult,
	canceled bool,
) ToolMessageItem {
	var item ToolMessageItem
	switch toolCall.Name {
	case tools.BashToolName:
		item = NewBashToolMessageItem(sty, toolCall, result, canceled)
	case tools.JobOutputToolName:
		item = NewJobOutputToolMessageItem(sty, toolCall, result, canceled)
	case tools.JobKillToolName:
		item = NewJobKillToolMessageItem(sty, toolCall, result, canceled)
	case tools.ViewToolName:
		item = NewViewToolMessageItem(sty, toolCall, result, canceled)
	case tools.WriteToolName:
		item = NewWriteToolMessageItem(sty, toolCall, result, canceled)
	case tools.EditToolName:
		item = NewEditToolMessageItem(sty, toolCall, result, canceled)
	case tools.MultiEditToolName:
		item = NewMultiEditToolMessageItem(sty, toolCall, result, canceled)
	case tools.GlobToolName:
		item = NewGlobToolMessageItem(sty, toolCall, result, canceled)
	case tools.GrepToolName:
		item = NewGrepToolMessageItem(sty, toolCall, result, canceled)
	case tools.LSToolName:
		item = NewLSToolMessageItem(sty, toolCall, result, canceled)
	case tools.DownloadToolName:
		item = NewDownloadToolMessageItem(sty, toolCall, result, canceled)
	case tools.FetchToolName:
		item = NewFetchToolMessageItem(sty, toolCall, result, canceled)
	case tools.SourcegraphToolName:
		item = NewSourcegraphToolMessageItem(sty, toolCall, result, canceled)
	case tools.DiagnosticsToolName:
		item = NewDiagnosticsToolMessageItem(sty, toolCall, result, canceled)
	case agent.AgentToolName:
		item = NewAgentToolMessageItem(sty, toolCall, result, canceled)
	case tools.AgenticFetchToolName:
		item = NewAgenticFetchToolMessageItem(sty, toolCall, result, canceled)
	case tools.WebFetchToolName:
		item = NewWebFetchToolMessageItem(sty, toolCall, result, canceled)
	case tools.WebSearchToolName:
		item = NewWebSearchToolMessageItem(sty, toolCall, result, canceled)
	case tools.TodosToolName:
		item = NewTodosToolMessageItem(sty, toolCall, result, canceled)
	case tools.ReferencesToolName:
		item = NewReferencesToolMessageItem(sty, toolCall, result, canceled)
	case tools.LSPRestartToolName:
		item = NewLSPRestartToolMessageItem(sty, toolCall, result, canceled)
	default:
		if strings.HasPrefix(toolCall.Name, "mcp_") {
			item = NewMCPToolMessageItem(sty, toolCall, result, canceled)
		} else {
			item = NewGenericToolMessageItem(sty, toolCall, result, canceled)
		}
	}
	item.SetMessageID(messageID)
	return item
}

// SetCompact 实现 Compactable 接口
func (t *baseToolMessageItem) SetCompact(compact bool) {
	t.isCompact = compact
	t.clearCache()
}

// ID 返回此工具消息项的唯一标识符
func (t *baseToolMessageItem) ID() string {
	return t.toolCall.ID
}

// StartAnimation 如果应该旋转，则启动助手消息动画
func (t *baseToolMessageItem) StartAnimation() tea.Cmd {
	if !t.isSpinning() {
		return nil
	}
	return t.anim.Start()
}

// Animate 如果应该旋转，则推进助手消息动画
func (t *baseToolMessageItem) Animate(msg anim.StepMsg) tea.Cmd {
	if !t.isSpinning() {
		return nil
	}
	return t.anim.Animate(msg)
}

// RawRender 实现 [MessageItem]
func (t *baseToolMessageItem) RawRender(width int) string {
	toolItemWidth := width - MessageLeftPaddingTotal
	if t.hasCappedWidth {
		toolItemWidth = cappedMessageWidth(width)
	}

	content, height, ok := t.getCachedRender(toolItemWidth)
	// 如果正在旋转或没有缓存，则重新渲染
	if !ok || t.isSpinning() {
		content = t.toolRenderer.RenderTool(t.sty, toolItemWidth, &ToolRenderOpts{
			ToolCall:        t.toolCall,
			Result:          t.result,
			Anim:            t.anim,
			ExpandedContent: t.expandedContent,
			Compact:         t.isCompact,
			IsSpinning:      t.isSpinning(),
			Status:          t.computeStatus(),
		})
		height = lipgloss.Height(content)
		// 缓存渲染的内容
		t.setCachedRender(content, toolItemWidth, height)
	}

	return t.renderHighlighted(content, toolItemWidth, height)
}

// Render 以给定宽度渲染工具消息项
func (t *baseToolMessageItem) Render(width int) string {
	style := t.sty.Chat.Message.ToolCallBlurred
	if t.focused {
		style = t.sty.Chat.Message.ToolCallFocused
	}

	if t.isCompact {
		style = t.sty.Chat.Message.ToolCallCompact
	}

	return style.Render(t.RawRender(width))
}

// ToolCall 返回与此消息项关联的工具调用
func (t *baseToolMessageItem) ToolCall() message.ToolCall {
	return t.toolCall
}

// SetToolCall 设置与此消息项关联的工具调用
func (t *baseToolMessageItem) SetToolCall(tc message.ToolCall) {
	t.toolCall = tc
	t.clearCache()
}

// SetResult 设置与此消息项关联的工具结果
func (t *baseToolMessageItem) SetResult(res *message.ToolResult) {
	t.result = res
	t.clearCache()
}

// MessageID 返回包含此工具调用的消息的 ID
func (t *baseToolMessageItem) MessageID() string {
	return t.messageID
}

// SetMessageID 设置包含此工具调用的消息的 ID
func (t *baseToolMessageItem) SetMessageID(id string) {
	t.messageID = id
}

// SetStatus 设置工具状态
func (t *baseToolMessageItem) SetStatus(status ToolStatus) {
	t.status = status
	t.clearCache()
}

// Status 返回当前工具状态
func (t *baseToolMessageItem) Status() ToolStatus {
	return t.status
}

// computeStatus 计算考虑结果后的有效状态
func (t *baseToolMessageItem) computeStatus() ToolStatus {
	if t.result != nil {
		if t.result.IsError {
			return ToolStatusError
		}
		return ToolStatusSuccess
	}
	return t.status
}

// isSpinning 返回工具是否应该显示动画
func (t *baseToolMessageItem) isSpinning() bool {
	if t.spinningFunc != nil {
		return t.spinningFunc(SpinningState{
			ToolCall: t.toolCall,
			Result:   t.result,
			Status:   t.status,
		})
	}
	return !t.toolCall.Finished && t.status != ToolStatusCanceled
}

// SetSpinningFunc 设置自定义函数以确定工具是否应该旋转
func (t *baseToolMessageItem) SetSpinningFunc(fn SpinningFunc) {
	t.spinningFunc = fn
}

// ToggleExpanded 切换思考框的展开状态
func (t *baseToolMessageItem) ToggleExpanded() bool {
	t.expandedContent = !t.expandedContent
	t.clearCache()
	return t.expandedContent
}

// HandleMouseClick 实现 MouseClickable
func (t *baseToolMessageItem) HandleMouseClick(btn ansi.MouseButton, x, y int) bool {
	return btn == ansi.MouseLeft
}

// HandleKeyEvent 实现 KeyEventHandler
func (t *baseToolMessageItem) HandleKeyEvent(key tea.KeyMsg) (bool, tea.Cmd) {
	if k := key.String(); k == "c" || k == "y" {
		text := t.formatToolForCopy()
		return true, common.CopyToClipboard(text, "工具内容已复制到剪贴板")
	}
	return false, nil
}

// pendingTool 渲染仍在进行中并带有动画的工具
func pendingTool(sty *styles.Styles, name string, anim *anim.Anim) string {
	icon := sty.Tool.IconPending.Render()
	toolName := sty.Tool.NameNormal.Render(name)

	var animView string
	if anim != nil {
		animView = anim.Render()
	}

	return fmt.Sprintf("%s %s %s", icon, toolName, animView)
}

// toolEarlyStateContent 在内容渲染之前处理错误/取消/等待状态
// 返回渲染的输出和是否处理了早期状态
func toolEarlyStateContent(sty *styles.Styles, opts *ToolRenderOpts, width int) (string, bool) {
	var msg string
	switch opts.Status {
	case ToolStatusError:
		msg = toolErrorContent(sty, opts.Result, width)
	case ToolStatusCanceled:
		msg = sty.Tool.StateCancelled.Render("已取消。")
	case ToolStatusAwaitingPermission:
		msg = sty.Tool.StateWaiting.Render("正在请求权限...")
	case ToolStatusRunning:
		msg = sty.Tool.StateWaiting.Render("等待工具响应...")
	default:
		return "", false
	}
	return msg, true
}

// toolErrorContent 使用 ERROR 标签格式化错误消息
func toolErrorContent(sty *styles.Styles, result *message.ToolResult, width int) string {
	if result == nil {
		return ""
	}
	errContent := strings.ReplaceAll(result.Content, "\n", " ")
	errTag := sty.Tool.ErrorTag.Render("错误")
	tagWidth := lipgloss.Width(errTag)
	errContent = ansi.Truncate(errContent, width-tagWidth-3, "…")
	return fmt.Sprintf("%s %s", errTag, sty.Tool.ErrorMessage.Render(errContent))
}

// toolIcon 返回工具调用的状态图标
// toolIcon 根据工具调用的状态返回状态图标
func toolIcon(sty *styles.Styles, status ToolStatus) string {
	switch status {
	case ToolStatusSuccess:
		return sty.Tool.IconSuccess.String()
	case ToolStatusError:
		return sty.Tool.IconError.String()
	case ToolStatusCanceled:
		return sty.Tool.IconCancelled.String()
	default:
		return sty.Tool.IconPending.String()
	}
}

// toolParamList 将参数格式化为 "main (key=value, ...)" 并进行截断
// toolParamList 将工具参数格式化为 "main (key=value, ...)" 并进行截断
func toolParamList(sty *styles.Styles, params []string, width int) string {
	// minSpaceForMainParam 是主参数所需的最小空间
	// 如果小于此值，则只显示主参数，不显示其他内容
	const minSpaceForMainParam = 30
	if len(params) == 0 {
		return ""
	}

	mainParam := params[0]

	// 从剩余参数构建 key=value 对（连续的键值对）
	var kvPairs []string
	for i := 1; i+1 < len(params); i += 2 {
		if params[i+1] != "" {
			kvPairs = append(kvPairs, fmt.Sprintf("%s=%s", params[i], params[i+1]))
		}
	}

	// 如果有足够空间，尝试包含 key=value 对
	output := mainParam
	if len(kvPairs) > 0 {
		partsStr := strings.Join(kvPairs, ", ")
		if remaining := width - lipgloss.Width(partsStr) - 3; remaining >= minSpaceForMainParam {
			output = fmt.Sprintf("%s (%s)", mainParam, partsStr)
		}
	}

	if width >= 0 {
		output = ansi.Truncate(output, width, "…")
	}
	return sty.Tool.ParamMain.Render(output)
}

// toolHeader 构建工具标题行："● ToolName params..."
func toolHeader(sty *styles.Styles, status ToolStatus, name string, width int, nested bool, params ...string) string {
	icon := toolIcon(sty, status)
	nameStyle := sty.Tool.NameNormal
	if nested {
		nameStyle = sty.Tool.NameNested
	}
	toolName := nameStyle.Render(name)
	prefix := fmt.Sprintf("%s %s ", icon, toolName)
	prefixWidth := lipgloss.Width(prefix)
	remainingWidth := width - prefixWidth
	paramsStr := toolParamList(sty, params, remainingWidth)
	return prefix + paramsStr
}

// toolOutputPlainContent 渲染纯文本，支持可选的展开功能
func toolOutputPlainContent(sty *styles.Styles, content string, width int, expanded bool) string {
	content = stringext.NormalizeSpace(content)
	lines := strings.Split(content, "\n")

	maxLines := responseContextHeight
	if expanded {
		maxLines = len(lines) // 显示所有行
	}

	var out []string
	for i, ln := range lines {
		if i >= maxLines {
			break
		}
		ln = " " + ln
		if lipgloss.Width(ln) > width {
			ln = ansi.Truncate(ln, width, "…")
		}
		out = append(out, sty.Tool.ContentLine.Width(width).Render(ln))
	}

	wasTruncated := len(lines) > responseContextHeight

	if !expanded && wasTruncated {
		out = append(out, sty.Tool.ContentTruncation.
			Width(width).
			Render(fmt.Sprintf(assistantMessageTruncateFormat, len(lines)-responseContextHeight)))
	}

	return strings.Join(out, "\n")
}

// toolOutputCodeContent 渲染代码，支持语法高亮和行号
func toolOutputCodeContent(sty *styles.Styles, path, content string, offset, width int, expanded bool) string {
	content = stringext.NormalizeSpace(content)

	lines := strings.Split(content, "\n")
	maxLines := responseContextHeight
	if expanded {
		maxLines = len(lines)
	}

	// 如有需要则截断
	displayLines := lines
	if len(lines) > maxLines {
		displayLines = lines[:maxLines]
	}

	bg := sty.Tool.ContentCodeBg
	highlighted, _ := common.SyntaxHighlight(sty, strings.Join(displayLines, "\n"), path, bg)
	highlightedLines := strings.Split(highlighted, "\n")

	// 计算行号宽度
	maxLineNumber := len(displayLines) + offset
	maxDigits := getDigits(maxLineNumber)
	numFmt := fmt.Sprintf("%%%dd", maxDigits)

	bodyWidth := width - toolBodyLeftPaddingTotal
	codeWidth := bodyWidth - maxDigits

	var out []string
	for i, ln := range highlightedLines {
		lineNum := sty.Tool.ContentLineNumber.Render(fmt.Sprintf(numFmt, i+1+offset))

		// 截断时考虑将要添加的填充
		ln = ansi.Truncate(ln, codeWidth-sty.Tool.ContentCodeLine.GetHorizontalPadding(), "…")

		codeLine := sty.Tool.ContentCodeLine.
			Width(codeWidth).
			Render(ln)

		out = append(out, lipgloss.JoinHorizontal(lipgloss.Left, lineNum, codeLine))
	}

	// 如有需要添加截断消息
	if len(lines) > maxLines && !expanded {
		out = append(out, sty.Tool.ContentCodeTruncation.
			Width(width).
			Render(fmt.Sprintf(assistantMessageTruncateFormat, len(lines)-maxLines)),
		)
	}

	return sty.Tool.Body.Render(strings.Join(out, "\n"))
}

// toolOutputImageContent 渲染图像数据及大小信息
func toolOutputImageContent(sty *styles.Styles, data, mediaType string) string {
	dataSize := len(data) * 3 / 4
	sizeStr := formatSize(dataSize)

	loaded := sty.Base.Foreground(sty.Green).Render("已加载")
	arrow := sty.Base.Foreground(sty.GreenDark).Render("→")
	typeStyled := sty.Base.Render(mediaType)
	sizeStyled := sty.Subtle.Render(sizeStr)

	return sty.Tool.Body.Render(fmt.Sprintf("%s %s %s %s", loaded, arrow, typeStyled, sizeStyled))
}

// getDigits 返回数字的位数
func getDigits(n int) int {
	if n == 0 {
		return 1
	}
	if n < 0 {
		n = -n
	}
	digits := 0
	for n > 0 {
		n /= 10
		digits++
	}
	return digits
}

// formatSize 将字节大小格式化为人类可读的格式
func formatSize(bytes int) string {
	const (
		kb = 1024
		mb = kb * 1024
	)
	switch {
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// toolOutputDiffContent 渲染旧内容和新内容之间的差异
func toolOutputDiffContent(sty *styles.Styles, file, oldContent, newContent string, width int, expanded bool) string {
	bodyWidth := width - toolBodyLeftPaddingTotal

	formatter := common.DiffFormatter(sty).
		Before(file, oldContent).
		After(file, newContent).
		Width(bodyWidth)

	// 对宽终端使用分屏视图
	if width > maxTextWidth {
		formatter = formatter.Split()
	}

	formatted := formatter.String()
	lines := strings.Split(formatted, "\n")

	// 如有需要则截断
	maxLines := responseContextHeight
	if expanded {
		maxLines = len(lines)
	}

	if len(lines) > maxLines && !expanded {
		truncMsg := sty.Tool.DiffTruncation.
			Width(bodyWidth).
			Render(fmt.Sprintf(assistantMessageTruncateFormat, len(lines)-maxLines))
		formatted = strings.Join(lines[:maxLines], "\n") + "\n" + truncMsg
	}

	return sty.Tool.Body.Render(formatted)
}

// formatTimeout 将超时秒数转换为持续时间字符串（例如 "30s"）
// 如果超时为 0，则返回空字符串
func formatTimeout(timeout int) string {
	if timeout == 0 {
		return ""
	}
	return fmt.Sprintf("%ds", timeout)
}

// formatNonZero 返回非零整数的字符串表示，零则返回空字符串
func formatNonZero(value int) string {
	if value == 0 {
		return ""
	}
	return fmt.Sprintf("%d", value)
}

// toolOutputMultiEditDiffContent 渲染差异，可选地包含失败编辑说明
func toolOutputMultiEditDiffContent(sty *styles.Styles, file string, meta tools.MultiEditResponseMetadata, totalEdits, width int, expanded bool) string {
	bodyWidth := width - toolBodyLeftPaddingTotal

	formatter := common.DiffFormatter(sty).
		Before(file, meta.OldContent).
		After(file, meta.NewContent).
		Width(bodyWidth)

	// 对宽终端使用分屏视图
	if width > maxTextWidth {
		formatter = formatter.Split()
	}

	formatted := formatter.String()
	lines := strings.Split(formatted, "\n")

	// 如有需要则截断
	maxLines := responseContextHeight
	if expanded {
		maxLines = len(lines)
	}

	if len(lines) > maxLines && !expanded {
		truncMsg := sty.Tool.DiffTruncation.
			Width(bodyWidth).
			Render(fmt.Sprintf(assistantMessageTruncateFormat, len(lines)-maxLines))
		formatted = truncMsg + "\n" + strings.Join(lines[:maxLines], "\n")
	}

	// 如果存在失败的编辑，添加说明
	if len(meta.EditsFailed) > 0 {
		noteTag := sty.Tool.NoteTag.Render("注意")
		noteMsg := fmt.Sprintf("%d 个编辑中成功 %d 个", totalEdits, meta.EditsApplied)
		note := fmt.Sprintf("%s %s", noteTag, sty.Tool.NoteMessage.Render(noteMsg))
		formatted = formatted + "\n\n" + note
	}

	return sty.Tool.Body.Render(formatted)
}

// roundedEnumerator 创建带有圆角的树枚举器
func roundedEnumerator(lPadding, width int) tree.Enumerator {
	if width == 0 {
		width = 2
	}
	if lPadding == 0 {
		lPadding = 1
	}
	return func(children tree.Children, index int) string {
		line := strings.Repeat("─", width)
		padding := strings.Repeat(" ", lPadding)
		if children.Length()-1 == index {
			return padding + "╰" + line
		}
		return padding + "├" + line
	}
}

// toolOutputMarkdownContent 渲染 Markdown 内容，支持可选的截断
func toolOutputMarkdownContent(sty *styles.Styles, content string, width int, expanded bool) string {
	content = stringext.NormalizeSpace(content)

	// 为可读性限制宽度
	if width > maxTextWidth {
		width = maxTextWidth
	}

	renderer := common.PlainMarkdownRenderer(sty, width)
	rendered, err := renderer.Render(content)
	if err != nil {
		return toolOutputPlainContent(sty, content, width, expanded)
	}

	lines := strings.Split(rendered, "\n")
	maxLines := responseContextHeight
	if expanded {
		maxLines = len(lines)
	}

	var out []string
	for i, ln := range lines {
		if i >= maxLines {
			break
		}
		out = append(out, ln)
	}

	if len(lines) > maxLines && !expanded {
		out = append(out, sty.Tool.ContentTruncation.
			Width(width).
			Render(fmt.Sprintf(assistantMessageTruncateFormat, len(lines)-maxLines)),
		)
	}

	return sty.Tool.Body.Render(strings.Join(out, "\n"))
}

// formatToolForCopy 格式化工具调用以便复制到剪贴板
func (t *baseToolMessageItem) formatToolForCopy() string {
	var parts []string

	toolName := prettifyToolName(t.toolCall.Name)
	parts = append(parts, fmt.Sprintf("## %s 工具调用", toolName))

	if t.toolCall.Input != "" {
		params := t.formatParametersForCopy()
		if params != "" {
			parts = append(parts, "### 参数：")
			parts = append(parts, params)
		}
	}

	if t.result != nil && t.result.ToolCallID != "" {
		if t.result.IsError {
			parts = append(parts, "### 错误：")
			parts = append(parts, t.result.Content)
		} else {
			parts = append(parts, "### 结果：")
			content := t.formatResultForCopy()
			if content != "" {
				parts = append(parts, content)
			}
		}
	} else if t.status == ToolStatusCanceled {
		parts = append(parts, "### 状态：")
		parts = append(parts, "已取消")
	} else {
		parts = append(parts, "### 状态：")
		parts = append(parts, "等待中...")
	}

	return strings.Join(parts, "\n\n")
}

// formatParametersForCopy 格式化工具参数以便复制到剪贴板
func (t *baseToolMessageItem) formatParametersForCopy() string {
	switch t.toolCall.Name {
	case tools.BashToolName:
		var params tools.BashParams
		if json.Unmarshal([]byte(t.toolCall.Input), &params) == nil {
			cmd := strings.ReplaceAll(params.Command, "\n", " ")
			cmd = strings.ReplaceAll(cmd, "\t", "    ")
			return fmt.Sprintf("**命令：** %s", cmd)
		}
	case tools.ViewToolName:
		var params tools.ViewParams
		if json.Unmarshal([]byte(t.toolCall.Input), &params) == nil {
			var parts []string
			parts = append(parts, fmt.Sprintf("**文件：** %s", fsext.PrettyPath(params.FilePath)))
			if params.Limit > 0 {
				parts = append(parts, fmt.Sprintf("**限制：** %d", params.Limit))
			}
			if params.Offset > 0 {
				parts = append(parts, fmt.Sprintf("**偏移：** %d", params.Offset))
			}
			return strings.Join(parts, "\n")
		}
	case tools.EditToolName:
		var params tools.EditParams
		if json.Unmarshal([]byte(t.toolCall.Input), &params) == nil {
			return fmt.Sprintf("**文件：** %s", fsext.PrettyPath(params.FilePath))
		}
	case tools.MultiEditToolName:
		var params tools.MultiEditParams
		if json.Unmarshal([]byte(t.toolCall.Input), &params) == nil {
			var parts []string
			parts = append(parts, fmt.Sprintf("**文件：** %s", fsext.PrettyPath(params.FilePath)))
			parts = append(parts, fmt.Sprintf("**编辑数：** %d", len(params.Edits)))
			return strings.Join(parts, "\n")
		}
	case tools.WriteToolName:
		var params tools.WriteParams
		if json.Unmarshal([]byte(t.toolCall.Input), &params) == nil {
			return fmt.Sprintf("**文件：** %s", fsext.PrettyPath(params.FilePath))
		}
	case tools.FetchToolName:
		var params tools.FetchParams
		if json.Unmarshal([]byte(t.toolCall.Input), &params) == nil {
			var parts []string
			parts = append(parts, fmt.Sprintf("**URL：** %s", params.URL))
			if params.Format != "" {
				parts = append(parts, fmt.Sprintf("**格式：** %s", params.Format))
			}
			if params.Timeout > 0 {
				parts = append(parts, fmt.Sprintf("**超时：** %ds", params.Timeout))
			}
			return strings.Join(parts, "\n")
		}
	case tools.AgenticFetchToolName:
		var params tools.AgenticFetchParams
		if json.Unmarshal([]byte(t.toolCall.Input), &params) == nil {
			var parts []string
			if params.URL != "" {
				parts = append(parts, fmt.Sprintf("**URL：** %s", params.URL))
			}
			if params.Prompt != "" {
				parts = append(parts, fmt.Sprintf("**提示：** %s", params.Prompt))
			}
			return strings.Join(parts, "\n")
		}
	case tools.WebFetchToolName:
		var params tools.WebFetchParams
		if json.Unmarshal([]byte(t.toolCall.Input), &params) == nil {
			return fmt.Sprintf("**URL：** %s", params.URL)
		}
	case tools.GrepToolName:
		var params tools.GrepParams
		if json.Unmarshal([]byte(t.toolCall.Input), &params) == nil {
			var parts []string
			parts = append(parts, fmt.Sprintf("**模式：** %s", params.Pattern))
			if params.Path != "" {
				parts = append(parts, fmt.Sprintf("**路径：** %s", params.Path))
			}
			if params.Include != "" {
				parts = append(parts, fmt.Sprintf("**包含：** %s", params.Include))
			}
			if params.LiteralText {
				parts = append(parts, "**字面量：** true")
			}
			return strings.Join(parts, "\n")
		}
	case tools.GlobToolName:
		var params tools.GlobParams
		if json.Unmarshal([]byte(t.toolCall.Input), &params) == nil {
			var parts []string
			parts = append(parts, fmt.Sprintf("**模式：** %s", params.Pattern))
			if params.Path != "" {
				parts = append(parts, fmt.Sprintf("**路径：** %s", params.Path))
			}
			return strings.Join(parts, "\n")
		}
	case tools.LSToolName:
		var params tools.LSParams
		if json.Unmarshal([]byte(t.toolCall.Input), &params) == nil {
			path := params.Path
			if path == "" {
				path = "."
			}
			return fmt.Sprintf("**路径：** %s", fsext.PrettyPath(path))
		}
	case tools.DownloadToolName:
		var params tools.DownloadParams
		if json.Unmarshal([]byte(t.toolCall.Input), &params) == nil {
			var parts []string
			parts = append(parts, fmt.Sprintf("**URL：** %s", params.URL))
			parts = append(parts, fmt.Sprintf("**文件路径：** %s", fsext.PrettyPath(params.FilePath)))
			if params.Timeout > 0 {
				parts = append(parts, fmt.Sprintf("**超时：** %s", (time.Duration(params.Timeout)*time.Second).String()))
			}
			return strings.Join(parts, "\n")
		}
	case tools.SourcegraphToolName:
		var params tools.SourcegraphParams
		if json.Unmarshal([]byte(t.toolCall.Input), &params) == nil {
			var parts []string
			parts = append(parts, fmt.Sprintf("**查询：** %s", params.Query))
			if params.Count > 0 {
				parts = append(parts, fmt.Sprintf("**数量：** %d", params.Count))
			}
			if params.ContextWindow > 0 {
				parts = append(parts, fmt.Sprintf("**上下文：** %d", params.ContextWindow))
			}
			return strings.Join(parts, "\n")
		}
	case tools.DiagnosticsToolName:
		return "**项目：** 诊断"
	case agent.AgentToolName:
		var params agent.AgentParams
		if json.Unmarshal([]byte(t.toolCall.Input), &params) == nil {
			return fmt.Sprintf("**任务：**\n%s", params.Prompt)
		}
	}

	var params map[string]any
	if json.Unmarshal([]byte(t.toolCall.Input), &params) == nil {
		var parts []string
		for key, value := range params {
			displayKey := strings.ReplaceAll(key, "_", " ")
			if len(displayKey) > 0 {
				displayKey = strings.ToUpper(displayKey[:1]) + displayKey[1:]
			}
			parts = append(parts, fmt.Sprintf("**%s：** %v", displayKey, value))
		}
		return strings.Join(parts, "\n")
	}

	return ""
}

// formatResultForCopy 格式化工具结果以便复制到剪贴板
func (t *baseToolMessageItem) formatResultForCopy() string {
	if t.result == nil {
		return ""
	}

	if t.result.Data != "" {
		if strings.HasPrefix(t.result.MIMEType, "image/") {
			return fmt.Sprintf("[图像：%s]", t.result.MIMEType)
		}
		return fmt.Sprintf("[媒体：%s]", t.result.MIMEType)
	}

	switch t.toolCall.Name {
	case tools.BashToolName:
		return t.formatBashResultForCopy()
	case tools.ViewToolName:
		return t.formatViewResultForCopy()
	case tools.EditToolName:
		return t.formatEditResultForCopy()
	case tools.MultiEditToolName:
		return t.formatMultiEditResultForCopy()
	case tools.WriteToolName:
		return t.formatWriteResultForCopy()
	case tools.FetchToolName:
		return t.formatFetchResultForCopy()
	case tools.AgenticFetchToolName:
		return t.formatAgenticFetchResultForCopy()
	case tools.WebFetchToolName:
		return t.formatWebFetchResultForCopy()
	case agent.AgentToolName:
		return t.formatAgentResultForCopy()
	case tools.DownloadToolName, tools.GrepToolName, tools.GlobToolName, tools.LSToolName, tools.SourcegraphToolName, tools.DiagnosticsToolName, tools.TodosToolName:
		return fmt.Sprintf("```\n%s\n```", t.result.Content)
	default:
		return t.result.Content
	}
}

// formatBashResultForCopy 格式化 bash 工具结果以便复制到剪贴板
func (t *baseToolMessageItem) formatBashResultForCopy() string {
	if t.result == nil {
		return ""
	}

	var meta tools.BashResponseMetadata
	if t.result.Metadata != "" {
		json.Unmarshal([]byte(t.result.Metadata), &meta)
	}

	output := meta.Output
	if output == "" && t.result.Content != tools.BashNoOutput {
		output = t.result.Content
	}

	if output == "" {
		return ""
	}

	return fmt.Sprintf("```bash\n%s\n```", output)
}

// formatViewResultForCopy 格式化 view 工具结果以便复制到剪贴板
func (t *baseToolMessageItem) formatViewResultForCopy() string {
	if t.result == nil {
		return ""
	}

	var meta tools.ViewResponseMetadata
	if t.result.Metadata != "" {
		json.Unmarshal([]byte(t.result.Metadata), &meta)
	}

	if meta.Content == "" {
		return t.result.Content
	}

	lang := ""
	if meta.FilePath != "" {
		ext := strings.ToLower(filepath.Ext(meta.FilePath))
		switch ext {
		case ".go":
			lang = "go"
		case ".js", ".mjs":
			lang = "javascript"
		case ".ts":
			lang = "typescript"
		case ".py":
			lang = "python"
		case ".rs":
			lang = "rust"
		case ".java":
			lang = "java"
		case ".c":
			lang = "c"
		case ".cpp", ".cc", ".cxx":
			lang = "cpp"
		case ".sh", ".bash":
			lang = "bash"
		case ".json":
			lang = "json"
		case ".yaml", ".yml":
			lang = "yaml"
		case ".xml":
			lang = "xml"
		case ".html":
			lang = "html"
		case ".css":
			lang = "css"
		case ".md":
			lang = "markdown"
		}
	}

	var result strings.Builder
	if lang != "" {
		fmt.Fprintf(&result, "```%s\n", lang)
	} else {
		result.WriteString("```\n")
	}
	result.WriteString(meta.Content)
	result.WriteString("\n```")

	return result.String()
}

// formatEditResultForCopy 格式化 edit 工具结果以便复制到剪贴板
func (t *baseToolMessageItem) formatEditResultForCopy() string {
	if t.result == nil || t.result.Metadata == "" {
		if t.result != nil {
			return t.result.Content
		}
		return ""
	}

	var meta tools.EditResponseMetadata
	if json.Unmarshal([]byte(t.result.Metadata), &meta) != nil {
		return t.result.Content
	}

	var params tools.EditParams
	json.Unmarshal([]byte(t.toolCall.Input), &params)

	var result strings.Builder

	if meta.OldContent != "" || meta.NewContent != "" {
		fileName := params.FilePath
		if fileName != "" {
			fileName = fsext.PrettyPath(fileName)
		}
		diffContent, additions, removals := diff.GenerateDiff(meta.OldContent, meta.NewContent, fileName)

		fmt.Fprintf(&result, "变更：+%d -%d\n", additions, removals)
		result.WriteString("```diff\n")
		result.WriteString(diffContent)
		result.WriteString("\n```")
	}

	return result.String()
}

// formatMultiEditResultForCopy 格式化 multi-edit 工具结果以便复制到剪贴板
func (t *baseToolMessageItem) formatMultiEditResultForCopy() string {
	if t.result == nil || t.result.Metadata == "" {
		if t.result != nil {
			return t.result.Content
		}
		return ""
	}

	var meta tools.MultiEditResponseMetadata
	if json.Unmarshal([]byte(t.result.Metadata), &meta) != nil {
		return t.result.Content
	}

	var params tools.MultiEditParams
	json.Unmarshal([]byte(t.toolCall.Input), &params)

	var result strings.Builder
	if meta.OldContent != "" || meta.NewContent != "" {
		fileName := params.FilePath
		if fileName != "" {
			fileName = fsext.PrettyPath(fileName)
		}
		diffContent, additions, removals := diff.GenerateDiff(meta.OldContent, meta.NewContent, fileName)

		fmt.Fprintf(&result, "变更：+%d -%d\n", additions, removals)
		result.WriteString("```diff\n")
		result.WriteString(diffContent)
		result.WriteString("\n```")
	}

	return result.String()
}

// formatWriteResultForCopy 格式化 write 工具结果以便复制到剪贴板
func (t *baseToolMessageItem) formatWriteResultForCopy() string {
	if t.result == nil {
		return ""
	}

	var params tools.WriteParams
	if json.Unmarshal([]byte(t.toolCall.Input), &params) != nil {
		return t.result.Content
	}

	lang := ""
	if params.FilePath != "" {
		ext := strings.ToLower(filepath.Ext(params.FilePath))
		switch ext {
		case ".go":
			lang = "go"
		case ".js", ".mjs":
			lang = "javascript"
		case ".ts":
			lang = "typescript"
		case ".py":
			lang = "python"
		case ".rs":
			lang = "rust"
		case ".java":
			lang = "java"
		case ".c":
			lang = "c"
		case ".cpp", ".cc", ".cxx":
			lang = "cpp"
		case ".sh", ".bash":
			lang = "bash"
		case ".json":
			lang = "json"
		case ".yaml", ".yml":
			lang = "yaml"
		case ".xml":
			lang = "xml"
		case ".html":
			lang = "html"
		case ".css":
			lang = "css"
		case ".md":
			lang = "markdown"
		}
	}

	var result strings.Builder
	fmt.Fprintf(&result, "文件：%s\n", fsext.PrettyPath(params.FilePath))
	if lang != "" {
		fmt.Fprintf(&result, "```%s\n", lang)
	} else {
		result.WriteString("```\n")
	}
	result.WriteString(params.Content)
	result.WriteString("\n```")

	return result.String()
}

// formatFetchResultForCopy 格式化 fetch 工具结果以便复制到剪贴板
func (t *baseToolMessageItem) formatFetchResultForCopy() string {
	if t.result == nil {
		return ""
	}

	var params tools.FetchParams
	if json.Unmarshal([]byte(t.toolCall.Input), &params) != nil {
		return t.result.Content
	}

	var result strings.Builder
	if params.URL != "" {
		fmt.Fprintf(&result, "URL：%s\n", params.URL)
	}
	if params.Format != "" {
		fmt.Fprintf(&result, "格式：%s\n", params.Format)
	}
	if params.Timeout > 0 {
		fmt.Fprintf(&result, "超时：%ds\n", params.Timeout)
	}
	result.WriteString("\n")

	result.WriteString(t.result.Content)

	return result.String()
}

// formatAgenticFetchResultForCopy 格式化 agentic fetch 工具结果以便复制到剪贴板
func (t *baseToolMessageItem) formatAgenticFetchResultForCopy() string {
	if t.result == nil {
		return ""
	}

	var params tools.AgenticFetchParams
	if json.Unmarshal([]byte(t.toolCall.Input), &params) != nil {
		return t.result.Content
	}

	var result strings.Builder
	if params.URL != "" {
		fmt.Fprintf(&result, "URL：%s\n", params.URL)
	}
	if params.Prompt != "" {
		fmt.Fprintf(&result, "提示：%s\n\n", params.Prompt)
	}

	result.WriteString("```markdown\n")
	result.WriteString(t.result.Content)
	result.WriteString("\n```")

	return result.String()
}

// formatWebFetchResultForCopy 格式化 web fetch 工具结果以便复制到剪贴板
func (t *baseToolMessageItem) formatWebFetchResultForCopy() string {
	if t.result == nil {
		return ""
	}

	var params tools.WebFetchParams
	if json.Unmarshal([]byte(t.toolCall.Input), &params) != nil {
		return t.result.Content
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("URL：%s\n\n", params.URL))
	result.WriteString("```markdown\n")
	result.WriteString(t.result.Content)
	result.WriteString("\n```")

	return result.String()
}

// formatAgentResultForCopy 格式化 agent 工具结果以便复制到剪贴板
func (t *baseToolMessageItem) formatAgentResultForCopy() string {
	if t.result == nil {
		return ""
	}

	var result strings.Builder

	if t.result.Content != "" {
		result.WriteString(fmt.Sprintf("```markdown\n%s\n```", t.result.Content))
	}

	return result.String()
}

// prettifyToolName 返回工具名称的可读名称
func prettifyToolName(name string) string {
	switch name {
	case agent.AgentToolName:
		return "代理"
	case tools.BashToolName:
		return "Bash"
	case tools.JobOutputToolName:
		return "任务：输出"
	case tools.JobKillToolName:
		return "任务：终止"
	case tools.DownloadToolName:
		return "下载"
	case tools.EditToolName:
		return "编辑"
	case tools.MultiEditToolName:
		return "多重编辑"
	case tools.FetchToolName:
		return "获取"
	case tools.AgenticFetchToolName:
		return "智能获取"
	case tools.WebFetchToolName:
		return "获取"
	case tools.WebSearchToolName:
		return "搜索"
	case tools.GlobToolName:
		return "Glob"
	case tools.GrepToolName:
		return "Grep"
	case tools.LSToolName:
		return "列表"
	case tools.SourcegraphToolName:
		return "Sourcegraph"
	case tools.TodosToolName:
		return "待办事项"
	case tools.ViewToolName:
		return "查看"
	case tools.WriteToolName:
		return "写入"
	default:
		return genericPrettyName(name)
	}
}
