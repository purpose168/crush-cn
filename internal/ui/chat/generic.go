package chat

import (
	"encoding/json"
	"strings"

	"github.com/purpose168/crush-cn/internal/message"
	"github.com/purpose168/crush-cn/internal/stringext"
	"github.com/purpose168/crush-cn/internal/ui/styles"
)

// GenericToolMessageItem 是表示未知工具调用的消息项。
type GenericToolMessageItem struct {
	*baseToolMessageItem
}

var _ ToolMessageItem = (*GenericToolMessageItem)(nil)

// NewGenericToolMessageItem 创建一个新的 [GenericToolMessageItem]。
func NewGenericToolMessageItem(
	sty *styles.Styles,
	toolCall message.ToolCall,
	result *message.ToolResult,
	canceled bool,
) ToolMessageItem {
	return newBaseToolMessageItem(sty, toolCall, result, &GenericToolRenderContext{}, canceled)
}

// GenericToolRenderContext 渲染未知/通用工具消息。
type GenericToolRenderContext struct{}

// RenderTool 实现 [ToolRenderer] 接口。
func (g *GenericToolRenderContext) RenderTool(sty *styles.Styles, width int, opts *ToolRenderOpts) string {
	cappedWidth := cappedMessageWidth(width)
	name := genericPrettyName(opts.ToolCall.Name)

	if opts.IsPending() {
		return pendingTool(sty, name, opts.Anim)
	}

	var params map[string]any
	if err := json.Unmarshal([]byte(opts.ToolCall.Input), &params); err != nil {
		return toolErrorContent(sty, &message.ToolResult{Content: "无效参数"}, cappedWidth)
	}

	var toolParams []string
	if len(params) > 0 {
		parsed, _ := json.Marshal(params)
		toolParams = append(toolParams, string(parsed))
	}

	header := toolHeader(sty, opts.Status, name, cappedWidth, opts.Compact, toolParams...)
	if opts.Compact {
		return header
	}

	if earlyState, ok := toolEarlyStateContent(sty, opts, cappedWidth); ok {
		return joinToolParts(header, earlyState)
	}

	if !opts.HasResult() || opts.Result.Content == "" {
		return header
	}

	bodyWidth := cappedWidth - toolBodyLeftPaddingTotal

	// 处理图像数据。
	if opts.Result.Data != "" && strings.HasPrefix(opts.Result.MIMEType, "image/") {
		body := sty.Tool.Body.Render(toolOutputImageContent(sty, opts.Result.Data, opts.Result.MIMEType))
		return joinToolParts(header, body)
	}

	// 尝试将结果解析为 JSON 以便美观显示。
	var result json.RawMessage
	var body string
	if err := json.Unmarshal([]byte(opts.Result.Content), &result); err == nil {
		prettyResult, err := json.MarshalIndent(result, "", "  ")
		if err == nil {
			body = sty.Tool.Body.Render(toolOutputCodeContent(sty, "result.json", string(prettyResult), 0, bodyWidth, opts.ExpandedContent))
		} else {
			body = sty.Tool.Body.Render(toolOutputPlainContent(sty, opts.Result.Content, bodyWidth, opts.ExpandedContent))
		}
	} else if looksLikeMarkdown(opts.Result.Content) {
		body = sty.Tool.Body.Render(toolOutputCodeContent(sty, "result.md", opts.Result.Content, 0, bodyWidth, opts.ExpandedContent))
	} else {
		body = sty.Tool.Body.Render(toolOutputPlainContent(sty, opts.Result.Content, bodyWidth, opts.ExpandedContent))
	}

	return joinToolParts(header, body)
}

// genericPrettyName 将 snake_case 或 kebab-case 格式的工具名称转换为
// 人类可读的标题格式。
func genericPrettyName(name string) string {
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	return stringext.Capitalize(name)
}
