package tools

import (
	"context"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/purpose168/crush-cn/internal/log"
)

// getRg 是一个延迟初始化函数，用于查找ripgrep (rg) 可执行文件的路径
var getRg = sync.OnceValue(func() string {
	path, err := exec.LookPath("rg")
	if err != nil {
		if log.Initialized() {
			slog.Warn("在$PATH中未找到Ripgrep (rg)。某些grep功能可能会受限或更慢。")
		}
		return ""
	}
	return path
})

// getRgCmd 创建一个用于列出文件的ripgrep命令
// ctx: 上下文对象
// globPattern: 全局模式
// 返回ripgrep命令对象
func getRgCmd(ctx context.Context, globPattern string) *exec.Cmd {
	name := getRg()
	if name == "" {
		return nil
	}
	args := []string{"--files", "-L", "--null"}
	if globPattern != "" {
		if !filepath.IsAbs(globPattern) && !strings.HasPrefix(globPattern, "/") {
			globPattern = "/" + globPattern
		}
		args = append(args, "--glob", globPattern)
	}
	return exec.CommandContext(ctx, name, args...)
}

// getRgSearchCmd 创建一个用于搜索的ripgrep命令
// ctx: 上下文对象
// pattern: 搜索模式
// path: 搜索路径
// include: 包含模式
// 返回ripgrep命令对象
func getRgSearchCmd(ctx context.Context, pattern, path, include string) *exec.Cmd {
	name := getRg()
	if name == "" {
		return nil
	}
	// 使用 -n 显示行号，-0 用于空分隔以处理Windows路径
	args := []string{"--json", "-H", "-n", "-0", pattern}
	if include != "" {
		args = append(args, "--glob", include)
	}
	args = append(args, path)

	return exec.CommandContext(ctx, name, args...)
}
