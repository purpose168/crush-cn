// 由 sqlc 自动生成的代码。请勿编辑。
// 版本信息:
//   sqlc v1.30.0

package db

import (
	"context"
	"database/sql"
	"fmt"
)

// DBTX 定义数据库事务接口，封装了数据库操作的核心方法
type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

// New 创建并返回一个新的 Queries 实例
// 参数 db: 实现了 DBTX 接口的数据库连接对象
// 返回值: 初始化后的 Queries 指针
func New(db DBTX) *Queries {
	return &Queries{db: db}
}

// Prepare 预编译所有 SQL 查询语句并返回 Queries 实例
// 该方法会预先准备所有数据库查询语句，以提高后续查询的性能
// 参数 ctx: 上下文对象，用于控制请求的生命周期
// 参数 db: 实现了 DBTX 接口的数据库连接对象
// 返回值: 初始化后的 Queries 指针和可能的错误
func Prepare(ctx context.Context, db DBTX) (*Queries, error) {
	q := Queries{db: db}
	var err error
	if q.createFileStmt, err = db.PrepareContext(ctx, createFile); err != nil {
		return nil, fmt.Errorf("准备查询 CreateFile 时出错: %w", err)
	}
	if q.createMessageStmt, err = db.PrepareContext(ctx, createMessage); err != nil {
		return nil, fmt.Errorf("准备查询 CreateMessage 时出错: %w", err)
	}
	if q.createSessionStmt, err = db.PrepareContext(ctx, createSession); err != nil {
		return nil, fmt.Errorf("准备查询 CreateSession 时出错: %w", err)
	}
	if q.deleteFileStmt, err = db.PrepareContext(ctx, deleteFile); err != nil {
		return nil, fmt.Errorf("准备查询 DeleteFile 时出错: %w", err)
	}
	if q.deleteMessageStmt, err = db.PrepareContext(ctx, deleteMessage); err != nil {
		return nil, fmt.Errorf("准备查询 DeleteMessage 时出错: %w", err)
	}
	if q.deleteSessionStmt, err = db.PrepareContext(ctx, deleteSession); err != nil {
		return nil, fmt.Errorf("准备查询 DeleteSession 时出错: %w", err)
	}
	if q.deleteSessionFilesStmt, err = db.PrepareContext(ctx, deleteSessionFiles); err != nil {
		return nil, fmt.Errorf("准备查询 DeleteSessionFiles 时出错: %w", err)
	}
	if q.deleteSessionMessagesStmt, err = db.PrepareContext(ctx, deleteSessionMessages); err != nil {
		return nil, fmt.Errorf("准备查询 DeleteSessionMessages 时出错: %w", err)
	}
	if q.getAverageResponseTimeStmt, err = db.PrepareContext(ctx, getAverageResponseTime); err != nil {
		return nil, fmt.Errorf("准备查询 GetAverageResponseTime 时出错: %w", err)
	}
	if q.getFileStmt, err = db.PrepareContext(ctx, getFile); err != nil {
		return nil, fmt.Errorf("准备查询 GetFile 时出错: %w", err)
	}
	if q.getFileByPathAndSessionStmt, err = db.PrepareContext(ctx, getFileByPathAndSession); err != nil {
		return nil, fmt.Errorf("准备查询 GetFileByPathAndSession 时出错: %w", err)
	}
	if q.getFileReadStmt, err = db.PrepareContext(ctx, getFileRead); err != nil {
		return nil, fmt.Errorf("准备查询 GetFileRead 时出错: %w", err)
	}
	if q.getHourDayHeatmapStmt, err = db.PrepareContext(ctx, getHourDayHeatmap); err != nil {
		return nil, fmt.Errorf("准备查询 GetHourDayHeatmap 时出错: %w", err)
	}
	if q.getMessageStmt, err = db.PrepareContext(ctx, getMessage); err != nil {
		return nil, fmt.Errorf("准备查询 GetMessage 时出错: %w", err)
	}
	if q.getRecentActivityStmt, err = db.PrepareContext(ctx, getRecentActivity); err != nil {
		return nil, fmt.Errorf("准备查询 GetRecentActivity 时出错: %w", err)
	}
	if q.getSessionByIDStmt, err = db.PrepareContext(ctx, getSessionByID); err != nil {
		return nil, fmt.Errorf("准备查询 GetSessionByID 时出错: %w", err)
	}
	if q.getToolUsageStmt, err = db.PrepareContext(ctx, getToolUsage); err != nil {
		return nil, fmt.Errorf("准备查询 GetToolUsage 时出错: %w", err)
	}
	if q.getTotalStatsStmt, err = db.PrepareContext(ctx, getTotalStats); err != nil {
		return nil, fmt.Errorf("准备查询 GetTotalStats 时出错: %w", err)
	}
	if q.getUsageByDayStmt, err = db.PrepareContext(ctx, getUsageByDay); err != nil {
		return nil, fmt.Errorf("准备查询 GetUsageByDay 时出错: %w", err)
	}
	if q.getUsageByDayOfWeekStmt, err = db.PrepareContext(ctx, getUsageByDayOfWeek); err != nil {
		return nil, fmt.Errorf("准备查询 GetUsageByDayOfWeek 时出错: %w", err)
	}
	if q.getUsageByHourStmt, err = db.PrepareContext(ctx, getUsageByHour); err != nil {
		return nil, fmt.Errorf("准备查询 GetUsageByHour 时出错: %w", err)
	}
	if q.getUsageByModelStmt, err = db.PrepareContext(ctx, getUsageByModel); err != nil {
		return nil, fmt.Errorf("准备查询 GetUsageByModel 时出错: %w", err)
	}
	if q.listAllUserMessagesStmt, err = db.PrepareContext(ctx, listAllUserMessages); err != nil {
		return nil, fmt.Errorf("准备查询 ListAllUserMessages 时出错: %w", err)
	}
	if q.listFilesByPathStmt, err = db.PrepareContext(ctx, listFilesByPath); err != nil {
		return nil, fmt.Errorf("准备查询 ListFilesByPath 时出错: %w", err)
	}
	if q.listFilesBySessionStmt, err = db.PrepareContext(ctx, listFilesBySession); err != nil {
		return nil, fmt.Errorf("准备查询 ListFilesBySession 时出错: %w", err)
	}
	if q.listLatestSessionFilesStmt, err = db.PrepareContext(ctx, listLatestSessionFiles); err != nil {
		return nil, fmt.Errorf("准备查询 ListLatestSessionFiles 时出错: %w", err)
	}
	if q.listMessagesBySessionStmt, err = db.PrepareContext(ctx, listMessagesBySession); err != nil {
		return nil, fmt.Errorf("准备查询 ListMessagesBySession 时出错: %w", err)
	}
	if q.listNewFilesStmt, err = db.PrepareContext(ctx, listNewFiles); err != nil {
		return nil, fmt.Errorf("准备查询 ListNewFiles 时出错: %w", err)
	}
	if q.listSessionReadFilesStmt, err = db.PrepareContext(ctx, listSessionReadFiles); err != nil {
		return nil, fmt.Errorf("准备查询 ListSessionReadFiles 时出错: %w", err)
	}
	if q.listSessionsStmt, err = db.PrepareContext(ctx, listSessions); err != nil {
		return nil, fmt.Errorf("准备查询 ListSessions 时出错: %w", err)
	}
	if q.listUserMessagesBySessionStmt, err = db.PrepareContext(ctx, listUserMessagesBySession); err != nil {
		return nil, fmt.Errorf("准备查询 ListUserMessagesBySession 时出错: %w", err)
	}
	if q.recordFileReadStmt, err = db.PrepareContext(ctx, recordFileRead); err != nil {
		return nil, fmt.Errorf("准备查询 RecordFileRead 时出错: %w", err)
	}
	if q.updateMessageStmt, err = db.PrepareContext(ctx, updateMessage); err != nil {
		return nil, fmt.Errorf("准备查询 UpdateMessage 时出错: %w", err)
	}
	if q.updateSessionStmt, err = db.PrepareContext(ctx, updateSession); err != nil {
		return nil, fmt.Errorf("准备查询 UpdateSession 时出错: %w", err)
	}
	if q.updateSessionTitleAndUsageStmt, err = db.PrepareContext(ctx, updateSessionTitleAndUsage); err != nil {
		return nil, fmt.Errorf("准备查询 UpdateSessionTitleAndUsage 时出错: %w", err)
	}
	return &q, nil
}

