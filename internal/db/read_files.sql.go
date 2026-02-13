// 由 sqlc 生成的代码。请勿编辑。
// 版本：
//   sqlc v1.30.0
// 源文件：read_files.sql

package db

import (
	"context"
)

// getFileRead - 获取文件读取记录的SQL查询语句
// name: GetFileRead :one - 返回单条记录
const getFileRead = `-- name: GetFileRead :one
SELECT session_id, path, read_at FROM read_files
WHERE session_id = ? AND path = ? LIMIT 1
`

// GetFileReadParams - 获取文件读取记录的参数结构体
type GetFileReadParams struct {
	SessionID string `json:"session_id"` // 会话ID
	Path      string `json:"path"`       // 文件路径
}

// GetFileRead - 根据会话ID和文件路径获取文件读取记录
// 参数：
//   - ctx: 上下文
//   - arg: 包含会话ID和文件路径的参数
//
// 返回：
//   - ReadFile: 文件读取记录
//   - error: 错误信息
func (q *Queries) GetFileRead(ctx context.Context, arg GetFileReadParams) (ReadFile, error) {
	row := q.queryRow(ctx, q.getFileReadStmt, getFileRead, arg.SessionID, arg.Path)
	var i ReadFile
	err := row.Scan(
		&i.SessionID,
		&i.Path,
		&i.ReadAt,
	)
	return i, err
}

// recordFileRead - 记录文件读取的SQL语句
// name: RecordFileRead :exec - 执行操作（不返回结果）
const recordFileRead = `-- name: RecordFileRead :exec
INSERT INTO read_files (
    session_id,
    path,
    read_at
) VALUES (
    ?,
    ?,
    strftime('%s', 'now')
) ON CONFLICT(path, session_id) DO UPDATE SET
    read_at = excluded.read_at
`

// RecordFileReadParams - 记录文件读取的参数结构体
type RecordFileReadParams struct {
	SessionID string `json:"session_id"` // 会话ID
	Path      string `json:"path"`       // 文件路径
}

// listSessionReadFiles - 列出会话中已读取文件的SQL查询语句
// name: ListSessionReadFiles :many - 返回多条记录
const listSessionReadFiles = `-- name: ListSessionReadFiles :many
SELECT session_id, path, read_at FROM read_files
WHERE session_id = ?
ORDER BY read_at DESC
`

// ListSessionReadFiles - 列出指定会话中所有已读取的文件
// 参数：
//   - ctx: 上下文
//   - sessionID: 会话ID
//
// 返回：
//   - []ReadFile: 文件读取记录列表
//   - error: 错误信息
func (q *Queries) ListSessionReadFiles(ctx context.Context, sessionID string) ([]ReadFile, error) {
	rows, err := q.query(ctx, q.listSessionReadFilesStmt, listSessionReadFiles, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ReadFile{}
	for rows.Next() {
		var i ReadFile
		if err := rows.Scan(
			&i.SessionID,
			&i.Path,
			&i.ReadAt,
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

// RecordFileRead - 记录文件读取操作
// 如果记录已存在（基于path和session_id的唯一约束），则更新read_at时间戳
// 参数：
//   - ctx: 上下文
//   - arg: 包含会话ID和文件路径的参数
//
// 返回：
//   - error: 错误信息
func (q *Queries) RecordFileRead(ctx context.Context, arg RecordFileReadParams) error {
	_, err := q.exec(ctx, q.recordFileReadStmt, recordFileRead,
		arg.SessionID,
		arg.Path,
	)
	return err
}
