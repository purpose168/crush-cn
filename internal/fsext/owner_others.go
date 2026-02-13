//go:build !windows

package fsext

import (
	"os"
	"syscall"
)

// Owner 获取指定路径的文件或目录的所有者用户ID。
func Owner(path string) (int, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	var uid int
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		uid = int(stat.Uid)
	} else {
		uid = os.Getuid()
	}
	return uid, nil
}
