package chat

import (
	"cmp"
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/purpose168/crush-cn/internal/agent/tools"
	"github.com/purpose168/crush-cn/internal/message"
	"github.com/purpose168/crush-cn/internal/ui/styles"
)

// -----------------------------------------------------------------------------
// Bash 工具
// -----------------------------------------------------------------------------

// BashToolMessageItem 是表示 bash 工具调用的消息项。
type BashToolMessageItem struct {
	*baseToolMessageItem
}

var _ ToolMessageItem = (*BashToolMessageItem)(nil)

// NewBashToolMessageItem 创建一个新的 [BashToolMessageItem]。
func NewBashToolMessageItem(
	sty *styles.Styles,
	toolCall message.ToolCall,
	result *message.ToolResult,
	canceled bool,
) ToolMessageItem {
	return newBaseToolMessageItem(sty, toolCall, result, &BashToolRenderContext{}, canceled)
}

// BashToolRenderContext 渲染 bash 工具消息。
type BashToolRenderContext struct{}

// RenderTool 实现 [ToolRenderer] 接口。
func (b *BashToolRenderContext) RenderTool(sty *styles.Styles, width int, opts *ToolRenderOpts) string {
	cappedWidth := cappedMessageWidth(width)
	// 如果工具调用处于待处理状态，返回待处理工具显示
	if opts.IsPending() {
		return pendingTool(sty, "Bash", opts.Anim)
	}

	var params tools.BashParams
	// 解析工具调用参数，如果解析失败则设置默认命令
	if err := json.Unmarshal([]byte(opts.ToolCall.Input), &params); err != nil {
		params.Command = "failed to parse command"
	}

	// 检查是否为后台作业
	var meta tools.BashResponseMetadata
	if opts.HasResult() {
		_ = json.Unmarshal([]byte(opts.Result.Metadata), &meta)
	}

	// 如果是后台作业，渲染作业工具界面
	if meta.Background {
		description := cmp.Or(meta.Description, params.Command)
		content := "Command: " + params.Command + "\n" + opts.Result.Content
		return renderJobTool(sty, opts, cappedWidth, "Start", meta.ShellID, description, content)
	}

	// 常规 bash 命令处理
	// 将换行符和制表符替换为空格，以便在单行显示
	cmd := strings.ReplaceAll(params.Command, "\n", " ")
	cmd = strings.ReplaceAll(cmd, "\t", "    ")
	toolParams := []string{cmd}
	// 如果命令在后台运行，添加后台标记
	if params.RunInBackground {
		toolParams = append(toolParams, "background", "true")
	}

	// 生成工具头部显示
	header := toolHeader(sty, opts.Status, "Bash", cappedWidth, opts.Compact, toolParams...)
	// 如果是紧凑模式，只返回头部
	if opts.Compact {
		return header
	}

	// 如果存在早期状态内容，返回头部和早期状态
	if earlyState, ok := toolEarlyStateContent(sty, opts, cappedWidth); ok {
		return joinToolParts(header, earlyState)
	}

	// 如果没有结果，只返回头部
	if !opts.HasResult() {
		return header
	}

	// 获取输出内容，优先使用元数据中的输出
	output := meta.Output
	if output == "" && opts.Result.Content != tools.BashNoOutput {
		output = opts.Result.Content
	}
	// 如果没有输出内容，只返回头部
	if output == "" {
		return header
	}

	// 渲染工具主体内容
	bodyWidth := cappedWidth - toolBodyLeftPaddingTotal
	body := sty.Tool.Body.Render(toolOutputPlainContent(sty, output, bodyWidth, opts.ExpandedContent))
	return joinToolParts(header, body)
}

// -----------------------------------------------------------------------------
// 作业输出工具
// -----------------------------------------------------------------------------

// JobOutputToolMessageItem 是 job_output 工具调用的消息项。
type JobOutputToolMessageItem struct {
	*baseToolMessageItem
}

var _ ToolMessageItem = (*JobOutputToolMessageItem)(nil)

// NewJobOutputToolMessageItem 创建一个新的 [JobOutputToolMessageItem]。
func NewJobOutputToolMessageItem(
	sty *styles.Styles,
	toolCall message.ToolCall,
	result *message.ToolResult,
	canceled bool,
) ToolMessageItem {
	return newBaseToolMessageItem(sty, toolCall, result, &JobOutputToolRenderContext{}, canceled)
}

// JobOutputToolRenderContext 渲染 job_output 工具消息。
type JobOutputToolRenderContext struct{}

