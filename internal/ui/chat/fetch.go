package chat

import (
	"encoding/json"

	"github.com/purpose168/crush-cn/internal/agent/tools"
	"github.com/purpose168/crush-cn/internal/message"
	"github.com/purpose168/crush-cn/internal/ui/styles"
)

// -----------------------------------------------------------------------------
// Fetch 工具
// -----------------------------------------------------------------------------

// FetchToolMessageItem 表示 fetch 工具调用的消息项。
type FetchToolMessageItem struct {
	*baseToolMessageItem
}

var _ ToolMessageItem = (*FetchToolMessageItem)(nil)

// NewFetchToolMessageItem 创建一个新的 [FetchToolMessageItem]。
func NewFetchToolMessageItem(
	sty *styles.Styles,
	toolCall message.ToolCall,
	result *message.ToolResult,
	canceled bool,
) ToolMessageItem {
	return newBaseToolMessageItem(sty, toolCall, result, &FetchToolRenderContext{}, canceled)
}

// FetchToolRenderContext 渲染 fetch 工具消息。
type FetchToolRenderContext struct{}

// RenderTool 实现 [ToolRenderer] 接口。
func (f *FetchToolRenderContext) RenderTool(sty *styles.Styles, width int, opts *ToolRenderOpts) string {
	// 计算受限的消息宽度
	cappedWidth := cappedMessageWidth(width)
	
	// 如果工具调用处于待处理状态，返回待处理工具显示
	if opts.IsPending() {
		return pendingTool(sty, "Fetch", opts.Anim)
	}

	// 解析工具调用参数
	var params tools.FetchParams
	if err := json.Unmarshal([]byte(opts.ToolCall.Input), &params); err != nil {
		return toolErrorContent(sty, &message.ToolResult{Content: "无效参数"}, cappedWidth)
	}

	// 构建工具参数列表
	toolParams := []string{params.URL}
	if params.Format != "" {
		toolParams = append(toolParams, "format", params.Format)
	}
	if params.Timeout != 0 {
		toolParams = append(toolParams, "timeout", formatTimeout(params.Timeout))
	}

	// 生成工具头部信息
	header := toolHeader(sty, opts.Status, "Fetch", cappedWidth, opts.Compact, toolParams...)
	if opts.Compact {
		return header
	}

	// 检查是否有早期状态内容（如错误或取消状态）
	if earlyState, ok := toolEarlyStateContent(sty, opts, cappedWidth); ok {
		return joinToolParts(header, earlyState)
	}

	// 如果结果为空，仅返回头部信息
	if opts.HasEmptyResult() {
		return header
	}

	// 根据格式确定文件扩展名以进行语法高亮
	file := getFileExtensionForFormat(params.Format)
	body := toolOutputCodeContent(sty, file, opts.Result.Content, 0, cappedWidth, opts.ExpandedContent)
	return joinToolParts(header, body)
}

// getFileExtensionForFormat 返回具有适当扩展名的文件名，用于语法高亮显示。
func getFileExtensionForFormat(format string) string {
	switch format {
	case "text":
		return "fetch.txt"
	case "html":
		return "fetch.html"
	default:
		return "fetch.md"
	}
}

// -----------------------------------------------------------------------------
// WebFetch 工具
// -----------------------------------------------------------------------------

// WebFetchToolMessageItem 表示 web_fetch 工具调用的消息项。
type WebFetchToolMessageItem struct {
	*baseToolMessageItem
}

var _ ToolMessageItem = (*WebFetchToolMessageItem)(nil)

// NewWebFetchToolMessageItem 创建一个新的 [WebFetchToolMessageItem]。
func NewWebFetchToolMessageItem(
	sty *styles.Styles,
	toolCall message.ToolCall,
	result *message.ToolResult,
	canceled bool,
) ToolMessageItem {
	return newBaseToolMessageItem(sty, toolCall, result, &WebFetchToolRenderContext{}, canceled)
}

