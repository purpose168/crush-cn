package fsext

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCrushIgnore(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	// 创建测试文件
	require.NoError(t, os.WriteFile("test1.txt", []byte("test"), 0o644))
	require.NoError(t, os.WriteFile("test2.log", []byte("test"), 0o644))
	require.NoError(t, os.WriteFile("test3.tmp", []byte("test"), 0o644))

	// 创建一个忽略 .log 文件的 .crushignore 文件
	require.NoError(t, os.WriteFile(".crushignore", []byte("*.log\n"), 0o644))

	dl := NewDirectoryLister(tempDir)
	require.True(t, dl.shouldIgnore("test2.log", nil), ".log 文件应该被忽略")
	require.False(t, dl.shouldIgnore("test1.txt", nil), ".txt 文件不应该被忽略")
	require.True(t, dl.shouldIgnore("test3.tmp", nil), ".tmp 文件应该被通用模式忽略")
}

func TestShouldExcludeFile(t *testing.T) {
	t.Parallel()

	// 创建用于测试的临时目录结构
	tempDir := t.TempDir()

	// 创建应该被忽略的目录
	nodeModules := filepath.Join(tempDir, "node_modules")
	target := filepath.Join(tempDir, "target")
	customIgnored := filepath.Join(tempDir, "custom_ignored")
	normalDir := filepath.Join(tempDir, "src")

	for _, dir := range []string{nodeModules, target, customIgnored, normalDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("创建目录 %s 失败: %v", dir, err)
		}
	}

	// 创建 .gitignore 文件
	gitignoreContent := "node_modules/\ntarget/\n"
	if err := os.WriteFile(filepath.Join(tempDir, ".gitignore"), []byte(gitignoreContent), 0o644); err != nil {
		t.Fatalf("创建 .gitignore 失败: %v", err)
	}

	// 创建 .crushignore 文件
	crushignoreContent := "custom_ignored/\n"
	if err := os.WriteFile(filepath.Join(tempDir, ".crushignore"), []byte(crushignoreContent), 0o644); err != nil {
		t.Fatalf("创建 .crushignore 失败: %v", err)
	}

	// 测试被忽略的目录是否正确忽略
	require.True(t, ShouldExcludeFile(tempDir, nodeModules), "期望 node_modules 被 .gitignore 忽略")
	require.True(t, ShouldExcludeFile(tempDir, target), "期望 target 被 .gitignore 忽略")
	require.True(t, ShouldExcludeFile(tempDir, customIgnored), "期望 custom_ignored 被 .crushignore 忽略")

	// 测试正常目录不被忽略
	require.False(t, ShouldExcludeFile(tempDir, normalDir), "期望 src 目录不被忽略")

	// 测试工作区根目录本身不被忽略
	require.False(t, ShouldExcludeFile(tempDir, tempDir), "期望工作区根目录不被忽略")
}

func TestShouldExcludeFileHierarchical(t *testing.T) {
	t.Parallel()

	// 创建用于测试分层忽略的嵌套目录结构
	tempDir := t.TempDir()

	// 创建嵌套目录
	subDir := filepath.Join(tempDir, "subdir")
	nestedNormal := filepath.Join(subDir, "normal_nested")

	for _, dir := range []string{subDir, nestedNormal} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("创建目录 %s 失败: %v", dir, err)
		}
	}

	// 在子目录中创建忽略 normal_nested 的 .crushignore 文件
	subCrushignore := "normal_nested/\n"
	if err := os.WriteFile(filepath.Join(subDir, ".crushignore"), []byte(subCrushignore), 0o644); err != nil {
		t.Fatalf("创建子目录 .crushignore 失败: %v", err)
	}

	// 测试分层忽略行为 - 这应该有效，因为 .crushignore 位于父目录中
	require.True(t, ShouldExcludeFile(tempDir, nestedNormal), "期望 normal_nested 被子目录 .crushignore 忽略")
	require.False(t, ShouldExcludeFile(tempDir, subDir), "期望子目录本身不被忽略")
}

func TestShouldExcludeFileCommonPatterns(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// 创建应该被通用模式忽略的目录
	commonIgnored := []string{
		filepath.Join(tempDir, ".git"),
		filepath.Join(tempDir, "node_modules"),
		filepath.Join(tempDir, "__pycache__"),
		filepath.Join(tempDir, "target"),
		filepath.Join(tempDir, ".vscode"),
	}

	for _, dir := range commonIgnored {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("创建目录 %s 失败: %v", dir, err)
		}
	}

	// 测试通用模式在没有显式忽略文件的情况下也能被忽略
	for _, dir := range commonIgnored {
		require.True(t, ShouldExcludeFile(tempDir, dir), "期望 %s 被通用模式忽略", filepath.Base(dir))
	}
}
