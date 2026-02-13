package chat

import (
	"encoding/json"

	"github.com/purpose168/crush-cn/internal/agent/tools"
	"github.com/purpose168/crush-cn/internal/fsext"
	"github.com/purpose168/crush-cn/internal/message"
	"github.com/purpose168/crush-cn/internal/ui/styles"
)

// -----------------------------------------------------------------------------
// 诊断工具
// -----------------------------------------------------------------------------

// DiagnosticsToolMessageItem 是表示诊断工具调用的消息项。
type DiagnosticsToolMessageItem struct {
	*baseToolMessageItem
}

var _ ToolMessageItem = (*DiagnosticsToolMessageItem)(nil)

// NewDiagnosticsToolMessageItem 创建一个新的 [DiagnosticsToolMessageItem]。
func NewDiagnosticsToolMessageItem(
	sty *styles.Styles,
	toolCall message.ToolCall,
	result *message.ToolResult,
	canceled bool,
) ToolMessageItem {
	return newBaseToolMessageItem(sty, toolCall, result, &DiagnosticsToolRenderContext{}, canceled)
}

// DiagnosticsToolRenderContext 渲染诊断工具消息。
type DiagnosticsToolRenderContext struct{}

// RenderTool 实现 [ToolRenderer] 接口。
func (d *DiagnosticsToolRenderContext) RenderTool(sty *styles.Styles, width int, opts *ToolRenderOpts) string {
	cappedWidth := cappedMessageWidth(width)
	if opts.IsPending() {
		return pendingTool(sty, "Diagnostics", opts.Anim)
	}

	var params tools.DiagnosticsParams
	_ = json.Unmarshal([]byte(opts.ToolCall.Input), &params)

	// 如果没有文件路径，则显示"project"（项目），否则显示文件路径
	mainParam := "project"
	if params.FilePath != "" {
		mainParam = fsext.PrettyPath(params.FilePath)
	}

	header := toolHeader(sty, opts.Status, "Diagnostics", cappedWidth, opts.Compact, mainParam)
	if opts.Compact {
		return header
	}

	if earlyState, ok := toolEarlyStateContent(sty, opts, cappedWidth); ok {
		return joinToolParts(header, earlyState)
	}

	if opts.HasEmptyResult() {
		return header
	}

	bodyWidth := cappedWidth - toolBodyLeftPaddingTotal
	body := sty.Tool.Body.Render(toolOutputPlainContent(sty, opts.Result.Content, bodyWidth, opts.ExpandedContent))
	return joinToolParts(header, body)
}