// Close 关闭所有预编译的 SQL 语句，释放相关资源
// 返回值: 关闭过程中遇到的第一个错误（如果有）
func (q *Queries) Close() error {
	var err error
	if q.createFileStmt != nil {
		if cerr := q.createFileStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 createFileStmt 时出错: %w", cerr)
		}
	}
	if q.createMessageStmt != nil {
		if cerr := q.createMessageStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 createMessageStmt 时出错: %w", cerr)
		}
	}
	if q.createSessionStmt != nil {
		if cerr := q.createSessionStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 createSessionStmt 时出错: %w", cerr)
		}
	}
	if q.deleteFileStmt != nil {
		if cerr := q.deleteFileStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 deleteFileStmt 时出错: %w", cerr)
		}
	}
	if q.deleteMessageStmt != nil {
		if cerr := q.deleteMessageStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 deleteMessageStmt 时出错: %w", cerr)
		}
	}
	if q.deleteSessionStmt != nil {
		if cerr := q.deleteSessionStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 deleteSessionStmt 时出错: %w", cerr)
		}
	}
	if q.deleteSessionFilesStmt != nil {
		if cerr := q.deleteSessionFilesStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 deleteSessionFilesStmt 时出错: %w", cerr)
		}
	}
	if q.deleteSessionMessagesStmt != nil {
		if cerr := q.deleteSessionMessagesStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 deleteSessionMessagesStmt 时出错: %w", cerr)
		}
	}
	if q.getAverageResponseTimeStmt != nil {
		if cerr := q.getAverageResponseTimeStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 getAverageResponseTimeStmt 时出错: %w", cerr)
		}
	}
	if q.getFileStmt != nil {
		if cerr := q.getFileStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 getFileStmt 时出错: %w", cerr)
		}
	}
	if q.getFileByPathAndSessionStmt != nil {
		if cerr := q.getFileByPathAndSessionStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 getFileByPathAndSessionStmt 时出错: %w", cerr)
		}
	}
	if q.getFileReadStmt != nil {
		if cerr := q.getFileReadStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 getFileReadStmt 时出错: %w", cerr)
		}
	}
	if q.getHourDayHeatmapStmt != nil {
		if cerr := q.getHourDayHeatmapStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 getHourDayHeatmapStmt 时出错: %w", cerr)
		}
	}
	if q.getMessageStmt != nil {
		if cerr := q.getMessageStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 getMessageStmt 时出错: %w", cerr)
		}
	}
	if q.getRecentActivityStmt != nil {
		if cerr := q.getRecentActivityStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 getRecentActivityStmt 时出错: %w", cerr)
		}
	}
	if q.getSessionByIDStmt != nil {
		if cerr := q.getSessionByIDStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 getSessionByIDStmt 时出错: %w", cerr)
		}
	}
	if q.getToolUsageStmt != nil {
		if cerr := q.getToolUsageStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 getToolUsageStmt 时出错: %w", cerr)
		}
	}
	if q.getTotalStatsStmt != nil {
		if cerr := q.getTotalStatsStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 getTotalStatsStmt 时出错: %w", cerr)
		}
	}
	if q.getUsageByDayStmt != nil {
		if cerr := q.getUsageByDayStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 getUsageByDayStmt 时出错: %w", cerr)
		}
	}
	if q.getUsageByDayOfWeekStmt != nil {
		if cerr := q.getUsageByDayOfWeekStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 getUsageByDayOfWeekStmt 时出错: %w", cerr)
		}
	}
	if q.getUsageByHourStmt != nil {
		if cerr := q.getUsageByHourStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 getUsageByHourStmt 时出错: %w", cerr)
		}
	}
	if q.getUsageByModelStmt != nil {
		if cerr := q.getUsageByModelStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 getUsageByModelStmt 时出错: %w", cerr)
		}
	}
	if q.listAllUserMessagesStmt != nil {
		if cerr := q.listAllUserMessagesStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 listAllUserMessagesStmt 时出错: %w", cerr)
		}
	}
	if q.listFilesByPathStmt != nil {
		if cerr := q.listFilesByPathStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 listFilesByPathStmt 时出错: %w", cerr)
		}
	}
	if q.listFilesBySessionStmt != nil {
		if cerr := q.listFilesBySessionStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 listFilesBySessionStmt 时出错: %w", cerr)
		}
	}
	if q.listLatestSessionFilesStmt != nil {
		if cerr := q.listLatestSessionFilesStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 listLatestSessionFilesStmt 时出错: %w", cerr)
		}
	}
	if q.listMessagesBySessionStmt != nil {
		if cerr := q.listMessagesBySessionStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 listMessagesBySessionStmt 时出错: %w", cerr)
		}
	}
	if q.listNewFilesStmt != nil {
		if cerr := q.listNewFilesStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 listNewFilesStmt 时出错: %w", cerr)
		}
	}
	if q.listSessionReadFilesStmt != nil {
		if cerr := q.listSessionReadFilesStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 listSessionReadFilesStmt 时出错: %w", cerr)
		}
	}
	if q.listSessionsStmt != nil {
		if cerr := q.listSessionsStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 listSessionsStmt 时出错: %w", cerr)
		}
	}
	if q.listUserMessagesBySessionStmt != nil {
		if cerr := q.listUserMessagesBySessionStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 listUserMessagesBySessionStmt 时出错: %w", cerr)
		}
	}
	if q.recordFileReadStmt != nil {
		if cerr := q.recordFileReadStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 recordFileReadStmt 时出错: %w", cerr)
		}
	}
	if q.updateMessageStmt != nil {
		if cerr := q.updateMessageStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 updateMessageStmt 时出错: %w", cerr)
		}
	}
	if q.updateSessionStmt != nil {
		if cerr := q.updateSessionStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 updateSessionStmt 时出错: %w", cerr)
		}
	}
	if q.updateSessionTitleAndUsageStmt != nil {
		if cerr := q.updateSessionTitleAndUsageStmt.Close(); cerr != nil {
			err = fmt.Errorf("关闭 updateSessionTitleAndUsageStmt 时出错: %w", cerr)
		}
	}
	return err
}

