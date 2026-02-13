//go:build (darwin && (amd64 || arm64)) || (freebsd && (amd64 || arm64)) || (linux && (386 || amd64 || arm || arm64 || loong64 || ppc64le || riscv64 || s390x)) || (windows && (386 || amd64 || arm64))

package db

import (
	"database/sql"
	"fmt"
	"net/url"

	_ "modernc.org/sqlite"
)

func openDB(dbPath string) (*sql.DB, error) {
	// 通过 _pragma 查询参数设置 pragma 以获得更好的性能。
	// 格式：_pragma=name(value)
	params := url.Values{}
	params.Add("_pragma", "foreign_keys(on)")
	params.Add("_pragma", "journal_mode(WAL)")
	params.Add("_pragma", "page_size(4096)")
	params.Add("_pragma", "cache_size(-8000)")
	params.Add("_pragma", "synchronous(NORMAL)")
	params.Add("_pragma", "secure_delete(on)")
	params.Add("_pragma", "busy_timeout(5000)")

	dsn := fmt.Sprintf("file:%s?%s", dbPath, params.Encode())
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}
	return db, nil
}
