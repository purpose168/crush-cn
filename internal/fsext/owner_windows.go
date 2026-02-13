//go:build windows

package fsext

import "os"

// Owner 获取指定路径的文件或目录所有者的用户ID。
func Owner(path string) (int, error) {
	_, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return -1, nil
}
