package chat

import (
	"encoding/json"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/tree"
	"github.com/purpose168/crush-cn/internal/agent"
	"github.com/purpose168/crush-cn/internal/message"
	"github.com/purpose168/crush-cn/internal/ui/anim"
	"github.com/purpose168/crush-cn/internal/ui/styles"
)

// -----------------------------------------------------------------------------
// 智能体工具 (Agent Tool)
// -----------------------------------------------------------------------------

// NestedToolContainer 是一个接口，用于可以包含嵌套工具调用的工具项。
// 该接口定义了管理嵌套工具的标准方法，支持工具的层级调用结构。
type NestedToolContainer interface {
	// NestedTools 返回所有嵌套的工具项
	NestedTools() []ToolMessageItem
	// SetNestedTools 设置嵌套的工具项列表
	SetNestedTools(tools []ToolMessageItem)
	// AddNestedTool 添加单个嵌套工具项
	AddNestedTool(tool ToolMessageItem)
}

// AgentToolMessageItem 是表示智能体工具调用的消息项。
// 它封装了智能体工具的执行状态、嵌套工具调用和渲染逻辑。
type AgentToolMessageItem struct {
	*baseToolMessageItem

	nestedTools []ToolMessageItem // 嵌套的工具项列表
}

var (
	_ ToolMessageItem     = (*AgentToolMessageItem)(nil) // 确保实现 ToolMessageItem 接口
	_ NestedToolContainer = (*AgentToolMessageItem)(nil) // 确保实现 NestedToolContainer 接口
)

// NewAgentToolMessageItem 创建一个新的 [AgentToolMessageItem] 实例。
// 参数说明：
//   - sty: 样式配置对象，用于控制消息项的视觉呈现
//   - toolCall: 工具调用信息，包含工具名称和输入参数
//   - result: 工具执行结果，可能为 nil（表示工具仍在执行中）
//   - canceled: 标识工具调用是否已被取消
//
// 返回值：初始化后的 AgentToolMessageItem 实例
func NewAgentToolMessageItem(
	sty *styles.Styles,
	toolCall message.ToolCall,
	result *message.ToolResult,
	canceled bool,
) *AgentToolMessageItem {
	t := &AgentToolMessageItem{}
	t.baseToolMessageItem = newBaseToolMessageItem(sty, toolCall, result, &AgentToolRenderContext{agent: t}, canceled)
	// 对于智能体工具，我们保持旋转动画直到工具调用完成
	t.spinningFunc = func(state SpinningState) bool {
		return !state.HasResult() && !state.IsCanceled()
	}
	return t
}

// Animate 推进消息动画，如果应该显示旋转动画的话。
// 该方法处理当前工具项及其嵌套工具的动画更新。
// 参数：
//   - msg: 动画步骤消息，包含动画ID和时间信息
//
// 返回值：tea.Cmd 命令，用于更新动画状态
func (a *AgentToolMessageItem) Animate(msg anim.StepMsg) tea.Cmd {
	// 如果已有结果或已取消，不执行动画
	if a.result != nil || a.Status() == ToolStatusCanceled {
		return nil
	}
	// 检查动画消息是否属于当前工具项
	if msg.ID == a.ID() {
		return a.anim.Animate(msg)
	}
	// 遍历嵌套工具，查找匹配的动画目标
	for _, nestedTool := range a.nestedTools {
		if msg.ID != nestedTool.ID() {
			continue
		}
		// 如果嵌套工具支持动画接口，则执行动画
		if s, ok := nestedTool.(Animatable); ok {
			return s.Animate(msg)
		}
	}
	return nil
}

// NestedTools 返回嵌套的工具项列表。
// 这些嵌套工具代表了智能体在执行过程中调用的子工具。
func (a *AgentToolMessageItem) NestedTools() []ToolMessageItem {
	return a.nestedTools
}

// SetNestedTools 设置嵌套的工具项列表。
// 设置后会清除缓存以触发重新渲染。
func (a *AgentToolMessageItem) SetNestedTools(tools []ToolMessageItem) {
	a.nestedTools = tools
	a.clearCache()
}

// AddNestedTool 添加单个嵌套工具项。
// 嵌套工具会被标记为紧凑渲染模式以优化显示空间。
func (a *AgentToolMessageItem) AddNestedTool(tool ToolMessageItem) {
	// 将嵌套工具标记为简单（紧凑）渲染模式
	if s, ok := tool.(Compactable); ok {
		s.SetCompact(true)
	}
	a.nestedTools = append(a.nestedTools, tool)
	a.clearCache()
}

// AgentToolRenderContext 负责渲染智能体工具消息。
// 它实现了 ToolRenderer 接口，提供自定义的渲染逻辑。
type AgentToolRenderContext struct {
	agent *AgentToolMessageItem // 关联的智能体工具消息项
}

