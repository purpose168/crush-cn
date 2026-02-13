package chat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/purpose168/crush-cn/internal/message"
	"github.com/purpose168/crush-cn/internal/stringext"
	"github.com/purpose168/crush-cn/internal/ui/styles"
)

// MCPToolMessageItem 是表示 MCP 工具调用的消息项
type MCPToolMessageItem struct {
	*baseToolMessageItem
}

var _ ToolMessageItem = (*MCPToolMessageItem)(nil)

// NewMCPToolMessageItem 创建一个新的 [MCPToolMessageItem]
// 参数:
//   - sty: 样式配置
//   - toolCall: 工具调用信息
//   - result: 工具执行结果
//   - canceled: 是否已取消
func NewMCPToolMessageItem(
	sty *styles.Styles,
	toolCall message.ToolCall,
	result *message.ToolResult,
	canceled bool,
) ToolMessageItem {
	return newBaseToolMessageItem(sty, toolCall, result, &MCPToolRenderContext{}, canceled)
}

// MCPToolRenderContext 渲染 MCP 工具消息
type MCPToolRenderContext struct{}

// RenderTool 实现 [ToolRenderer] 接口
// 渲染 MCP 工具调用的显示内容，包括工具名称、参数和执行结果
func (b *MCPToolRenderContext) RenderTool(sty *styles.Styles, width int, opts *ToolRenderOpts) string {
	// 计算消息的最大宽度
	cappedWidth := cappedMessageWidth(width)
	// 解析工具名称，格式应为: mcp_{server}_{tool}
	toolNameParts := strings.SplitN(opts.ToolCall.Name, "_", 3)
	if len(toolNameParts) != 3 {
		// 工具名称格式无效
		return toolErrorContent(sty, &message.ToolResult{Content: "Invalid tool name"}, cappedWidth)
	}
	mcpName := prettyName(toolNameParts[1])
	toolName := prettyName(toolNameParts[2])

	// 应用样式渲染 MCP 服务器名称和工具名称
	mcpName = sty.Tool.MCPName.Render(mcpName)
	toolName = sty.Tool.MCPToolName.Render(toolName)

	// 组合完整的工具名称显示
	name := fmt.Sprintf("%s %s %s", mcpName, sty.Tool.MCPArrow.String(), toolName)

	if opts.IsPending() {
		// 如果工具调用正在等待中，显示等待状态
		return pendingTool(sty, name, opts.Anim)
	}

	// 解析工具参数
	var params map[string]any
	if err := json.Unmarshal([]byte(opts.ToolCall.Input), &params); err != nil {
		// 参数格式无效
		return toolErrorContent(sty, &message.ToolResult{Content: "Invalid parameters"}, cappedWidth)
	}

	var toolParams []string
	if len(params) > 0 {
		// 将参数序列化为 JSON 字符串
		parsed, _ := json.Marshal(params)
		toolParams = append(toolParams, string(parsed))
	}

	// 生成工具头部信息
	header := toolHeader(sty, opts.Status, name, cappedWidth, opts.Compact, toolParams...)
	if opts.Compact {
		// 紧凑模式下只返回头部信息
		return header
	}

	// 检查是否有早期状态内容（如取消或错误）
	if earlyState, ok := toolEarlyStateContent(sty, opts, cappedWidth); ok {
		return joinToolParts(header, earlyState)
	}

	if !opts.HasResult() || opts.Result.Content == "" {
		// 没有结果内容时只返回头部
		return header
	}

	bodyWidth := cappedWidth - toolBodyLeftPaddingTotal
	// 检查结果是否为 JSON 格式
	var result json.RawMessage
	var body string
	if err := json.Unmarshal([]byte(opts.Result.Content), &result); err == nil {
		// 如果是 JSON，格式化输出
		prettyResult, err := json.MarshalIndent(result, "", "  ")
		if err == nil {
			// 成功格式化 JSON，以代码块形式显示
			body = sty.Tool.Body.Render(toolOutputCodeContent(sty, "result.json", string(prettyResult), 0, bodyWidth, opts.ExpandedContent))
		} else {
			// JSON 格式化失败，以纯文本形式显示
			body = sty.Tool.Body.Render(toolOutputPlainContent(sty, opts.Result.Content, bodyWidth, opts.ExpandedContent))
		}
	} else if looksLikeMarkdown(opts.Result.Content) {
		// 如果内容看起来像 Markdown，以 Markdown 格式显示
		body = sty.Tool.Body.Render(toolOutputCodeContent(sty, "result.md", opts.Result.Content, 0, bodyWidth, opts.ExpandedContent))
	} else {
		// 其他情况以纯文本形式显示
		body = sty.Tool.Body.Render(toolOutputPlainContent(sty, opts.Result.Content, bodyWidth, opts.ExpandedContent))
	}
	// 组合头部和主体内容
	return joinToolParts(header, body)
}

// prettyName 将名称格式化为更易读的形式
// 将下划线和连字符替换为空格，并将首字母大写
func prettyName(name string) string {
	// 替换下划线为空格
	name = strings.ReplaceAll(name, "_", " ")
	// 替换连字符为空格
	name = strings.ReplaceAll(name, "-", " ")
	// 将首字母大写
	return stringext.Capitalize(name)
}

// looksLikeMarkdown 通过检查常见的 Markdown 模式来判断内容是否为 Markdown 格式
func looksLikeMarkdown(content string) bool {
	// 定义常见的 Markdown 模式
	patterns := []string{
		"# ",  // 标题
		"## ", // 标题
		"**",  // 粗体
		"```", // 代码块
		"- ",  // 无序列表
		"1. ", // 有序列表
		"> ",  // 引用块
		"---", // 水平分隔线
		"***", // 水平分隔线
	}
	for _, p := range patterns {
		if strings.Contains(content, p) {
			return true
		}
	}
	return false
}
