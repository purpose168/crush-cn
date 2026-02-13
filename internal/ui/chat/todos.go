package chat

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/purpose168/crush-cn/internal/agent/tools"
	"github.com/purpose168/crush-cn/internal/message"
	"github.com/purpose168/crush-cn/internal/session"
	"github.com/purpose168/crush-cn/internal/ui/styles"
)

// -----------------------------------------------------------------------------
// 待办事项工具
// -----------------------------------------------------------------------------

// TodosToolMessageItem 表示待办事项工具调用的消息项。
type TodosToolMessageItem struct {
	*baseToolMessageItem
}

var _ ToolMessageItem = (*TodosToolMessageItem)(nil)

// NewTodosToolMessageItem 创建一个新的 [TodosToolMessageItem]。
func NewTodosToolMessageItem(
	sty *styles.Styles,
	toolCall message.ToolCall,
	result *message.ToolResult,
	canceled bool,
) ToolMessageItem {
	return newBaseToolMessageItem(sty, toolCall, result, &TodosToolRenderContext{}, canceled)
}

// TodosToolRenderContext 渲染待办事项工具消息。
type TodosToolRenderContext struct{}

// RenderTool 实现 [ToolRenderer] 接口。
func (t *TodosToolRenderContext) RenderTool(sty *styles.Styles, width int, opts *ToolRenderOpts) string {
	// 计算限制后的消息宽度
	cappedWidth := cappedMessageWidth(width)
	
	// 如果工具调用处于待处理状态，返回待处理工具显示
	if opts.IsPending() {
		return pendingTool(sty, "待办事项", opts.Anim)
	}

	var params tools.TodosParams
	var meta tools.TodosResponseMetadata
	var headerText string
	var body string

	// 解析参数以获取待处理状态（在结果可用之前）
	if err := json.Unmarshal([]byte(opts.ToolCall.Input), &params); err == nil {
		completedCount := 0
		inProgressTask := ""
		
		// 遍历所有待办事项，统计已完成数量和正在进行的任务
		for _, todo := range params.Todos {
			if todo.Status == "completed" {
				completedCount++
			}
			if todo.Status == "in_progress" {
				if todo.ActiveForm != "" {
					inProgressTask = todo.ActiveForm
				} else {
					inProgressTask = todo.Content
				}
			}
		}

		// 从参数生成默认显示（用于待处理状态或无元数据时）
		ratio := sty.Tool.TodoRatio.Render(fmt.Sprintf("%d/%d", completedCount, len(params.Todos)))
		headerText = ratio
		if inProgressTask != "" {
			headerText = fmt.Sprintf("%s · %s", ratio, inProgressTask)
		}

		// 如果有元数据，使用它来提供更丰富的显示
		if opts.HasResult() && opts.Result.Metadata != "" {
			if err := json.Unmarshal([]byte(opts.Result.Metadata), &meta); err == nil {
				if meta.IsNew {
					// 新创建的待办事项列表
					if meta.JustStarted != "" {
						headerText = fmt.Sprintf("创建了 %d 个待办事项，开始第一个", meta.Total)
					} else {
						headerText = fmt.Sprintf("创建了 %d 个待办事项", meta.Total)
					}
					body = FormatTodosList(sty, meta.Todos, styles.ArrowRightIcon, cappedWidth)
				} else {
					// 根据变化构建标题
					hasCompleted := len(meta.JustCompleted) > 0
					hasStarted := meta.JustStarted != ""
					allCompleted := meta.Completed == meta.Total

					ratio := sty.Tool.TodoRatio.Render(fmt.Sprintf("%d/%d", meta.Completed, meta.Total))
					if hasCompleted && hasStarted {
						// 完成了任务并开始下一个
						text := sty.Subtle.Render(fmt.Sprintf(" · 已完成 %d 个，开始下一个", len(meta.JustCompleted)))
						headerText = fmt.Sprintf("%s%s", ratio, text)
					} else if hasCompleted {
						// 仅完成任务
						text := sty.Subtle.Render(fmt.Sprintf(" · 已完成 %d 个", len(meta.JustCompleted)))
						if allCompleted {
							text = sty.Subtle.Render(" · 已全部完成")
						}
						headerText = fmt.Sprintf("%s%s", ratio, text)
					} else if hasStarted {
						// 开始新任务
						headerText = fmt.Sprintf("%s%s", ratio, sty.Subtle.Render(" · 开始任务"))
					} else {
						headerText = ratio
					}

					// 构建详细内容
					if allCompleted {
						// 全部完成时显示所有待办事项，就像创建时一样
						body = FormatTodosList(sty, meta.Todos, styles.ArrowRightIcon, cappedWidth)
					} else if meta.JustStarted != "" {
						// 显示正在进行的任务
						body = sty.Tool.TodoInProgressIcon.Render(styles.ArrowRightIcon+" ") +
							sty.Base.Render(meta.JustStarted)
					}
				}
			}
		}
	}

	// 构建工具标题
	toolParams := []string{headerText}
	header := toolHeader(sty, opts.Status, "待办事项", cappedWidth, opts.Compact, toolParams...)
	if opts.Compact {
		return header
	}

	// 检查是否有早期状态内容
	if earlyState, ok := toolEarlyStateContent(sty, opts, cappedWidth); ok {
		return joinToolParts(header, earlyState)
	}

	if body == "" {
		return header
	}

	return joinToolParts(header, sty.Tool.Body.Render(body))
}

// FormatTodosList 格式化待办事项列表以供显示。
func FormatTodosList(sty *styles.Styles, todos []session.Todo, inProgressIcon string, width int) string {
	if len(todos) == 0 {
		return ""
	}

	// 复制并排序待办事项列表
	sorted := make([]session.Todo, len(todos))
	copy(sorted, todos)
	sortTodos(sorted)

	var lines []string
	for _, todo := range sorted {
		var prefix string
		textStyle := sty.Base

		// 根据状态设置不同的图标和样式
		switch todo.Status {
		case session.TodoStatusCompleted:
			// 已完成状态
			prefix = sty.Tool.TodoCompletedIcon.Render(styles.TodoCompletedIcon) + " "
		case session.TodoStatusInProgress:
			// 进行中状态
			prefix = sty.Tool.TodoInProgressIcon.Render(inProgressIcon + " ")
		default:
			// 待处理状态
			prefix = sty.Tool.TodoPendingIcon.Render(styles.TodoPendingIcon) + " "
		}

		// 如果任务正在进行且有活动形式描述，使用活动形式
		text := todo.Content
		if todo.Status == session.TodoStatusInProgress && todo.ActiveForm != "" {
			text = todo.ActiveForm
		}
		
		// 构建行并截断以适应宽度
		line := prefix + textStyle.Render(text)
		line = ansi.Truncate(line, width, "…")

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// sortTodos 按状态对待办事项排序：已完成、进行中、待处理。
func sortTodos(todos []session.Todo) {
	slices.SortStableFunc(todos, func(a, b session.Todo) int {
		return statusOrder(a.Status) - statusOrder(b.Status)
	})
}

// statusOrder 返回待办事项状态的排序顺序。
func statusOrder(s session.TodoStatus) int {
	switch s {
	case session.TodoStatusCompleted:
		return 0 // 已完成优先级最高
	case session.TodoStatusInProgress:
		return 1 // 进行中次之
	default:
		return 2 // 待处理最后
	}
}