// RenderTool 实现 [ToolRenderer] 接口，渲染智能体工具消息。
// 该方法构建完整的工具消息视图，包括头部、提示文本、嵌套工具树和执行结果。
// 参数：
//   - sty: 样式配置对象
//   - width: 可用渲染宽度
//   - opts: 工具渲染选项，包含状态、动画等信息
//
// 返回值：渲染后的字符串表示
func (r *AgentToolRenderContext) RenderTool(sty *styles.Styles, width int, opts *ToolRenderOpts) string {
	cappedWidth := cappedMessageWidth(width)
	// 如果工具调用未完成、未取消且没有嵌套工具，显示等待状态
	if !opts.ToolCall.Finished && !opts.IsCanceled() && len(r.agent.nestedTools) == 0 {
		return pendingTool(sty, "Agent", opts.Anim)
	}

	// 解析智能体工具参数
	var params agent.AgentParams
	_ = json.Unmarshal([]byte(opts.ToolCall.Input), &params)

	// 处理提示文本，将换行符替换为空格
	prompt := params.Prompt
	prompt = strings.ReplaceAll(prompt, "\n", " ")

	// 构建工具头部
	header := toolHeader(sty, opts.Status, "Agent", cappedWidth, opts.Compact)
	if opts.Compact {
		return header
	}

	// 构建任务标签和提示文本
	taskTag := sty.Tool.AgentTaskTag.Render("Task")
	taskTagWidth := lipgloss.Width(taskTag)

	// 计算提示文本的剩余可用宽度
	remainingWidth := min(cappedWidth-taskTagWidth-3, maxTextWidth-taskTagWidth-3) // -3 用于间距

	promptText := sty.Tool.AgentPrompt.Width(remainingWidth).Render(prompt)

	// 组合头部和提示文本
	header = lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			taskTag,
			" ",
			promptText,
		),
	)

	// 构建嵌套工具调用的树形结构
	childTools := tree.Root(header)

	// 将每个嵌套工具添加为树的子节点
	for _, nestedTool := range r.agent.nestedTools {
		childView := nestedTool.Render(remainingWidth)
		childTools.Child(childView)
	}

	// 构建输出内容的各个部分
	var parts []string
	parts = append(parts, childTools.Enumerator(roundedEnumerator(2, taskTagWidth-5)).String())

	// 如果工具仍在运行，显示动画
	if !opts.HasResult() && !opts.IsCanceled() {
		parts = append(parts, "", opts.Anim.Render())
	}

	result := lipgloss.JoinVertical(lipgloss.Left, parts...)

	// 完成时添加主体内容
	if opts.HasResult() && opts.Result.Content != "" {
		body := toolOutputMarkdownContent(sty, opts.Result.Content, cappedWidth-toolBodyLeftPaddingTotal, opts.ExpandedContent)
		return joinToolParts(result, body)
	}

	return result
}

// -----------------------------------------------------------------------------
// 智能体获取工具 (Agentic Fetch Tool)
// -----------------------------------------------------------------------------

// AgenticFetchToolMessageItem 是表示智能体获取工具调用的消息项。
// 该工具支持通过智能体方式获取网络资源，并可以包含嵌套的工具调用。
type AgenticFetchToolMessageItem struct {
	*baseToolMessageItem

	nestedTools []ToolMessageItem // 嵌套的工具项列表
}

var (
	_ ToolMessageItem     = (*AgenticFetchToolMessageItem)(nil) // 确保实现 ToolMessageItem 接口
	_ NestedToolContainer = (*AgenticFetchToolMessageItem)(nil) // 确保实现 NestedToolContainer 接口
)

// NewAgenticFetchToolMessageItem 创建一个新的 [AgenticFetchToolMessageItem] 实例。
// 参数说明：
//   - sty: 样式配置对象，用于控制消息项的视觉呈现
//   - toolCall: 工具调用信息，包含URL和提示参数
//   - result: 工具执行结果，可能为 nil（表示工具仍在执行中）
//   - canceled: 标识工具调用是否已被取消
//
// 返回值：初始化后的 AgenticFetchToolMessageItem 实例
func NewAgenticFetchToolMessageItem(
	sty *styles.Styles,
	toolCall message.ToolCall,
	result *message.ToolResult,
	canceled bool,
) *AgenticFetchToolMessageItem {
	t := &AgenticFetchToolMessageItem{}
	t.baseToolMessageItem = newBaseToolMessageItem(sty, toolCall, result, &AgenticFetchToolRenderContext{fetch: t}, canceled)
	// 对于智能体获取工具，我们保持旋转动画直到工具调用完成
	t.spinningFunc = func(state SpinningState) bool {
		return !state.HasResult() && !state.IsCanceled()
	}
	return t
}