// exec 执行 SQL 查询语句，根据是否在事务中使用预编译语句或直接执行
// 参数 ctx: 上下文对象
// 参数 stmt: 预编译的 SQL 语句（可能为 nil）
// 参数 query: SQL 查询字符串
// 参数 args: 查询参数
// 返回值: 执行结果和可能的错误
func (q *Queries) exec(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) (sql.Result, error) {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).ExecContext(ctx, args...)
	case stmt != nil:
		return stmt.ExecContext(ctx, args...)
	default:
		return q.db.ExecContext(ctx, query, args...)
	}
}

// query 执行 SQL 查询并返回多行结果，根据是否在事务中使用预编译语句或直接执行
// 参数 ctx: 上下文对象
// 参数 stmt: 预编译的 SQL 语句（可能为 nil）
// 参数 query: SQL 查询字符串
// 参数 args: 查询参数
// 返回值: 查询结果集和可能的错误
func (q *Queries) query(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) (*sql.Rows, error) {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).QueryContext(ctx, args...)
	case stmt != nil:
		return stmt.QueryContext(ctx, args...)
	default:
		return q.db.QueryContext(ctx, query, args...)
	}
}

// queryRow 执行 SQL 查询并返回单行结果，根据是否在事务中使用预编译语句或直接执行
// 参数 ctx: 上下文对象
// 参数 stmt: 预编译的 SQL 语句（可能为 nil）
// 参数 query: SQL 查询字符串
// 参数 args: 查询参数
// 返回值: 单行查询结果
func (q *Queries) queryRow(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) *sql.Row {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).QueryRowContext(ctx, args...)
	case stmt != nil:
		return stmt.QueryRowContext(ctx, args...)
	default:
		return q.db.QueryRowContext(ctx, query, args...)
	}
}

