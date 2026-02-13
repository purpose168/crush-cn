// 由 sqlc 自动生成的代码。请勿手动编辑。
// 版本信息:
//   sqlc v1.30.0

package db

import (
	"database/sql"
)

// File 表示文件记录的结构体
// 用于存储会话中的文件信息，包括文件路径、内容和版本等
type File struct {
	ID        string `json:"id"`         // 文件唯一标识符
	SessionID string `json:"session_id"` // 所属会话的ID
	Path      string `json:"path"`       // 文件路径
	Content   string `json:"content"`    // 文件内容
	Version   int64  `json:"version"`    // 文件版本号
	CreatedAt int64  `json:"created_at"` // 创建时间戳（Unix时间戳）
	UpdatedAt int64  `json:"updated_at"` // 更新时间戳（Unix时间戳）
}

// Message 表示消息记录的结构体
// 用于存储会话中的消息信息，包括角色、内容、模型等
type Message struct {
	ID               string         `json:"id"`                 // 消息唯一标识符
	SessionID        string         `json:"session_id"`         // 所属会话的ID
	Role             string         `json:"role"`               // 消息角色（如user、assistant、system等）
	Parts            string         `json:"parts"`              // 消息内容部分（JSON格式）
	Model            sql.NullString `json:"model"`              // 使用的模型名称
	CreatedAt        int64          `json:"created_at"`         // 创建时间戳（Unix时间戳）
	UpdatedAt        int64          `json:"updated_at"`         // 更新时间戳（Unix时间戳）
	FinishedAt       sql.NullInt64  `json:"finished_at"`        // 消息完成时间戳（Unix时间戳）
	Provider         sql.NullString `json:"provider"`           // 服务提供商
	IsSummaryMessage int64          `json:"is_summary_message"` // 是否为摘要消息（0：否，1：是）
}

// ReadFile 表示文件读取记录的结构体
// 用于跟踪会话中文件的读取情况
type ReadFile struct {
	SessionID string `json:"session_id"` // 所属会话的ID
	Path      string `json:"path"`       // 文件路径
	ReadAt    int64  `json:"read_at"`    // 文件最后读取时间（Unix时间戳）
}

// Session 表示会话记录的结构体
// 用于存储会话的元信息，包括标题、消息数量、令牌使用量、成本等
type Session struct {
	ID               string         `json:"id"`                 // 会话唯一标识符
	ParentSessionID  sql.NullString `json:"parent_session_id"`  // 父会话的ID（用于会话层级关系）
	Title            string         `json:"title"`              // 会话标题
	MessageCount     int64          `json:"message_count"`      // 消息总数
	PromptTokens     int64          `json:"prompt_tokens"`      // 提示词令牌（Prompt Tokens）使用量
	CompletionTokens int64          `json:"completion_tokens"`  // 完成令牌（Completion Tokens）使用量
	Cost             float64        `json:"cost"`               // 会话总成本
	UpdatedAt        int64          `json:"updated_at"`         // 更新时间戳（Unix时间戳）
	CreatedAt        int64          `json:"created_at"`         // 创建时间戳（Unix时间戳）
	SummaryMessageID sql.NullString `json:"summary_message_id"` // 摘要消息的ID
	Todos            sql.NullString `json:"todos"`              // 待办事项列表（JSON格式）
}
