package chat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/purpose168/crush-cn/internal/agent/tools"
	"github.com/purpose168/crush-cn/internal/fsext"
	"github.com/purpose168/crush-cn/internal/message"
	"github.com/purpose168/crush-cn/internal/ui/styles"
)

// -----------------------------------------------------------------------------
// 查看工具 (View Tool)
// -----------------------------------------------------------------------------

// ViewToolMessageItem 表示查看工具调用的消息项。
type ViewToolMessageItem struct {
	*baseToolMessageItem
}

var _ ToolMessageItem = (*ViewToolMessageItem)(nil)

// NewViewToolMessageItem 创建一个新的 [ViewToolMessageItem]。
func NewViewToolMessageItem(
	sty *styles.Styles,
	toolCall message.ToolCall,
	result *message.ToolResult,
	canceled bool,
) ToolMessageItem {
	return newBaseToolMessageItem(sty, toolCall, result, &ViewToolRenderContext{}, canceled)
}

// ViewToolRenderContext 渲染查看工具消息。
type ViewToolRenderContext struct{}

// RenderTool 实现 [ToolRenderer] 接口。
func (v *ViewToolRenderContext) RenderTool(sty *styles.Styles, width int, opts *ToolRenderOpts) string {
	// 计算限制后的消息宽度
	cappedWidth := cappedMessageWidth(width)
	
	// 如果工具调用处于待处理状态，返回待处理工具显示
	if opts.IsPending() {
		return pendingTool(sty, "View", opts.Anim)
	}

	// 解析工具调用参数
	var params tools.ViewParams
	if err := json.Unmarshal([]byte(opts.ToolCall.Input), &params); err != nil {
		return toolErrorContent(sty, &message.ToolResult{Content: "无效参数"}, cappedWidth)
	}

	// 构建工具参数显示列表
	file := fsext.PrettyPath(params.FilePath)
	toolParams := []string{file}
	if params.Limit != 0 {
		toolParams = append(toolParams, "limit", fmt.Sprintf("%d", params.Limit))
	}
	if params.Offset != 0 {
		toolParams = append(toolParams, "offset", fmt.Sprintf("%d", params.Offset))
	}

	// 生成工具头部信息
	header := toolHeader(sty, opts.Status, "View", cappedWidth, opts.Compact, toolParams...)
	if opts.Compact {
		return header
	}

	// 检查是否有早期状态内容（如错误或取消状态）
	if earlyState, ok := toolEarlyStateContent(sty, opts, cappedWidth); ok {
		return joinToolParts(header, earlyState)
	}

	// 如果没有结果，只返回头部
	if !opts.HasResult() {
		return header
	}

	// 处理图片内容
	if opts.Result.Data != "" && strings.HasPrefix(opts.Result.MIMEType, "image/") {
		body := toolOutputImageContent(sty, opts.Result.Data, opts.Result.MIMEType)
		return joinToolParts(header, body)
	}

	// 优先从元数据中获取内容（包含实际的文件内容）
	var meta tools.ViewResponseMetadata
	content := opts.Result.Content
	if err := json.Unmarshal([]byte(opts.Result.Metadata), &meta); err == nil && meta.Content != "" {
		content = meta.Content
	}

	if content == "" {
		return header
	}

	// 渲染代码内容并进行语法高亮
	body := toolOutputCodeContent(sty, params.FilePath, content, params.Offset, cappedWidth, opts.ExpandedContent)
	return joinToolParts(header, body)
}

// -----------------------------------------------------------------------------
// 写入工具 (Write Tool)
// -----------------------------------------------------------------------------

// WriteToolMessageItem 表示写入工具调用的消息项。
type WriteToolMessageItem struct {
	*baseToolMessageItem
}

var _ ToolMessageItem = (*WriteToolMessageItem)(nil)

// NewWriteToolMessageItem 创建一个新的 [WriteToolMessageItem]。
func NewWriteToolMessageItem(
	sty *styles.Styles,
	toolCall message.ToolCall,
	result *message.ToolResult,
	canceled bool,
) ToolMessageItem {
	return newBaseToolMessageItem(sty, toolCall, result, &WriteToolRenderContext{}, canceled)
}

// WriteToolRenderContext 渲染写入工具消息。
type WriteToolRenderContext struct{}

