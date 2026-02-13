// Package db 提供数据库相关的功能
// 本文件使用 Go 语言的 embed 指令将数据库迁移 SQL 文件嵌入到二进制文件中
// 这样可以在不依赖外部文件系统的情况下执行数据库迁移操作
package db

import "embed"

// 使用 embed 指令将 migrations 目录下的所有 .sql 文件嵌入到程序中
// 这些 SQL 文件包含了数据库迁移脚本，用于创建和修改数据库表结构
//
//go:embed migrations/*.sql
var FS embed.FS // FS 是一个嵌入的文件系统，包含了所有的数据库迁移 SQL 文件
