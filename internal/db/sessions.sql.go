// 由 sqlc 自动生成的代码。请勿编辑。
// 版本信息:
//   sqlc v1.30.0
// 源文件: sessions.sql

package db

import (
	"context"
	"database/sql"
)

const createSession = `-- 名称: CreateSession :one
INSERT INTO sessions (
    id,
    parent_session_id,
    title,
    message_count,
    prompt_tokens,
    completion_tokens,
    cost,
    summary_message_id,
    updated_at,
    created_at
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    null,
    strftime('%s', 'now'),
    strftime('%s', 'now')
) RETURNING id, parent_session_id, title, message_count, prompt_tokens, completion_tokens, cost, updated_at, created_at, summary_message_id, todos
`

// CreateSessionParams 创建会话参数结构体
type CreateSessionParams struct {
	ID               string         `json:"id"`                // 会话ID
	ParentSessionID  sql.NullString `json:"parent_session_id"` // 父会话ID
	Title            string         `json:"title"`             // 会话标题
	MessageCount     int64          `json:"message_count"`     // 消息数量
	PromptTokens     int64          `json:"prompt_tokens"`     // 提示词令牌数
	CompletionTokens int64          `json:"completion_tokens"` // 完成令牌数
	Cost             float64        `json:"cost"`              // 成本
}

// CreateSession 创建新会话
// 参数:
//   - ctx: 上下文
//   - arg: 创建会话参数
//
// 返回:
//   - Session: 创建的会话对象
//   - error: 错误信息
func (q *Queries) CreateSession(ctx context.Context, arg CreateSessionParams) (Session, error) {
	row := q.queryRow(ctx, q.createSessionStmt, createSession,
		arg.ID,
		arg.ParentSessionID,
		arg.Title,
		arg.MessageCount,
		arg.PromptTokens,
		arg.CompletionTokens,
		arg.Cost,
	)
	var i Session
	err := row.Scan(
		&i.ID,
		&i.ParentSessionID,
		&i.Title,
		&i.MessageCount,
		&i.PromptTokens,
		&i.CompletionTokens,
		&i.Cost,
		&i.UpdatedAt,
		&i.CreatedAt,
		&i.SummaryMessageID,
		&i.Todos,
	)
	return i, err
}

const deleteSession = `-- 名称: DeleteSession :exec
DELETE FROM sessions
WHERE id = ?
`

// DeleteSession 删除会话
// 参数:
//   - ctx: 上下文
//   - id: 会话ID
//
// 返回:
//   - error: 错误信息
func (q *Queries) DeleteSession(ctx context.Context, id string) error {
	_, err := q.exec(ctx, q.deleteSessionStmt, deleteSession, id)
	return err
}

const getSessionByID = `-- 名称: GetSessionByID :one
SELECT id, parent_session_id, title, message_count, prompt_tokens, completion_tokens, cost, updated_at, created_at, summary_message_id, todos
FROM sessions
WHERE id = ? LIMIT 1
`

// GetSessionByID 根据ID获取会话
// 参数:
//   - ctx: 上下文
//   - id: 会话ID
//
// 返回:
//   - Session: 会话对象
//   - error: 错误信息
func (q *Queries) GetSessionByID(ctx context.Context, id string) (Session, error) {
	row := q.queryRow(ctx, q.getSessionByIDStmt, getSessionByID, id)
	var i Session
	err := row.Scan(
		&i.ID,
		&i.ParentSessionID,
		&i.Title,
		&i.MessageCount,
		&i.PromptTokens,
		&i.CompletionTokens,
		&i.Cost,
		&i.UpdatedAt,
		&i.CreatedAt,
		&i.SummaryMessageID,
		&i.Todos,
	)
	return i, err
}

const listSessions = `-- 名称: ListSessions :many
SELECT id, parent_session_id, title, message_count, prompt_tokens, completion_tokens, cost, updated_at, created_at, summary_message_id, todos
FROM sessions
WHERE parent_session_id is NULL
ORDER BY updated_at DESC
`

