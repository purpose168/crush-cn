// Package home 提供处理用户主目录的工具函数。
package home

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// homedir 存储用户主目录路径
// homedirErr 存储获取主目录时的错误信息
var homedir, homedirErr = os.UserHomeDir()

// init 初始化函数，在包加载时执行
// 如果获取用户主目录失败，记录错误日志
func init() {
	if homedirErr != nil {
		slog.Error("获取用户主目录失败", "error", homedirErr)
	}
}

// Dir 返回用户主目录路径
// 返回值: 用户主目录的完整路径字符串
func Dir() string {
	return homedir
}

// Short 将完整路径中的用户主目录部分替换为 `~` 符号
// 这是一个路径缩写函数，用于在显示时简化路径表示
// 参数 p: 需要缩写的完整路径
// 返回值: 缩写后的路径（主目录部分用 ~ 表示）
func Short(p string) string {
	// 如果主目录为空或路径不以主目录开头，直接返回原路径
	if homedir == "" || !strings.HasPrefix(p, homedir) {
		return p
	}
	// 将主目录路径替换为 ~，保留路径的其余部分
	return filepath.Join("~", strings.TrimPrefix(p, homedir))
}

// Long 将路径中的 `~` 符号展开为实际的用户主目录路径
// 这是一个路径展开函数，用于将缩写路径转换为完整路径
// 参数 p: 需要展开的路径（可能包含 ~）
// 返回值: 展开后的完整路径
func Long(p string) string {
	// 如果主目录为空或路径不以 ~ 开头，直接返回原路径
	if homedir == "" || !strings.HasPrefix(p, "~") {
		return p
	}
	// 将 ~ 替换为实际的主目录路径（只替换第一个匹配项）
	return strings.Replace(p, "~", homedir, 1)
}
