package fsext

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestListDirectory 测试 ListDirectory 函数的功能
func TestListDirectory(t *testing.T) {
	// 创建临时测试目录
	tmp := t.TempDir()

	// 定义测试文件映射：文件名 -> 文件内容
	testFiles := map[string]string{
		"regular.txt":     "content",        // 普通文本文件
		".hidden":         "hidden content", // 隐藏文件
		".gitignore":      ".*\n*.log\n",    // gitignore 配置文件
		"subdir/file.go":  "package main",   // 子目录中的 Go 源文件
		"subdir/.another": "more hidden",    // 子目录中的隐藏文件
		"build.log":       "build output",   // 构建日志文件
	}

	// 创建所有测试文件和目录
	for name, content := range testFiles {
		fp := filepath.Join(tmp, name)
		dir := filepath.Dir(fp)
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(fp, []byte(content), 0o644))
	}

	// 测试场景：无数量限制
	t.Run("无数量限制", func(t *testing.T) {
		files, truncated, err := ListDirectory(tmp, nil, -1, -1)
		require.NoError(t, err)
		require.False(t, truncated) // 不应被截断
		require.Len(t, files, 4)    // 应返回 4 个文件
		require.ElementsMatch(t, []string{
			"regular.txt",
			"subdir",
			"subdir/.another",
			"subdir/file.go",
		}, relPaths(t, files, tmp))
	})

	// 测试场景：有数量限制
	t.Run("有数量限制", func(t *testing.T) {
		files, truncated, err := ListDirectory(tmp, nil, -1, 2)
		require.NoError(t, err)
		require.True(t, truncated) // 应被截断
		require.Len(t, files, 2)   // 应返回 2 个文件
	})
}

// relPaths 将绝对路径列表转换为相对于基准目录的相对路径列表
// 参数：
//   - tb: 测试上下文
//   - in: 绝对路径列表
//   - base: 基准目录
//
// 返回：相对路径列表
func relPaths(tb testing.TB, in []string, base string) []string {
	tb.Helper()
	out := make([]string, 0, len(in))
	for _, p := range in {
		rel, err := filepath.Rel(base, p)
		require.NoError(tb, err)
		out = append(out, filepath.ToSlash(rel))
	}
	return out
}
