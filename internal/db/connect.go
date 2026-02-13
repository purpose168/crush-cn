package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/pressly/goose/v3"
)

// Connect 打开 SQLite 数据库连接并运行迁移
// Connect opens a SQLite database connection and runs migrations.
func Connect(ctx context.Context, dataDir string) (*sql.DB, error) {
	// 检查数据目录是否已设置
	if dataDir == "" {
		return nil, fmt.Errorf("data.dir 未设置")
	}
	// 构建数据库文件路径
	dbPath := filepath.Join(dataDir, "crush.db")

	// 打开数据库连接
	db, err := openDB(dbPath)
	if err != nil {
		return nil, err
	}

	// 验证数据库连接是否可用
	if err = db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 设置 goose 的基础文件系统
	goose.SetBaseFS(FS)

	// 设置数据库方言为 sqlite3
	if err := goose.SetDialect("sqlite3"); err != nil {
		slog.Error("设置方言失败", "error", err)
		return nil, fmt.Errorf("设置方言失败: %w", err)
	}

	// 执行数据库迁移
	if err := goose.Up(db, "migrations"); err != nil {
		slog.Error("应用迁移失败", "error", err)
		return nil, fmt.Errorf("应用迁移失败: %w", err)
	}

	// 返回数据库连接
	return db, nil
}