// Queries 结构体封装了所有数据库查询操作
// 包含数据库连接、事务对象以及所有预编译的 SQL 语句
type Queries struct {
	db                             DBTX      // 数据库连接对象，实现了 DBTX 接口
	tx                             *sql.Tx   // 数据库事务对象（可选）
	createFileStmt                 *sql.Stmt // 创建文件的预编译语句
	createMessageStmt              *sql.Stmt // 创建消息的预编译语句
	createSessionStmt              *sql.Stmt // 创建会话的预编译语句
	deleteFileStmt                 *sql.Stmt // 删除文件的预编译语句
	deleteMessageStmt              *sql.Stmt // 删除消息的预编译语句
	deleteSessionStmt              *sql.Stmt // 删除会话的预编译语句
	deleteSessionFilesStmt         *sql.Stmt // 删除会话文件的预编译语句
	deleteSessionMessagesStmt      *sql.Stmt // 删除会话消息的预编译语句
	getAverageResponseTimeStmt     *sql.Stmt // 获取平均响应时间的预编译语句
	getFileStmt                    *sql.Stmt // 获取文件的预编译语句
	getFileByPathAndSessionStmt    *sql.Stmt // 根据路径和会话获取文件的预编译语句
	getFileReadStmt                *sql.Stmt // 获取文件读取记录的预编译语句
	getHourDayHeatmapStmt          *sql.Stmt // 获取小时-日期热力图的预编译语句
	getMessageStmt                 *sql.Stmt // 获取消息的预编译语句
	getRecentActivityStmt          *sql.Stmt // 获取最近活动的预编译语句
	getSessionByIDStmt             *sql.Stmt // 根据ID获取会话的预编译语句
	getToolUsageStmt               *sql.Stmt // 获取工具使用情况的预编译语句
	getTotalStatsStmt              *sql.Stmt // 获取总统计数据的预编译语句
	getUsageByDayStmt              *sql.Stmt // 按天获取使用情况的预编译语句
	getUsageByDayOfWeekStmt        *sql.Stmt // 按星期获取使用情况的预编译语句
	getUsageByHourStmt             *sql.Stmt // 按小时获取使用情况的预编译语句
	getUsageByModelStmt            *sql.Stmt // 按模型获取使用情况的预编译语句
	listAllUserMessagesStmt        *sql.Stmt // 列出所有用户消息的预编译语句
	listFilesByPathStmt            *sql.Stmt // 按路径列出文件的预编译语句
	listFilesBySessionStmt         *sql.Stmt // 按会话列出文件的预编译语句
	listLatestSessionFilesStmt     *sql.Stmt // 列出最新会话文件的预编译语句
	listMessagesBySessionStmt      *sql.Stmt // 按会话列出消息的预编译语句
	listNewFilesStmt               *sql.Stmt // 列出新文件的预编译语句
	listSessionReadFilesStmt       *sql.Stmt // 列出会话已读文件的预编译语句
	listSessionsStmt               *sql.Stmt // 列出会话的预编译语句
	listUserMessagesBySessionStmt  *sql.Stmt // 按会话列出用户消息的预编译语句
	recordFileReadStmt             *sql.Stmt // 记录文件读取的预编译语句
	updateMessageStmt              *sql.Stmt // 更新消息的预编译语句
	updateSessionStmt              *sql.Stmt // 更新会话的预编译语句
	updateSessionTitleAndUsageStmt *sql.Stmt // 更新会话标题和使用情况的预编译语句
}

