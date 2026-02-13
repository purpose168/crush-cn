//go:build !((darwin && (amd64 || arm64)) || (freebsd && (amd64 || arm64)) || (linux && (386 || amd64 || arm || arm64 || loong64 || ppc64le || riscv64 || s390x)) || (windows && (386 || amd64 || arm64)))

package db

import (
	"database/sql"
	"fmt"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// openDB 打开 SQLite 数据库并配置性能优化参数
// dbPath: 数据库文件的路径
// 返回: *sql.DB 数据库连接对象, error 错误信息
func openDB(dbPath string) (*sql.DB, error) {
	// 设置数据库参数（PRAGMA）以提升性能
	pragmas := []string{
		"PRAGMA foreign_keys = ON;",    // 启用外键约束
		"PRAGMA journal_mode = WAL;",   // 使用WAL（预写式日志）模式
		"PRAGMA page_size = 4096;",     // 设置页面大小为4096字节
		"PRAGMA cache_size = -8000;",   // 设置缓存大小为8000KB
		"PRAGMA synchronous = NORMAL;", // 设置同步模式为普通级别
		"PRAGMA secure_delete = ON;",   // 启用安全删除
		"PRAGMA busy_timeout = 5000;",  // 设置忙碌超时为5000毫秒
	}

	db, err := driver.Open(dbPath, func(c *sqlite3.Conn) error {
		for _, pragma := range pragmas {
			if err := c.Exec(pragma); err != nil {
				return fmt.Errorf("设置数据库参数（PRAGMA）%q 失败: %w", pragma, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}
	return db, nil
}
