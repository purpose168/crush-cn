// 由 sqlc 自动生成的代码。请勿编辑。
// 版本:
//   sqlc v1.30.0
// 源文件: stats.sql

package db

import (
	"context"
	"database/sql"
)

// getAverageResponseTime 获取平均响应时间的SQL查询
// 计算所有助手角色消息的平均响应时间（完成时间 - 创建时间）
const getAverageResponseTime = `-- name: GetAverageResponseTime :one
SELECT
    CAST(COALESCE(AVG(finished_at - created_at), 0) AS INTEGER) as avg_response_seconds
FROM messages
WHERE role = 'assistant'
  AND finished_at IS NOT NULL
  AND finished_at > created_at
`

// GetAverageResponseTime 获取平均响应时间
// 返回助手消息的平均响应时间（秒）
func (q *Queries) GetAverageResponseTime(ctx context.Context) (int64, error) {
	row := q.queryRow(ctx, q.getAverageResponseTimeStmt, getAverageResponseTime)
	var avg_response_seconds int64
	err := row.Scan(&avg_response_seconds)
	return avg_response_seconds, err
}

// getHourDayHeatmap 获取小时-天热力图数据的SQL查询
// 按星期几和小时分组统计会话数量
const getHourDayHeatmap = `-- name: GetHourDayHeatmap :many
SELECT
    CAST(strftime('%w', created_at, 'unixepoch') AS INTEGER) as day_of_week,
    CAST(strftime('%H', created_at, 'unixepoch') AS INTEGER) as hour,
    COUNT(*) as session_count
FROM sessions
WHERE parent_session_id IS NULL
GROUP BY day_of_week, hour
ORDER BY day_of_week, hour
`

// GetHourDayHeatmapRow 小时-天热力图查询结果行
type GetHourDayHeatmapRow struct {
	DayOfWeek    int64 `json:"day_of_week"`   // 星期几（0=周日，1=周一，...，6=周六）
	Hour         int64 `json:"hour"`          // 小时（0-23）
	SessionCount int64 `json:"session_count"` // 会话数量
}