// ListSessions 获取所有根会话列表（不包含子会话）
// 参数:
//   - ctx: 上下文
//
// 返回:
//   - []Session: 会话列表
//   - error: 错误信息
func (q *Queries) ListSessions(ctx context.Context) ([]Session, error) {
	rows, err := q.query(ctx, q.listSessionsStmt, listSessions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Session{}
	for rows.Next() {
		var i Session
		if err := rows.Scan(
			&i.ID,
			&i.ParentSessionID,
			&i.Title,
			&i.MessageCount,
			&i.PromptTokens,
			&i.CompletionTokens,
			&i.Cost,
			&i.UpdatedAt,
			&i.CreatedAt,
			&i.SummaryMessageID,
			&i.Todos,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateSession = `-- 名称: UpdateSession :one
UPDATE sessions
SET
    title = ?,
    prompt_tokens = ?,
    completion_tokens = ?,
    summary_message_id = ?,
    cost = ?,
    todos = ?
WHERE id = ?
RETURNING id, parent_session_id, title, message_count, prompt_tokens, completion_tokens, cost, updated_at, created_at, summary_message_id, todos
`

// UpdateSessionParams 更新会话参数结构体
type UpdateSessionParams struct {
	Title            string         `json:"title"`              // 会话标题
	PromptTokens     int64          `json:"prompt_tokens"`      // 提示词令牌数
	CompletionTokens int64          `json:"completion_tokens"`  // 完成令牌数
	SummaryMessageID sql.NullString `json:"summary_message_id"` // 摘要消息ID
	Cost             float64        `json:"cost"`               // 成本
	Todos            sql.NullString `json:"todos"`              // 待办事项
	ID               string         `json:"id"`                 // 会话ID
}

// UpdateSession 更新会话信息
// 参数:
//   - ctx: 上下文
//   - arg: 更新会话参数
//
// 返回:
//   - Session: 更新后的会话对象
//   - error: 错误信息
func (q *Queries) UpdateSession(ctx context.Context, arg UpdateSessionParams) (Session, error) {
	row := q.queryRow(ctx, q.updateSessionStmt, updateSession,
		arg.Title,
		arg.PromptTokens,
		arg.CompletionTokens,
		arg.SummaryMessageID,
		arg.Cost,
		arg.Todos,
		arg.ID,
	)
	var i Session
	err := row.Scan(
		&i.ID,
		&i.ParentSessionID,
		&i.Title,
		&i.MessageCount,
		&i.PromptTokens,
		&i.CompletionTokens,
		&i.Cost,
		&i.UpdatedAt,
		&i.CreatedAt,
		&i.SummaryMessageID,
		&i.Todos,
	)
	return i, err
}

const updateSessionTitleAndUsage = `-- 名称: UpdateSessionTitleAndUsage :exec
UPDATE sessions
SET
    title = ?,
    prompt_tokens = prompt_tokens + ?,
    completion_tokens = completion_tokens + ?,
    cost = cost + ?
WHERE id = ?
`

// UpdateSessionTitleAndUsageParams 更新会话标题和使用量参数结构体
type UpdateSessionTitleAndUsageParams struct {
	Title            string  `json:"title"`             // 会话标题
	PromptTokens     int64   `json:"prompt_tokens"`     // 提示词令牌增量
	CompletionTokens int64   `json:"completion_tokens"` // 完成令牌增量
	Cost             float64 `json:"cost"`              // 成本增量
	ID               string  `json:"id"`                // 会话ID
}

// UpdateSessionTitleAndUsage 更新会话标题和使用量（增量更新）
// 参数:
//   - ctx: 上下文
//   - arg: 更新会话标题和使用量参数
//
// 返回:
//   - error: 错误信息
func (q *Queries) UpdateSessionTitleAndUsage(ctx context.Context, arg UpdateSessionTitleAndUsageParams) error {
	_, err := q.exec(ctx, q.updateSessionTitleAndUsageStmt, updateSessionTitleAndUsage,
		arg.Title,
		arg.PromptTokens,
		arg.CompletionTokens,
		arg.Cost,
		arg.ID,
	)
	return err
}