// WebFetchToolRenderContext 渲染 web_fetch 工具消息。
type WebFetchToolRenderContext struct{}

// RenderTool 实现 [ToolRenderer] 接口。
func (w *WebFetchToolRenderContext) RenderTool(sty *styles.Styles, width int, opts *ToolRenderOpts) string {
	// 计算受限的消息宽度
	cappedWidth := cappedMessageWidth(width)
	
	// 如果工具调用处于待处理状态，返回待处理工具显示
	if opts.IsPending() {
		return pendingTool(sty, "Fetch", opts.Anim)
	}

	// 解析工具调用参数
	var params tools.WebFetchParams
	if err := json.Unmarshal([]byte(opts.ToolCall.Input), &params); err != nil {
		return toolErrorContent(sty, &message.ToolResult{Content: "无效参数"}, cappedWidth)
	}

	// 构建工具参数列表
	toolParams := []string{params.URL}
	header := toolHeader(sty, opts.Status, "Fetch", cappedWidth, opts.Compact, toolParams...)
	if opts.Compact {
		return header
	}

	// 检查是否有早期状态内容（如错误或取消状态）
	if earlyState, ok := toolEarlyStateContent(sty, opts, cappedWidth); ok {
		return joinToolParts(header, earlyState)
	}

	// 如果结果为空，仅返回头部信息
	if opts.HasEmptyResult() {
		return header
	}

	// 渲染 Markdown 格式的输出内容
	body := toolOutputMarkdownContent(sty, opts.Result.Content, cappedWidth, opts.ExpandedContent)
	return joinToolParts(header, body)
}

// -----------------------------------------------------------------------------
// WebSearch 工具
// -----------------------------------------------------------------------------

// WebSearchToolMessageItem 表示 web_search 工具调用的消息项。
type WebSearchToolMessageItem struct {
	*baseToolMessageItem
}

var _ ToolMessageItem = (*WebSearchToolMessageItem)(nil)

// NewWebSearchToolMessageItem 创建一个新的 [WebSearchToolMessageItem]。
func NewWebSearchToolMessageItem(
	sty *styles.Styles,
	toolCall message.ToolCall,
	result *message.ToolResult,
	canceled bool,
) ToolMessageItem {
	return newBaseToolMessageItem(sty, toolCall, result, &WebSearchToolRenderContext{}, canceled)
}

// WebSearchToolRenderContext 渲染 web_search 工具消息。
type WebSearchToolRenderContext struct{}

// RenderTool 实现 [ToolRenderer] 接口。
func (w *WebSearchToolRenderContext) RenderTool(sty *styles.Styles, width int, opts *ToolRenderOpts) string {
	// 计算受限的消息宽度
	cappedWidth := cappedMessageWidth(width)
	
	// 如果工具调用处于待处理状态，返回待处理工具显示
	if opts.IsPending() {
		return pendingTool(sty, "Search", opts.Anim)
	}

	// 解析工具调用参数
	var params tools.WebSearchParams
	if err := json.Unmarshal([]byte(opts.ToolCall.Input), &params); err != nil {
		return toolErrorContent(sty, &message.ToolResult{Content: "无效参数"}, cappedWidth)
	}

	// 构建工具参数列表
	toolParams := []string{params.Query}
	header := toolHeader(sty, opts.Status, "Search", cappedWidth, opts.Compact, toolParams...)
	if opts.Compact {
		return header
	}

	// 检查是否有早期状态内容（如错误或取消状态）
	if earlyState, ok := toolEarlyStateContent(sty, opts, cappedWidth); ok {
		return joinToolParts(header, earlyState)
	}

	// 如果结果为空，仅返回头部信息
	if opts.HasEmptyResult() {
		return header
	}

	// 渲染 Markdown 格式的输出内容
	body := toolOutputMarkdownContent(sty, opts.Result.Content, cappedWidth, opts.ExpandedContent)
	return joinToolParts(header, body)
}
