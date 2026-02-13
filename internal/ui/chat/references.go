package chat

import (
	"encoding/json"

	"github.com/purpose168/crush-cn/internal/agent/tools"
	"github.com/purpose168/crush-cn/internal/fsext"
	"github.com/purpose168/crush-cn/internal/message"
	"github.com/purpose168/crush-cn/internal/ui/styles"
)

// ReferencesToolMessageItem 是表示引用工具调用的消息项。
type ReferencesToolMessageItem struct {
	*baseToolMessageItem
}

var _ ToolMessageItem = (*ReferencesToolMessageItem)(nil)

// NewReferencesToolMessageItem 创建一个新的 [ReferencesToolMessageItem]。
func NewReferencesToolMessageItem(
	sty *styles.Styles,
	toolCall message.ToolCall,
	result *message.ToolResult,
	canceled bool,
) ToolMessageItem {
	return newBaseToolMessageItem(sty, toolCall, result, &ReferencesToolRenderContext{}, canceled)
}

// ReferencesToolRenderContext 渲染引用工具消息。
type ReferencesToolRenderContext struct{}

// RenderTool 实现 [ToolRenderer] 接口。
func (r *ReferencesToolRenderContext) RenderTool(sty *styles.Styles, width int, opts *ToolRenderOpts) string {
	// 计算限制后的消息宽度
	cappedWidth := cappedMessageWidth(width)
	
	// 如果工具调用处于待处理状态，返回待处理工具显示
	if opts.IsPending() {
		return pendingTool(sty, "查找引用", opts.Anim)
	}

	// 解析工具调用参数
	var params tools.ReferencesParams
	_ = json.Unmarshal([]byte(opts.ToolCall.Input), &params)

	// 构建工具参数列表
	toolParams := []string{params.Symbol}
	if params.Path != "" {
		toolParams = append(toolParams, "路径", fsext.PrettyPath(params.Path))
	}

	// 生成工具头部信息
	header := toolHeader(sty, opts.Status, "查找引用", cappedWidth, opts.Compact, toolParams...)
	
	// 如果是紧凑模式，只返回头部信息
	if opts.Compact {
		return header
	}

	// 检查是否有早期状态内容（如错误或取消状态）
	if earlyState, ok := toolEarlyStateContent(sty, opts, cappedWidth); ok {
		return joinToolParts(header, earlyState)
	}

	// 如果结果为空，只返回头部信息
	if opts.HasEmptyResult() {
		return header
	}

	// 计算消息体宽度并渲染工具输出内容
	bodyWidth := cappedWidth - toolBodyLeftPaddingTotal
	body := sty.Tool.Body.Render(toolOutputPlainContent(sty, opts.Result.Content, bodyWidth, opts.ExpandedContent))
	
	// 组合头部和消息体
	return joinToolParts(header, body)
}