// NestedTools 返回嵌套的工具项列表。
// 这些嵌套工具代表了智能体获取过程中调用的子工具。
func (a *AgenticFetchToolMessageItem) NestedTools() []ToolMessageItem {
	return a.nestedTools
}

// SetNestedTools 设置嵌套的工具项列表。
// 设置后会清除缓存以触发重新渲染。
func (a *AgenticFetchToolMessageItem) SetNestedTools(tools []ToolMessageItem) {
	a.nestedTools = tools
	a.clearCache()
}

// AddNestedTool 添加单个嵌套工具项。
// 嵌套工具会被标记为紧凑渲染模式以优化显示空间。
func (a *AgenticFetchToolMessageItem) AddNestedTool(tool ToolMessageItem) {
	// 将嵌套工具标记为简单（紧凑）渲染模式
	if s, ok := tool.(Compactable); ok {
		s.SetCompact(true)
	}
	a.nestedTools = append(a.nestedTools, tool)
	a.clearCache()
}

// AgenticFetchToolRenderContext 负责渲染智能体获取工具消息。
// 它实现了 ToolRenderer 接口，提供自定义的渲染逻辑。
type AgenticFetchToolRenderContext struct {
	fetch *AgenticFetchToolMessageItem // 关联的智能体获取工具消息项
}

// agenticFetchParams 定义智能体获取工具的参数结构。
// 该结构体与 tools.AgenticFetchParams 保持一致。
type agenticFetchParams struct {
	URL    string `json:"url,omitempty"` // 要获取的URL地址（可选）
	Prompt string `json:"prompt"`        // 处理获取内容的提示文本
}

// RenderTool 实现 [ToolRenderer] 接口，渲染智能体获取工具消息。
// 该方法构建完整的工具消息视图，包括头部、URL、提示文本、嵌套工具树和执行结果。
// 参数：
//   - sty: 样式配置对象
//   - width: 可用渲染宽度
//   - opts: 工具渲染选项，包含状态、动画等信息
//
// 返回值：渲染后的字符串表示
func (r *AgenticFetchToolRenderContext) RenderTool(sty *styles.Styles, width int, opts *ToolRenderOpts) string {
	cappedWidth := cappedMessageWidth(width)
	// 如果工具调用未完成、未取消且没有嵌套工具，显示等待状态
	if !opts.ToolCall.Finished && !opts.IsCanceled() && len(r.fetch.nestedTools) == 0 {
		return pendingTool(sty, "Agentic Fetch", opts.Anim)
	}

	// 解析智能体获取工具参数
	var params agenticFetchParams
	_ = json.Unmarshal([]byte(opts.ToolCall.Input), &params)

	// 处理提示文本，将换行符替换为空格
	prompt := params.Prompt
	prompt = strings.ReplaceAll(prompt, "\n", " ")

	// 构建头部，包含可选的URL参数
	var toolParams []string
	if params.URL != "" {
		toolParams = append(toolParams, params.URL)
	}

	header := toolHeader(sty, opts.Status, "Agentic Fetch", cappedWidth, opts.Compact, toolParams...)
	if opts.Compact {
		return header
	}

	// 构建提示标签
	promptTag := sty.Tool.AgenticFetchPromptTag.Render("Prompt")
	promptTagWidth := lipgloss.Width(promptTag)

	// 计算提示文本的剩余可用宽度
	remainingWidth := min(cappedWidth-promptTagWidth-3, maxTextWidth-promptTagWidth-3) // -3 用于间距

	promptText := sty.Tool.AgentPrompt.Width(remainingWidth).Render(prompt)

	// 组合头部和提示文本
	header = lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			promptTag,
			" ",
			promptText,
		),
	)

	// 构建嵌套工具调用的树形结构
	childTools := tree.Root(header)

	// 将每个嵌套工具添加为树的子节点
	for _, nestedTool := range r.fetch.nestedTools {
		childView := nestedTool.Render(remainingWidth)
		childTools.Child(childView)
	}

	// 构建输出内容的各个部分
	var parts []string
	parts = append(parts, childTools.Enumerator(roundedEnumerator(2, promptTagWidth-5)).String())

	// 如果工具仍在运行，显示动画
	if !opts.HasResult() && !opts.IsCanceled() {
		parts = append(parts, "", opts.Anim.Render())
	}

	result := lipgloss.JoinVertical(lipgloss.Left, parts...)

	// 完成时添加主体内容
	if opts.HasResult() && opts.Result.Content != "" {
		body := toolOutputMarkdownContent(sty, opts.Result.Content, cappedWidth-toolBodyLeftPaddingTotal, opts.ExpandedContent)
		return joinToolParts(result, body)
	}

	return result
}
