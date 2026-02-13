package chat

import (
	"encoding/json"

	"github.com/purpose168/crush-cn/internal/agent/tools"
	"github.com/purpose168/crush-cn/internal/message"
	"github.com/purpose168/crush-cn/internal/ui/styles"
)

// LSPRestartToolMessageItem 是表示 LSP 重启工具调用的消息项。
// 该结构体用于封装 LSP (Language Server Protocol) 重启工具的消息内容，
// 继承自 baseToolMessageItem，提供基础的工具消息处理功能。
type LSPRestartToolMessageItem struct {
	*baseToolMessageItem
}

var _ ToolMessageItem = (*LSPRestartToolMessageItem)(nil)

// NewLSPRestartToolMessageItem 创建一个新的 [LSPRestartToolMessageItem] 实例。
// 参数说明：
//   - sty: 样式配置对象，用于定义消息的显示样式
//   - toolCall: 工具调用信息，包含工具名称和输入参数
//   - result: 工具执行结果，包含输出内容和执行状态
//   - canceled: 标识工具调用是否被取消
// 返回值：实现了 ToolMessageItem 接口的消息项实例
func NewLSPRestartToolMessageItem(
	sty *styles.Styles,
	toolCall message.ToolCall,
	result *message.ToolResult,
	canceled bool,
) ToolMessageItem {
	return newBaseToolMessageItem(sty, toolCall, result, &LSPRestartToolRenderContext{}, canceled)
}

// LSPRestartToolRenderContext 渲染 LSP 重启工具消息的上下文。
// 该结构体实现了 ToolRenderer 接口，负责将 LSP 重启工具的调用和结果
// 转换为用户界面可显示的格式化文本。
type LSPRestartToolRenderContext struct{}

// RenderTool 实现 [ToolRenderer] 接口，渲染工具消息的完整内容。
// 参数说明：
//   - sty: 样式配置对象
//   - width: 可用的显示宽度（字符数）
//   - opts: 工具渲染选项，包含工具调用信息、结果和显示状态
// 返回值：格式化后的工具消息字符串
func (r *LSPRestartToolRenderContext) RenderTool(sty *styles.Styles, width int, opts *ToolRenderOpts) string {
	// 计算限制后的消息宽度，确保内容不会超出显示区域
	cappedWidth := cappedMessageWidth(width)
	
	// 如果工具调用处于待处理状态，显示加载动画
	if opts.IsPending() {
		return pendingTool(sty, "重启 LSP", opts.Anim)
	}

	// 解析工具调用参数，提取 LSP 名称等配置信息
	var params tools.LSPRestartParams
	_ = json.Unmarshal([]byte(opts.ToolCall.Input), &params)

	// 构建工具参数列表，用于在工具头部显示
	var toolParams []string
	if params.Name != "" {
		toolParams = append(toolParams, params.Name)
	}

	// 生成工具头部信息，包含状态图标、工具名称和参数
	header := toolHeader(sty, opts.Status, "重启 LSP", cappedWidth, opts.Compact, toolParams...)
	
	// 如果是紧凑模式，仅返回头部信息
	if opts.Compact {
		return header
	}

	// 如果存在早期状态内容（如错误或取消信息），则返回头部和早期状态的组合
	if earlyState, ok := toolEarlyStateContent(sty, opts, cappedWidth); ok {
		return joinToolParts(header, earlyState)
	}

	// 如果结果为空，仅返回头部信息
	if opts.HasEmptyResult() {
		return header
	}

	// 计算工具主体内容的可用宽度（减去左侧内边距）
	bodyWidth := cappedWidth - toolBodyLeftPaddingTotal
	// 渲染工具主体内容，包含工具执行的输出结果
	body := sty.Tool.Body.Render(toolOutputPlainContent(sty, opts.Result.Content, bodyWidth, opts.ExpandedContent))
	return joinToolParts(header, body)
}