// WithTx 创建并返回一个与指定事务关联的新的 Queries 实例
// 该方法允许在事务上下文中执行所有数据库操作
// 参数 tx: 数据库事务对象
// 返回值: 与事务关联的新的 Queries 实例
func (q *Queries) WithTx(tx *sql.Tx) *Queries {
	return &Queries{
		db:                             tx,
		tx:                             tx,
		createFileStmt:                 q.createFileStmt,
		createMessageStmt:              q.createMessageStmt,
		createSessionStmt:              q.createSessionStmt,
		deleteFileStmt:                 q.deleteFileStmt,
		deleteMessageStmt:              q.deleteMessageStmt,
		deleteSessionStmt:              q.deleteSessionStmt,
		deleteSessionFilesStmt:         q.deleteSessionFilesStmt,
		deleteSessionMessagesStmt:      q.deleteSessionMessagesStmt,
		getAverageResponseTimeStmt:     q.getAverageResponseTimeStmt,
		getFileStmt:                    q.getFileStmt,
		getFileByPathAndSessionStmt:    q.getFileByPathAndSessionStmt,
		getFileReadStmt:                q.getFileReadStmt,
		getHourDayHeatmapStmt:          q.getHourDayHeatmapStmt,
		getMessageStmt:                 q.getMessageStmt,
		getRecentActivityStmt:          q.getRecentActivityStmt,
		getSessionByIDStmt:             q.getSessionByIDStmt,
		getToolUsageStmt:               q.getToolUsageStmt,
		getTotalStatsStmt:              q.getTotalStatsStmt,
		getUsageByDayStmt:              q.getUsageByDayStmt,
		getUsageByDayOfWeekStmt:        q.getUsageByDayOfWeekStmt,
		getUsageByHourStmt:             q.getUsageByHourStmt,
		getUsageByModelStmt:            q.getUsageByModelStmt,
		listAllUserMessagesStmt:        q.listAllUserMessagesStmt,
		listFilesByPathStmt:            q.listFilesByPathStmt,
		listFilesBySessionStmt:         q.listFilesBySessionStmt,
		listLatestSessionFilesStmt:     q.listLatestSessionFilesStmt,
		listMessagesBySessionStmt:      q.listMessagesBySessionStmt,
		listNewFilesStmt:               q.listNewFilesStmt,
		listSessionReadFilesStmt:       q.listSessionReadFilesStmt,
		listSessionsStmt:               q.listSessionsStmt,
		listUserMessagesBySessionStmt:  q.listUserMessagesBySessionStmt,
		recordFileReadStmt:             q.recordFileReadStmt,
		updateMessageStmt:              q.updateMessageStmt,
		updateSessionStmt:              q.updateSessionStmt,
		updateSessionTitleAndUsageStmt: q.updateSessionTitleAndUsageStmt,
	}
}