// RenderTool 实现 [ToolRenderer] 接口。
func (w *WriteToolRenderContext) RenderTool(sty *styles.Styles, width int, opts *ToolRenderOpts) string {
	// 计算限制后的消息宽度
	cappedWidth := cappedMessageWidth(width)
	
	// 如果工具调用处于待处理状态，返回待处理工具显示
	if opts.IsPending() {
		return pendingTool(sty, "Write", opts.Anim)
	}

	// 解析工具调用参数
	var params tools.WriteParams
	if err := json.Unmarshal([]byte(opts.ToolCall.Input), &params); err != nil {
		return toolErrorContent(sty, &message.ToolResult{Content: "无效参数"}, cappedWidth)
	}

	// 生成工具头部信息
	file := fsext.PrettyPath(params.FilePath)
	header := toolHeader(sty, opts.Status, "Write", cappedWidth, opts.Compact, file)
	if opts.Compact {
		return header
	}

	// 检查是否有早期状态内容（如错误或取消状态）
	if earlyState, ok := toolEarlyStateContent(sty, opts, cappedWidth); ok {
		return joinToolParts(header, earlyState)
	}

	// 如果没有内容，只返回头部
	if params.Content == "" {
		return header
	}

	// 渲染代码内容并进行语法高亮
	body := toolOutputCodeContent(sty, params.FilePath, params.Content, 0, cappedWidth, opts.ExpandedContent)
	return joinToolParts(header, body)
}

// -----------------------------------------------------------------------------
// 编辑工具 (Edit Tool)
// -----------------------------------------------------------------------------

// EditToolMessageItem 表示编辑工具调用的消息项。
type EditToolMessageItem struct {
	*baseToolMessageItem
}

var _ ToolMessageItem = (*EditToolMessageItem)(nil)

// NewEditToolMessageItem 创建一个新的 [EditToolMessageItem]。
func NewEditToolMessageItem(
	sty *styles.Styles,
	toolCall message.ToolCall,
	result *message.ToolResult,
	canceled bool,
) ToolMessageItem {
	return newBaseToolMessageItem(sty, toolCall, result, &EditToolRenderContext{}, canceled)
}

// EditToolRenderContext 渲染编辑工具消息。
type EditToolRenderContext struct{}

// RenderTool 实现 [ToolRenderer] 接口。
func (e *EditToolRenderContext) RenderTool(sty *styles.Styles, width int, opts *ToolRenderOpts) string {
	// 编辑工具使用完整宽度显示差异内容
	if opts.IsPending() {
		return pendingTool(sty, "Edit", opts.Anim)
	}

	// 解析工具调用参数
	var params tools.EditParams
	if err := json.Unmarshal([]byte(opts.ToolCall.Input), &params); err != nil {
		return toolErrorContent(sty, &message.ToolResult{Content: "无效参数"}, width)
	}

	// 生成工具头部信息
	file := fsext.PrettyPath(params.FilePath)
	header := toolHeader(sty, opts.Status, "Edit", width, opts.Compact, file)
	if opts.Compact {
		return header
	}

	// 检查是否有早期状态内容（如错误或取消状态）
	if earlyState, ok := toolEarlyStateContent(sty, opts, width); ok {
		return joinToolParts(header, earlyState)
	}

	// 如果没有结果，只返回头部
	if !opts.HasResult() {
		return header
	}

	// 从元数据中获取差异内容
	var meta tools.EditResponseMetadata
	if err := json.Unmarshal([]byte(opts.Result.Metadata), &meta); err != nil {
		// 如果无法解析元数据，显示纯文本内容
		bodyWidth := width - toolBodyLeftPaddingTotal
		body := sty.Tool.Body.Render(toolOutputPlainContent(sty, opts.Result.Content, bodyWidth, opts.ExpandedContent))
		return joinToolParts(header, body)
	}

	// 渲染差异对比内容
	body := toolOutputDiffContent(sty, file, meta.OldContent, meta.NewContent, width, opts.ExpandedContent)
	return joinToolParts(header, body)
}

// -----------------------------------------------------------------------------
// 多重编辑工具 (MultiEdit Tool)
// -----------------------------------------------------------------------------

// MultiEditToolMessageItem 表示多重编辑工具调用的消息项。
type MultiEditToolMessageItem struct {
	*baseToolMessageItem
}

var _ ToolMessageItem = (*MultiEditToolMessageItem)(nil)

// NewMultiEditToolMessageItem 创建一个新的 [MultiEditToolMessageItem]。
func NewMultiEditToolMessageItem(
	sty *styles.Styles,
	toolCall message.ToolCall,
	result *message.ToolResult,
	canceled bool,
) ToolMessageItem {
	return newBaseToolMessageItem(sty, toolCall, result, &MultiEditToolRenderContext{}, canceled)
}

// MultiEditToolRenderContext 渲染多重编辑工具消息。
type MultiEditToolRenderContext struct{}

