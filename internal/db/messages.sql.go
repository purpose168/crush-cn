// 由 sqlc 自动生成的代码。请勿编辑。
// 版本：
//   sqlc v1.30.0
// 源文件：messages.sql

package db

import (
	"context"
	"database/sql"
)

const createMessage = `-- 名称: CreateMessage :one
-- 功能: 创建新消息记录
-- 说明: 向 messages 表插入一条新消息，并返回完整的消息信息
INSERT INTO messages (
    id,
    session_id,
    role,
    parts,
    model,
    provider,
    is_summary_message,
    created_at,
    updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, strftime('%s', 'now'), strftime('%s', 'now')
)
RETURNING id, session_id, role, parts, model, created_at, updated_at, finished_at, provider, is_summary_message
`

// CreateMessageParams 创建消息的参数结构体
// 包含创建新消息所需的所有字段
type CreateMessageParams struct {
	ID               string         `json:"id"`                 // 消息唯一标识符
	SessionID        string         `json:"session_id"`         // 会话唯一标识符
	Role             string         `json:"role"`               // 消息角色（如：user、assistant、system）
	Parts            string         `json:"parts"`              // 消息内容部分（JSON 格式）
	Model            sql.NullString `json:"model"`              // 使用的模型名称（可为空）
	Provider         sql.NullString `json:"provider"`           // 服务提供商（可为空）
	IsSummaryMessage int64          `json:"is_summary_message"` // 是否为摘要消息（0: 否, 1: 是）
}

// CreateMessage 创建新消息
// 参数: ctx - 上下文对象，arg - 创建消息所需的参数
// 返回: 创建成功的 Message 对象和可能的错误
func (q *Queries) CreateMessage(ctx context.Context, arg CreateMessageParams) (Message, error) {
	row := q.queryRow(ctx, q.createMessageStmt, createMessage,
		arg.ID,
		arg.SessionID,
		arg.Role,
		arg.Parts,
		arg.Model,
		arg.Provider,
		arg.IsSummaryMessage,
	)
	var i Message
	err := row.Scan(
		&i.ID,
		&i.SessionID,
		&i.Role,
		&i.Parts,
		&i.Model,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.FinishedAt,
		&i.Provider,
		&i.IsSummaryMessage,
	)
	return i, err
}

const deleteMessage = `-- 名称: DeleteMessage :exec
-- 功能: 根据消息 ID 删除消息
-- 说明: 从 messages 表中删除指定 ID 的消息记录
DELETE FROM messages
WHERE id = ?
`

// DeleteMessage 根据消息 ID 删除消息
// 参数: ctx - 上下文对象，id - 要删除的消息 ID
// 返回: 可能的错误
func (q *Queries) DeleteMessage(ctx context.Context, id string) error {
	_, err := q.exec(ctx, q.deleteMessageStmt, deleteMessage, id)
	return err
}

const deleteSessionMessages = `-- 名称: DeleteSessionMessages :exec
-- 功能: 删除指定会话的所有消息
-- 说明: 从 messages 表中删除指定 session_id 的所有消息记录
DELETE FROM messages
WHERE session_id = ?
`

// DeleteSessionMessages 删除指定会话的所有消息
// 参数: ctx - 上下文对象，sessionID - 要删除消息的会话 ID
// 返回: 可能的错误
func (q *Queries) DeleteSessionMessages(ctx context.Context, sessionID string) error {
	_, err := q.exec(ctx, q.deleteSessionMessagesStmt, deleteSessionMessages, sessionID)
	return err
}

const getMessage = `-- 名称: GetMessage :one
-- 功能: 根据消息 ID 获取消息
-- 说明: 从 messages 表中查询指定 ID 的消息记录
SELECT id, session_id, role, parts, model, created_at, updated_at, finished_at, provider, is_summary_message
FROM messages
WHERE id = ? LIMIT 1
`

// GetMessage 根据消息 ID 获取消息
// 参数: ctx - 上下文对象，id - 要查询的消息 ID
// 返回: 查询到的 Message 对象和可能的错误
func (q *Queries) GetMessage(ctx context.Context, id string) (Message, error) {
	row := q.queryRow(ctx, q.getMessageStmt, getMessage, id)
	var i Message
	err := row.Scan(
		&i.ID,
		&i.SessionID,
		&i.Role,
		&i.Parts,
		&i.Model,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.FinishedAt,
		&i.Provider,
		&i.IsSummaryMessage,
	)
	return i, err
}