// RenderTool 实现 [ToolRenderer] 接口。
func (j *JobOutputToolRenderContext) RenderTool(sty *styles.Styles, width int, opts *ToolRenderOpts) string {
	cappedWidth := cappedMessageWidth(width)
	// 如果工具调用处于待处理状态，返回待处理工具显示
	if opts.IsPending() {
		return pendingTool(sty, "Job", opts.Anim)
	}

	var params tools.JobOutputParams
	// 解析工具调用参数，如果解析失败则返回错误内容
	if err := json.Unmarshal([]byte(opts.ToolCall.Input), &params); err != nil {
		return toolErrorContent(sty, &message.ToolResult{Content: "Invalid parameters"}, cappedWidth)
	}

	// 从元数据中获取描述信息
	var description string
	if opts.HasResult() && opts.Result.Metadata != "" {
		var meta tools.JobOutputResponseMetadata
		if err := json.Unmarshal([]byte(opts.Result.Metadata), &meta); err == nil {
			description = cmp.Or(meta.Description, meta.Command)
		}
	}

	// 获取结果内容
	content := ""
	if opts.HasResult() {
		content = opts.Result.Content
	}
	return renderJobTool(sty, opts, cappedWidth, "Output", params.ShellID, description, content)
}

// -----------------------------------------------------------------------------
// 作业终止工具
// -----------------------------------------------------------------------------

// JobKillToolMessageItem 是 job_kill 工具调用的消息项。
type JobKillToolMessageItem struct {
	*baseToolMessageItem
}

var _ ToolMessageItem = (*JobKillToolMessageItem)(nil)

// NewJobKillToolMessageItem 创建一个新的 [JobKillToolMessageItem]。
func NewJobKillToolMessageItem(
	sty *styles.Styles,
	toolCall message.ToolCall,
	result *message.ToolResult,
	canceled bool,
) ToolMessageItem {
	return newBaseToolMessageItem(sty, toolCall, result, &JobKillToolRenderContext{}, canceled)
}

// JobKillToolRenderContext 渲染 job_kill 工具消息。
type JobKillToolRenderContext struct{}

// RenderTool 实现 [ToolRenderer] 接口。
func (j *JobKillToolRenderContext) RenderTool(sty *styles.Styles, width int, opts *ToolRenderOpts) string {
	cappedWidth := cappedMessageWidth(width)
	// 如果工具调用处于待处理状态，返回待处理工具显示
	if opts.IsPending() {
		return pendingTool(sty, "Job", opts.Anim)
	}

	var params tools.JobKillParams
	// 解析工具调用参数，如果解析失败则返回错误内容
	if err := json.Unmarshal([]byte(opts.ToolCall.Input), &params); err != nil {
		return toolErrorContent(sty, &message.ToolResult{Content: "Invalid parameters"}, cappedWidth)
	}

	// 从元数据中获取描述信息
	var description string
	if opts.HasResult() && opts.Result.Metadata != "" {
		var meta tools.JobKillResponseMetadata
		if err := json.Unmarshal([]byte(opts.Result.Metadata), &meta); err == nil {
			description = cmp.Or(meta.Description, meta.Command)
		}
	}

	// 获取结果内容
	content := ""
	if opts.HasResult() {
		content = opts.Result.Content
	}
	return renderJobTool(sty, opts, cappedWidth, "Kill", params.ShellID, description, content)
}

// renderJobTool 渲染作业相关工具，使用通用模式：
// 头部 → 嵌套检查 → 早期状态 → 主体。
func renderJobTool(sty *styles.Styles, opts *ToolRenderOpts, width int, action, shellID, description, content string) string {
	// 生成作业头部
	header := jobHeader(sty, opts.Status, action, shellID, description, width)
	// 如果是紧凑模式，只返回头部
	if opts.Compact {
		return header
	}

	// 如果存在早期状态内容，返回头部和早期状态
	if earlyState, ok := toolEarlyStateContent(sty, opts, width); ok {
		return joinToolParts(header, earlyState)
	}

	// 如果没有内容，只返回头部
	if content == "" {
		return header
	}

	// 渲染工具主体内容
	bodyWidth := width - toolBodyLeftPaddingTotal
	body := sty.Tool.Body.Render(toolOutputPlainContent(sty, content, bodyWidth, opts.ExpandedContent))
	return joinToolParts(header, body)
}

// jobHeader 为作业相关工具构建头部。
// 格式: "● Job (Action) PID shellID description..."
func jobHeader(sty *styles.Styles, status ToolStatus, action, shellID, description string, width int) string {
	// 获取工具图标
	icon := toolIcon(sty, status)
	// 渲染各个部分：作业名称、动作、进程ID
	jobPart := sty.Tool.JobToolName.Render("Job")
	actionPart := sty.Tool.JobAction.Render("(" + action + ")")
	pidPart := sty.Tool.JobPID.Render("PID " + shellID)

	// 组合前缀部分
	prefix := fmt.Sprintf("%s %s %s %s", icon, jobPart, actionPart, pidPart)

	// 如果没有描述，只返回前缀
	if description == "" {
		return prefix
	}

	// 计算可用宽度并截断描述
	prefixWidth := lipgloss.Width(prefix)
	availableWidth := width - prefixWidth - 1
	// 如果可用宽度不足，只返回前缀
	if availableWidth < 10 {
		return prefix
	}

	// 截断描述并渲染
	truncatedDesc := ansi.Truncate(description, availableWidth, "…")
	return prefix + " " + sty.Tool.JobDescription.Render(truncatedDesc)
}

// joinToolParts 使用空行分隔符连接头部和主体。
func joinToolParts(header, body string) string {
	return strings.Join([]string{header, "", body}, "\n")
}