// RenderTool 实现 [ToolRenderer] 接口。
func (m *MultiEditToolRenderContext) RenderTool(sty *styles.Styles, width int, opts *ToolRenderOpts) string {
	// 多重编辑工具使用完整宽度显示差异内容
	if opts.IsPending() {
		return pendingTool(sty, "Multi-Edit", opts.Anim)
	}

	// 解析工具调用参数
	var params tools.MultiEditParams
	if err := json.Unmarshal([]byte(opts.ToolCall.Input), &params); err != nil {
		return toolErrorContent(sty, &message.ToolResult{Content: "无效参数"}, width)
	}

	// 构建工具参数显示列表
	file := fsext.PrettyPath(params.FilePath)
	toolParams := []string{file}
	if len(params.Edits) > 0 {
		toolParams = append(toolParams, "edits", fmt.Sprintf("%d", len(params.Edits)))
	}

	// 生成工具头部信息
	header := toolHeader(sty, opts.Status, "Multi-Edit", width, opts.Compact, toolParams...)
	if opts.Compact {
		return header
	}

	// 检查是否有早期状态内容（如错误或取消状态）
	if earlyState, ok := toolEarlyStateContent(sty, opts, width); ok {
		return joinToolParts(header, earlyState)
	}

	// 如果没有结果，只返回头部
	if !opts.HasResult() {
		return header
	}

	// 从元数据中获取差异内容
	var meta tools.MultiEditResponseMetadata
	if err := json.Unmarshal([]byte(opts.Result.Metadata), &meta); err != nil {
		// 如果无法解析元数据，显示纯文本内容
		bodyWidth := width - toolBodyLeftPaddingTotal
		body := sty.Tool.Body.Render(toolOutputPlainContent(sty, opts.Result.Content, bodyWidth, opts.ExpandedContent))
		return joinToolParts(header, body)
	}

	// 渲染差异对比内容，并可选显示失败编辑的提示
	body := toolOutputMultiEditDiffContent(sty, file, meta, len(params.Edits), width, opts.ExpandedContent)
	return joinToolParts(header, body)
}

// -----------------------------------------------------------------------------
// 下载工具 (Download Tool)
// -----------------------------------------------------------------------------

// DownloadToolMessageItem 表示下载工具调用的消息项。
type DownloadToolMessageItem struct {
	*baseToolMessageItem
}

var _ ToolMessageItem = (*DownloadToolMessageItem)(nil)

// NewDownloadToolMessageItem 创建一个新的 [DownloadToolMessageItem]。
func NewDownloadToolMessageItem(
	sty *styles.Styles,
	toolCall message.ToolCall,
	result *message.ToolResult,
	canceled bool,
) ToolMessageItem {
	return newBaseToolMessageItem(sty, toolCall, result, &DownloadToolRenderContext{}, canceled)
}

// DownloadToolRenderContext 渲染下载工具消息。
type DownloadToolRenderContext struct{}

// RenderTool 实现 [ToolRenderer] 接口。
func (d *DownloadToolRenderContext) RenderTool(sty *styles.Styles, width int, opts *ToolRenderOpts) string {
	// 计算限制后的消息宽度
	cappedWidth := cappedMessageWidth(width)
	
	// 如果工具调用处于待处理状态，返回待处理工具显示
	if opts.IsPending() {
		return pendingTool(sty, "Download", opts.Anim)
	}

	// 解析工具调用参数
	var params tools.DownloadParams
	if err := json.Unmarshal([]byte(opts.ToolCall.Input), &params); err != nil {
		return toolErrorContent(sty, &message.ToolResult{Content: "无效参数"}, cappedWidth)
	}

	// 构建工具参数显示列表
	toolParams := []string{params.URL}
	if params.FilePath != "" {
		toolParams = append(toolParams, "file_path", fsext.PrettyPath(params.FilePath))
	}
	if params.Timeout != 0 {
		toolParams = append(toolParams, "timeout", formatTimeout(params.Timeout))
	}

	// 生成工具头部信息
	header := toolHeader(sty, opts.Status, "Download", cappedWidth, opts.Compact, toolParams...)
	if opts.Compact {
		return header
	}

	// 检查是否有早期状态内容（如错误或取消状态）
	if earlyState, ok := toolEarlyStateContent(sty, opts, cappedWidth); ok {
		return joinToolParts(header, earlyState)
	}

	// 如果结果为空，只返回头部
	if opts.HasEmptyResult() {
		return header
	}

	// 渲染工具输出内容
	bodyWidth := cappedWidth - toolBodyLeftPaddingTotal
	body := sty.Tool.Body.Render(toolOutputPlainContent(sty, opts.Result.Content, bodyWidth, opts.ExpandedContent))
	return joinToolParts(header, body)
}
