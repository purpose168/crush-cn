// 本文件由 sqlc 自动生成。请勿手动编辑。
// 版本信息:
//   sqlc v1.30.0

package db

import (
	"context"
)

// Querier 定义了数据库查询接口，包含所有数据库操作方法
type Querier interface {
	// CreateFile 创建新文件记录
	CreateFile(ctx context.Context, arg CreateFileParams) (File, error)
	// CreateMessage 创建新消息记录
	CreateMessage(ctx context.Context, arg CreateMessageParams) (Message, error)
	// CreateSession 创建新会话记录
	CreateSession(ctx context.Context, arg CreateSessionParams) (Session, error)
	// DeleteFile 根据ID删除文件记录
	DeleteFile(ctx context.Context, id string) error
	// DeleteMessage 根据ID删除消息记录
	DeleteMessage(ctx context.Context, id string) error
	// DeleteSession 根据ID删除会话记录
	DeleteSession(ctx context.Context, id string) error
	// DeleteSessionFiles 删除指定会话的所有关联文件
	DeleteSessionFiles(ctx context.Context, sessionID string) error
	// DeleteSessionMessages 删除指定会话的所有关联消息
	DeleteSessionMessages(ctx context.Context, sessionID string) error
	// GetAverageResponseTime 获取平均响应时间（毫秒）
	GetAverageResponseTime(ctx context.Context) (int64, error)
	// GetFile 根据ID获取文件记录
	GetFile(ctx context.Context, id string) (File, error)
	// GetFileByPathAndSession 根据文件路径和会话ID获取文件记录
	GetFileByPathAndSession(ctx context.Context, arg GetFileByPathAndSessionParams) (File, error)
	// GetFileRead 获取文件读取记录
	GetFileRead(ctx context.Context, arg GetFileReadParams) (ReadFile, error)
	// GetHourDayHeatmap 获取小时-日期热力图数据（用于分析使用模式）
	GetHourDayHeatmap(ctx context.Context) ([]GetHourDayHeatmapRow, error)
	// GetMessage 根据ID获取消息记录
	GetMessage(ctx context.Context, id string) (Message, error)
	// GetRecentActivity 获取最近的活动记录
	GetRecentActivity(ctx context.Context) ([]GetRecentActivityRow, error)
	// GetSessionByID 根据ID获取会话记录
	GetSessionByID(ctx context.Context, id string) (Session, error)
	// GetToolUsage 获取工具使用统计
	GetToolUsage(ctx context.Context) ([]GetToolUsageRow, error)
	// GetTotalStats 获取总体统计数据
	GetTotalStats(ctx context.Context) (GetTotalStatsRow, error)
	// GetUsageByDay 获取按日期统计的使用数据
	GetUsageByDay(ctx context.Context) ([]GetUsageByDayRow, error)
	// GetUsageByDayOfWeek 获取按星期统计的使用数据
	GetUsageByDayOfWeek(ctx context.Context) ([]GetUsageByDayOfWeekRow, error)
	// GetUsageByHour 获取按小时统计的使用数据
	GetUsageByHour(ctx context.Context) ([]GetUsageByHourRow, error)
	// GetUsageByModel 获取按模型统计的使用数据
	GetUsageByModel(ctx context.Context) ([]GetUsageByModelRow, error)
	// ListAllUserMessages 列出所有用户消息
	ListAllUserMessages(ctx context.Context) ([]Message, error)
	// ListFilesByPath 根据路径列出文件
	ListFilesByPath(ctx context.Context, path string) ([]File, error)
	// ListFilesBySession 列出指定会话的所有文件
	ListFilesBySession(ctx context.Context, sessionID string) ([]File, error)
	// ListLatestSessionFiles 列出指定会话的最新文件
	ListLatestSessionFiles(ctx context.Context, sessionID string) ([]File, error)
	// ListMessagesBySession 列出指定会话的所有消息
	ListMessagesBySession(ctx context.Context, sessionID string) ([]Message, error)
	// ListNewFiles 列出新创建的文件
	ListNewFiles(ctx context.Context) ([]File, error)
	// ListSessionReadFiles 列出指定会话已读取的文件
	ListSessionReadFiles(ctx context.Context, sessionID string) ([]ReadFile, error)
	// ListSessions 列出所有会话
	ListSessions(ctx context.Context) ([]Session, error)
	// ListUserMessagesBySession 列出指定会话的用户消息
	ListUserMessagesBySession(ctx context.Context, sessionID string) ([]Message, error)
	// RecordFileRead 记录文件读取操作
	RecordFileRead(ctx context.Context, arg RecordFileReadParams) error
	// UpdateMessage 更新消息记录
	UpdateMessage(ctx context.Context, arg UpdateMessageParams) error
	// UpdateSession 更新会话记录
	UpdateSession(ctx context.Context, arg UpdateSessionParams) (Session, error)
	// UpdateSessionTitleAndUsage 更新会话标题和使用统计
	UpdateSessionTitleAndUsage(ctx context.Context, arg UpdateSessionTitleAndUsageParams) error
}

// 确保 Queries 类型实现了 Querier 接口
var _ Querier = (*Queries)(nil)