// GetHourDayHeatmap 获取小时-天热力图数据
// 返回按星期几和小时分组的会话统计
func (q *Queries) GetHourDayHeatmap(ctx context.Context) ([]GetHourDayHeatmapRow, error) {
	rows, err := q.query(ctx, q.getHourDayHeatmapStmt, getHourDayHeatmap)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetHourDayHeatmapRow{}
	for rows.Next() {
		var i GetHourDayHeatmapRow
		if err := rows.Scan(&i.DayOfWeek, &i.Hour, &i.SessionCount); err != nil {
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

// getRecentActivity 获取最近活动统计的SQL查询
// 统计最近30天内每天的会话数、总令牌数和成本
const getRecentActivity = `-- name: GetRecentActivity :many
SELECT
    date(created_at, 'unixepoch') as day,
    COUNT(*) as session_count,
    SUM(prompt_tokens + completion_tokens) as total_tokens,
    SUM(cost) as cost
FROM sessions
WHERE parent_session_id IS NULL
  AND created_at >= strftime('%s', 'now', '-30 days')
GROUP BY date(created_at, 'unixepoch')
ORDER BY day ASC
`

// GetRecentActivityRow 最近活动统计查询结果行
type GetRecentActivityRow struct {
	Day          interface{}     `json:"day"`           // 日期
	SessionCount int64           `json:"session_count"` // 会话数量
	TotalTokens  sql.NullFloat64 `json:"total_tokens"`  // 总令牌数
	Cost         sql.NullFloat64 `json:"cost"`          // 成本
}

// GetRecentActivity 获取最近活动统计
// 返回最近30天内每天的活动数据
func (q *Queries) GetRecentActivity(ctx context.Context) ([]GetRecentActivityRow, error) {
	rows, err := q.query(ctx, q.getRecentActivityStmt, getRecentActivity)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetRecentActivityRow{}
	for rows.Next() {
		var i GetRecentActivityRow
		if err := rows.Scan(
			&i.Day,
			&i.SessionCount,
			&i.TotalTokens,
			&i.Cost,
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

// getToolUsage 获取工具使用统计的SQL查询
// 统计各个工具的调用次数
const getToolUsage = `-- name: GetToolUsage :many
SELECT
    json_extract(value, '$.data.name') as tool_name,
    COUNT(*) as call_count
FROM messages, json_each(parts)
WHERE json_extract(value, '$.type') = 'tool_call'
  AND json_extract(value, '$.data.name') IS NOT NULL
GROUP BY tool_name
ORDER BY call_count DESC
`

// GetToolUsageRow 工具使用统计查询结果行
type GetToolUsageRow struct {
	ToolName  interface{} `json:"tool_name"`  // 工具名称
	CallCount int64       `json:"call_count"` // 调用次数
}

// GetToolUsage 获取工具使用统计
// 返回各工具的调用次数统计
func (q *Queries) GetToolUsage(ctx context.Context) ([]GetToolUsageRow, error) {
	rows, err := q.query(ctx, q.getToolUsageStmt, getToolUsage)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetToolUsageRow{}
	for rows.Next() {
		var i GetToolUsageRow
		if err := rows.Scan(&i.ToolName, &i.CallCount); err != nil {
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

// getTotalStats 获取总体统计数据的SQL查询
// 统计总会话数、总令牌数、总成本、总消息数及平均值
const getTotalStats = `-- name: GetTotalStats :one
SELECT
    COUNT(*) as total_sessions,
    COALESCE(SUM(prompt_tokens), 0) as total_prompt_tokens,
    COALESCE(SUM(completion_tokens), 0) as total_completion_tokens,
    COALESCE(SUM(cost), 0) as total_cost,
    COALESCE(SUM(message_count), 0) as total_messages,
    COALESCE(AVG(prompt_tokens + completion_tokens), 0) as avg_tokens_per_session,
    COALESCE(AVG(message_count), 0) as avg_messages_per_session
FROM sessions
WHERE parent_session_id IS NULL
`

// GetTotalStatsRow 总体统计查询结果行
type GetTotalStatsRow struct {
	TotalSessions         int64       `json:"total_sessions"`           // 总会话数
	TotalPromptTokens     interface{} `json:"total_prompt_tokens"`      // 总提示令牌数
	TotalCompletionTokens interface{} `json:"total_completion_tokens"`  // 总补全令牌数
	TotalCost             interface{} `json:"total_cost"`               // 总成本
	TotalMessages         interface{} `json:"total_messages"`           // 总消息数
	AvgTokensPerSession   interface{} `json:"avg_tokens_per_session"`   // 每会话平均令牌数
	AvgMessagesPerSession interface{} `json:"avg_messages_per_session"` // 每会话平均消息数
}

// GetTotalStats 获取总体统计数据
// 返回会话、令牌、成本等汇总统计信息
func (q *Queries) GetTotalStats(ctx context.Context) (GetTotalStatsRow, error) {
	row := q.queryRow(ctx, q.getTotalStatsStmt, getTotalStats)
	var i GetTotalStatsRow
	err := row.Scan(
		&i.TotalSessions,
		&i.TotalPromptTokens,
		&i.TotalCompletionTokens,
		&i.TotalCost,
		&i.TotalMessages,
		&i.AvgTokensPerSession,
		&i.AvgMessagesPerSession,
	)
	return i, err
}

// getUsageByDay 获取每日使用统计的SQL查询
// 按天统计令牌使用量和成本
const getUsageByDay = `-- name: GetUsageByDay :many
SELECT
    date(created_at, 'unixepoch') as day,
    SUM(prompt_tokens) as prompt_tokens,
    SUM(completion_tokens) as completion_tokens,
    SUM(cost) as cost,
    COUNT(*) as session_count
FROM sessions
WHERE parent_session_id IS NULL
GROUP BY date(created_at, 'unixepoch')
ORDER BY day DESC
`

// GetUsageByDayRow 每日使用统计查询结果行
type GetUsageByDayRow struct {
	Day              interface{}     `json:"day"`               // 日期
	PromptTokens     sql.NullFloat64 `json:"prompt_tokens"`     // 提示令牌数
	CompletionTokens sql.NullFloat64 `json:"completion_tokens"` // 补全令牌数
	Cost             sql.NullFloat64 `json:"cost"`              // 成本
	SessionCount     int64           `json:"session_count"`     // 会话数量
}

// GetUsageByDay 获取每日使用统计
// 返回每天的令牌使用量和成本统计
func (q *Queries) GetUsageByDay(ctx context.Context) ([]GetUsageByDayRow, error) {
	rows, err := q.query(ctx, q.getUsageByDayStmt, getUsageByDay)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetUsageByDayRow{}
	for rows.Next() {
		var i GetUsageByDayRow
		if err := rows.Scan(
			&i.Day,
			&i.PromptTokens,
			&i.CompletionTokens,
			&i.Cost,
			&i.SessionCount,
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

// getUsageByDayOfWeek 获取按星期几统计使用情况的SQL查询
// 统计每周各天的会话数和令牌使用量
const getUsageByDayOfWeek = `-- name: GetUsageByDayOfWeek :many
SELECT
    CAST(strftime('%w', created_at, 'unixepoch') AS INTEGER) as day_of_week,
    COUNT(*) as session_count,
    SUM(prompt_tokens) as prompt_tokens,
    SUM(completion_tokens) as completion_tokens
FROM sessions
WHERE parent_session_id IS NULL
GROUP BY day_of_week
ORDER BY day_of_week
`

// GetUsageByDayOfWeekRow 按星期几统计查询结果行
type GetUsageByDayOfWeekRow struct {
	DayOfWeek        int64           `json:"day_of_week"`       // 星期几（0=周日，1=周一，...，6=周六）
	SessionCount     int64           `json:"session_count"`     // 会话数量
	PromptTokens     sql.NullFloat64 `json:"prompt_tokens"`     // 提示令牌数
	CompletionTokens sql.NullFloat64 `json:"completion_tokens"` // 补全令牌数
}

// GetUsageByDayOfWeek 获取按星期几统计的使用情况
// 返回每周各天的会话和令牌统计
func (q *Queries) GetUsageByDayOfWeek(ctx context.Context) ([]GetUsageByDayOfWeekRow, error) {
	rows, err := q.query(ctx, q.getUsageByDayOfWeekStmt, getUsageByDayOfWeek)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetUsageByDayOfWeekRow{}
	for rows.Next() {
		var i GetUsageByDayOfWeekRow
		if err := rows.Scan(
			&i.DayOfWeek,
			&i.SessionCount,
			&i.PromptTokens,
			&i.CompletionTokens,
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

// getUsageByHour 获取按小时统计使用情况的SQL查询
// 统计每小时的会话数量
const getUsageByHour = `-- name: GetUsageByHour :many
SELECT
    CAST(strftime('%H', created_at, 'unixepoch') AS INTEGER) as hour,
    COUNT(*) as session_count
FROM sessions
WHERE parent_session_id IS NULL
GROUP BY hour
ORDER BY hour
`

// GetUsageByHourRow 按小时统计查询结果行
type GetUsageByHourRow struct {
	Hour         int64 `json:"hour"`          // 小时（0-23）
	SessionCount int64 `json:"session_count"` // 会话数量
}

// GetUsageByHour 获取按小时统计的使用情况
// 返回每小时的会话统计
func (q *Queries) GetUsageByHour(ctx context.Context) ([]GetUsageByHourRow, error) {
	rows, err := q.query(ctx, q.getUsageByHourStmt, getUsageByHour)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetUsageByHourRow{}
	for rows.Next() {
		var i GetUsageByHourRow
		if err := rows.Scan(&i.Hour, &i.SessionCount); err != nil {
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

// getUsageByModel 获取按模型统计使用情况的SQL查询
// 统计各模型和提供商的消息数量
const getUsageByModel = `-- name: GetUsageByModel :many
SELECT
    COALESCE(model, 'unknown') as model,
    COALESCE(provider, 'unknown') as provider,
    COUNT(*) as message_count
FROM messages
WHERE role = 'assistant'
GROUP BY model, provider
ORDER BY message_count DESC
`

// GetUsageByModelRow 按模型统计查询结果行
type GetUsageByModelRow struct {
	Model        string `json:"model"`         // 模型名称
	Provider     string `json:"provider"`      // 提供商
	MessageCount int64  `json:"message_count"` // 消息数量
}

// GetUsageByModel 获取按模型统计的使用情况
// 返回各模型和提供商的消息统计
func (q *Queries) GetUsageByModel(ctx context.Context) ([]GetUsageByModelRow, error) {
	rows, err := q.query(ctx, q.getUsageByModelStmt, getUsageByModel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetUsageByModelRow{}
	for rows.Next() {
		var i GetUsageByModelRow
		if err := rows.Scan(&i.Model, &i.Provider, &i.MessageCount); err != nil {
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