const listAllUserMessages = `-- 名称: ListAllUserMessages :many
-- 功能: 获取所有用户消息
-- 说明: 从 messages 表中查询所有角色为 'user' 的消息，按创建时间倒序排列
SELECT id, session_id, role, parts, model, created_at, updated_at, finished_at, provider, is_summary_message
FROM messages
WHERE role = 'user'
ORDER BY created_at DESC
`

// ListAllUserMessages 获取所有用户消息
// 参数: ctx - 上下文对象
// 返回: 所有用户消息的 Message 对象切片和可能的错误
func (q *Queries) ListAllUserMessages(ctx context.Context) ([]Message, error) {
	rows, err := q.query(ctx, q.listAllUserMessagesStmt, listAllUserMessages)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Message{}
	for rows.Next() {
		var i Message
		if err := rows.Scan(
			&i.ID,
			&i.SessionID,
			&i.Role,
			&i.Parts,
			&i.Model,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.FinishedAt,
			&i.Provider,
			&i.IsSummaryMessage,
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

const listMessagesBySession = `-- 名称: ListMessagesBySession :many
-- 功能: 获取指定会话的所有消息
-- 说明: 从 messages 表中查询指定 session_id 的所有消息，按创建时间正序排列
SELECT id, session_id, role, parts, model, created_at, updated_at, finished_at, provider, is_summary_message
FROM messages
WHERE session_id = ?
ORDER BY created_at ASC
`

// ListMessagesBySession 获取指定会话的所有消息
// 参数: ctx - 上下文对象，sessionID - 要查询的会话 ID
// 返回: 该会话的所有消息的 Message 对象切片和可能的错误
func (q *Queries) ListMessagesBySession(ctx context.Context, sessionID string) ([]Message, error) {
	rows, err := q.query(ctx, q.listMessagesBySessionStmt, listMessagesBySession, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Message{}
	for rows.Next() {
		var i Message
		if err := rows.Scan(
			&i.ID,
			&i.SessionID,
			&i.Role,
			&i.Parts,
			&i.Model,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.FinishedAt,
			&i.Provider,
			&i.IsSummaryMessage,
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

const listUserMessagesBySession = `-- 名称: ListUserMessagesBySession :many
-- 功能: 获取指定会话的所有用户消息
-- 说明: 从 messages 表中查询指定 session_id 且角色为 'user' 的消息，按创建时间倒序排列
SELECT id, session_id, role, parts, model, created_at, updated_at, finished_at, provider, is_summary_message
FROM messages
WHERE session_id = ? AND role = 'user'
ORDER BY created_at DESC
`

// ListUserMessagesBySession 获取指定会话的所有用户消息
// 参数: ctx - 上下文对象，sessionID - 要查询的会话 ID
// 返回: 该会话的所有用户消息的 Message 对象切片和可能的错误
func (q *Queries) ListUserMessagesBySession(ctx context.Context, sessionID string) ([]Message, error) {
	rows, err := q.query(ctx, q.listUserMessagesBySessionStmt, listUserMessagesBySession, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Message{}
	for rows.Next() {
		var i Message
		if err := rows.Scan(
			&i.ID,
			&i.SessionID,
			&i.Role,
			&i.Parts,
			&i.Model,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.FinishedAt,
			&i.Provider,
			&i.IsSummaryMessage,
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

const updateMessage = `-- 名称: UpdateMessage :exec
-- 功能: 更新消息内容
-- 说明: 更新指定 ID 消息的 parts（内容）、finished_at（完成时间）和 updated_at（更新时间）
UPDATE messages
SET
    parts = ?,
    finished_at = ?,
    updated_at = strftime('%s', 'now')
WHERE id = ?
`

// UpdateMessageParams 更新消息的参数结构体
// 包含更新消息所需的字段
type UpdateMessageParams struct {
	Parts      string        `json:"parts"`       // 消息内容部分（JSON 格式）
	FinishedAt sql.NullInt64 `json:"finished_at"` // 消息完成时间（Unix 时间戳，可为空）
	ID         string        `json:"id"`          // 要更新的消息 ID
}

// UpdateMessage 更新消息内容
// 参数: ctx - 上下文对象，arg - 更新消息所需的参数
// 返回: 可能的错误
func (q *Queries) UpdateMessage(ctx context.Context, arg UpdateMessageParams) error {
	_, err := q.exec(ctx, q.updateMessageStmt, updateMessage, arg.Parts, arg.FinishedAt, arg.ID)
	return err
}
