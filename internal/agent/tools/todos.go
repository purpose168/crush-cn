package tools

import (
	"context"
	_ "embed"
	"fmt"

	"charm.land/fantasy"
	"github.com/purpose168/crush-cn/internal/session"
)

//go:embed todos.md
var todosDescription []byte

const TodosToolName = "todos"

type TodosParams struct {
	Todos []TodoItem `json:"todos" description:"更新后的待办事项列表"`
}

type TodoItem struct {
	Content    string `json:"content" description:"需要完成的任务（命令式形式）"`
	Status     string `json:"status" description:"任务状态：pending（待处理）、in_progress（进行中）或 completed（已完成）"`
	ActiveForm string `json:"active_form" description:"现在进行时形式（例如，'运行测试'）"`
}

// TodosResponseMetadata 待办事项工具的响应元数据
// 包含待办事项列表的状态信息和变更详情

type TodosResponseMetadata struct {
	IsNew         bool           `json:"is_new"`         // 是否是新的待办事项列表
	Todos         []session.Todo `json:"todos"`         // 完整的待办事项列表
	JustCompleted []string       `json:"just_completed,omitempty"` // 本次更新中刚完成的任务
	JustStarted   string         `json:"just_started,omitempty"`   // 本次更新中刚开始的任务
	Completed     int            `json:"completed"`     // 已完成任务数量
	Total         int            `json:"total"`         // 总任务数量
}

// NewTodosTool 创建一个新的待办事项管理工具
// 参数：sessions - 会话服务，用于存储和获取会话数据
// 返回：一个配置好的待办事项管理工具
func NewTodosTool(sessions session.Service) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		TodosToolName,
		string(todosDescription),
		func(ctx context.Context, params TodosParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			// 从上下文获取会话ID
			sessionID := GetSessionFromContext(ctx)
			if sessionID == "" {
				return fantasy.ToolResponse{}, fmt.Errorf("管理待办事项需要会话ID")
			}

			// 获取当前会话
			currentSession, err := sessions.Get(ctx, sessionID)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("获取会话失败: %w", err)
			}

			// 检查是否是新的待办事项列表
			isNew := len(currentSession.Todos) == 0
			// 记录每个待办事项的旧状态
			oldStatusByContent := make(map[string]session.TodoStatus)
			for _, todo := range currentSession.Todos {
				oldStatusByContent[todo.Content] = todo.Status
			}

			// 验证所有待办事项的状态是否有效
			for _, item := range params.Todos {
				switch item.Status {
				case "pending", "in_progress", "completed":
				default:
					return fantasy.ToolResponse{}, fmt.Errorf("待办事项 %q 的状态 %q 无效", item.Content, item.Status)
				}
			}

			// 构建新的待办事项列表
			todos := make([]session.Todo, len(params.Todos))
			var justCompleted []string // 刚完成的任务
			var justStarted string     // 刚开始的任务
			completedCount := 0         // 已完成任务计数

			for i, item := range params.Todos {
				todos[i] = session.Todo{
					Content:    item.Content,
					Status:     session.TodoStatus(item.Status),
					ActiveForm: item.ActiveForm,
				}

				newStatus := session.TodoStatus(item.Status)
				oldStatus, existed := oldStatusByContent[item.Content]

				// 处理已完成的任务
				if newStatus == session.TodoStatusCompleted {
					completedCount++
					if existed && oldStatus != session.TodoStatusCompleted {
						justCompleted = append(justCompleted, item.Content)
					}
				}

				// 处理刚开始的任务
				if newStatus == session.TodoStatusInProgress {
					if !existed || oldStatus != session.TodoStatusInProgress {
						if item.ActiveForm != "" {
							justStarted = item.ActiveForm
						} else {
							justStarted = item.Content
						}
					}
				}
			}

			// 保存更新后的待办事项列表
			currentSession.Todos = todos
			_, err = sessions.Save(ctx, currentSession)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("保存待办事项失败: %w", err)
			}

			// 构建响应消息
			response := "待办事项列表更新成功。\n\n"

			// 计算各状态任务数量
			pendingCount := 0
			inProgressCount := 0

			for _, todo := range todos {
				switch todo.Status {
				case session.TodoStatusPending:
					pendingCount++
				case session.TodoStatusInProgress:
					inProgressCount++
				}
			}

			// 添加状态统计信息
			response += fmt.Sprintf("状态: %d 待处理, %d 进行中, %d 已完成\n",
				pendingCount, inProgressCount, completedCount)

			// 添加后续操作提示
			response += "待办事项已成功修改。请继续使用待办事项列表跟踪您的进度。如果适用，请继续执行当前任务。"

			// 构建响应元数据
			metadata := TodosResponseMetadata{
				IsNew:         isNew,
				Todos:         todos,
				JustCompleted: justCompleted,
				JustStarted:   justStarted,
				Completed:     completedCount,
				Total:         len(todos),
			}

			// 返回带元数据的响应
			return fantasy.WithResponseMetadata(fantasy.NewTextResponse(response), metadata), nil
		})
}
