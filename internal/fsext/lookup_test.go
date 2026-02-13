package fsext

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/purpose168/crush-cn/internal/home"
	"github.com/stretchr/testify/require"
)

func TestLookupClosest(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	t.Run("在起始目录中找到目标文件", func(t *testing.T) {
		testDir := t.TempDir()

		// 在当前目录中创建目标文件
		targetFile := filepath.Join(testDir, "target.txt")
		err := os.WriteFile(targetFile, []byte("test"), 0o644)
		require.NoError(t, err)

		foundPath, found := LookupClosest(testDir, "target.txt")
		require.True(t, found)
		require.Equal(t, targetFile, foundPath)
	})

	t.Run("在父目录中找到目标文件", func(t *testing.T) {
		testDir := t.TempDir()

		// 创建子目录
		subDir := filepath.Join(testDir, "subdir")
		err := os.Mkdir(subDir, 0o755)
		require.NoError(t, err)

		// 在父目录中创建目标文件
		targetFile := filepath.Join(testDir, "target.txt")
		err = os.WriteFile(targetFile, []byte("test"), 0o644)
		require.NoError(t, err)

		foundPath, found := LookupClosest(subDir, "target.txt")
		require.True(t, found)
		require.Equal(t, targetFile, foundPath)
	})

	t.Run("在祖父目录中找到目标文件", func(t *testing.T) {
		testDir := t.TempDir()

		// 创建嵌套子目录
		subDir := filepath.Join(testDir, "subdir")
		err := os.Mkdir(subDir, 0o755)
		require.NoError(t, err)

		subSubDir := filepath.Join(subDir, "subsubdir")
		err = os.Mkdir(subSubDir, 0o755)
		require.NoError(t, err)

		// 在祖父目录中创建目标文件
		targetFile := filepath.Join(testDir, "target.txt")
		err = os.WriteFile(targetFile, []byte("test"), 0o644)
		require.NoError(t, err)

		foundPath, found := LookupClosest(subSubDir, "target.txt")
		require.True(t, found)
		require.Equal(t, targetFile, foundPath)
	})

	t.Run("未找到目标文件", func(t *testing.T) {
		testDir := t.TempDir()

		foundPath, found := LookupClosest(testDir, "nonexistent.txt")
		require.False(t, found)
		require.Empty(t, foundPath)
	})

	t.Run("找到目标目录", func(t *testing.T) {
		testDir := t.TempDir()

		// 在当前目录中创建目标目录
		targetDir := filepath.Join(testDir, "targetdir")
		err := os.Mkdir(targetDir, 0o755)
		require.NoError(t, err)

		foundPath, found := LookupClosest(testDir, "targetdir")
		require.True(t, found)
		require.Equal(t, targetDir, foundPath)
	})

	t.Run("在主目录处停止搜索", func(t *testing.T) {
		// 此测试存在局限性，因为我们无法轻松在主目录之上创建文件
		// 但我们可以通过从主目录本身进行搜索来测试该行为
		homeDir := home.Dir()

		// 从主目录搜索一个不存在的文件
		foundPath, found := LookupClosest(homeDir, "nonexistent_file_12345.txt")
		require.False(t, found)
		require.Empty(t, foundPath)
	})

	t.Run("无效的起始目录", func(t *testing.T) {
		foundPath, found := LookupClosest("/invalid/path/that/does/not/exist", "target.txt")
		require.False(t, found)
		require.Empty(t, foundPath)
	})

	t.Run("相对路径处理", func(t *testing.T) {
		// 在当前目录中创建目标文件
		require.NoError(t, os.WriteFile("target.txt", []byte("test"), 0o644))

		// 使用相对路径进行搜索
		foundPath, found := LookupClosest(".", "target.txt")
		require.True(t, found)

		// 解析符号链接以处理 macOS /private/var 与 /var 的差异
		expectedPath, err := filepath.EvalSymlinks(filepath.Join(tempDir, "target.txt"))
		require.NoError(t, err)
		actualPath, err := filepath.EvalSymlinks(foundPath)
		require.NoError(t, err)
		require.Equal(t, expectedPath, actualPath)
	})
}

func TestLookupClosestWithOwnership(t *testing.T) {
	// 注意：以跨平台方式测试所有权边界较为困难，
	// 需要创建具有不同所有者的复杂目录结构。
	// 此测试侧重于所有权检查通过时的基本功能。

	tempDir := t.TempDir()
	t.Chdir(tempDir)

	t.Run("搜索遵循相同所有权", func(t *testing.T) {
		testDir := t.TempDir()

		// 创建子目录结构
		subDir := filepath.Join(testDir, "subdir")
		err := os.Mkdir(subDir, 0o755)
		require.NoError(t, err)

		// 在父目录中创建目标文件
		targetFile := filepath.Join(testDir, "target.txt")
		err = os.WriteFile(targetFile, []byte("test"), 0o644)
		require.NoError(t, err)

		// 假设所有权相同，搜索应能找到目标
		foundPath, found := LookupClosest(subDir, "target.txt")
		require.True(t, found)
		require.Equal(t, targetFile, foundPath)
	})
}

func TestLookup(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	t.Run("无目标时返回空切片", func(t *testing.T) {
		testDir := t.TempDir()

		found, err := Lookup(testDir)
		require.NoError(t, err)
		require.Empty(t, found)
	})

	t.Run("在起始目录中找到单个目标文件", func(t *testing.T) {
		testDir := t.TempDir()

		// 在当前目录中创建目标文件
		targetFile := filepath.Join(testDir, "target.txt")
		err := os.WriteFile(targetFile, []byte("test"), 0o644)
		require.NoError(t, err)

		found, err := Lookup(testDir, "target.txt")
		require.NoError(t, err)
		require.Len(t, found, 1)
		require.Equal(t, targetFile, found[0])
	})

	t.Run("在起始目录中找到多个目标文件", func(t *testing.T) {
		testDir := t.TempDir()

		// 在当前目录中创建多个目标文件
		targetFile1 := filepath.Join(testDir, "target1.txt")
		targetFile2 := filepath.Join(testDir, "target2.txt")
		targetFile3 := filepath.Join(testDir, "target3.txt")

		err := os.WriteFile(targetFile1, []byte("test1"), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(targetFile2, []byte("test2"), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(targetFile3, []byte("test3"), 0o644)
		require.NoError(t, err)

		found, err := Lookup(testDir, "target1.txt", "target2.txt", "target3.txt")
		require.NoError(t, err)
		require.Len(t, found, 3)
		require.Contains(t, found, targetFile1)
		require.Contains(t, found, targetFile2)
		require.Contains(t, found, targetFile3)
	})

	t.Run("在父目录中找到目标文件", func(t *testing.T) {
		testDir := t.TempDir()

		// 创建子目录
		subDir := filepath.Join(testDir, "subdir")
		err := os.Mkdir(subDir, 0o755)
		require.NoError(t, err)

		// 在父目录中创建目标文件
		targetFile1 := filepath.Join(testDir, "target1.txt")
		targetFile2 := filepath.Join(testDir, "target2.txt")
		err = os.WriteFile(targetFile1, []byte("test1"), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(targetFile2, []byte("test2"), 0o644)
		require.NoError(t, err)

		found, err := Lookup(subDir, "target1.txt", "target2.txt")
		require.NoError(t, err)
		require.Len(t, found, 2)
		require.Contains(t, found, targetFile1)
		require.Contains(t, found, targetFile2)
	})

	t.Run("在多个目录层级中找到目标文件", func(t *testing.T) {
		testDir := t.TempDir()

		// 创建嵌套子目录
		subDir := filepath.Join(testDir, "subdir")
		err := os.Mkdir(subDir, 0o755)
		require.NoError(t, err)

		subSubDir := filepath.Join(subDir, "subsubdir")
		err = os.Mkdir(subSubDir, 0o755)
		require.NoError(t, err)

		// 在不同层级创建目标文件
		targetFile1 := filepath.Join(testDir, "target1.txt")
		targetFile2 := filepath.Join(subDir, "target2.txt")
		targetFile3 := filepath.Join(subSubDir, "target3.txt")

		err = os.WriteFile(targetFile1, []byte("test1"), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(targetFile2, []byte("test2"), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(targetFile3, []byte("test3"), 0o644)
		require.NoError(t, err)

		found, err := Lookup(subSubDir, "target1.txt", "target2.txt", "target3.txt")
		require.NoError(t, err)
		require.Len(t, found, 3)
		require.Contains(t, found, targetFile1)
		require.Contains(t, found, targetFile2)
		require.Contains(t, found, targetFile3)
	})

	t.Run("部分目标文件未找到", func(t *testing.T) {
		testDir := t.TempDir()

		// 仅创建部分目标文件
		targetFile1 := filepath.Join(testDir, "target1.txt")
		targetFile2 := filepath.Join(testDir, "target2.txt")

		err := os.WriteFile(targetFile1, []byte("test1"), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(targetFile2, []byte("test2"), 0o644)
		require.NoError(t, err)

		// 搜索已存在和不存在的目标文件
		found, err := Lookup(testDir, "target1.txt", "nonexistent.txt", "target2.txt", "another_nonexistent.txt")
		require.NoError(t, err)
		require.Len(t, found, 2)
		require.Contains(t, found, targetFile1)
		require.Contains(t, found, targetFile2)
	})

	t.Run("未找到任何目标文件", func(t *testing.T) {
		testDir := t.TempDir()

		found, err := Lookup(testDir, "nonexistent1.txt", "nonexistent2.txt", "nonexistent3.txt")
		require.NoError(t, err)
		require.Empty(t, found)
	})

	t.Run("找到目标目录", func(t *testing.T) {
		testDir := t.TempDir()

		// 创建目标目录
		targetDir1 := filepath.Join(testDir, "targetdir1")
		targetDir2 := filepath.Join(testDir, "targetdir2")
		err := os.Mkdir(targetDir1, 0o755)
		require.NoError(t, err)
		err = os.Mkdir(targetDir2, 0o755)
		require.NoError(t, err)

		found, err := Lookup(testDir, "targetdir1", "targetdir2")
		require.NoError(t, err)
		require.Len(t, found, 2)
		require.Contains(t, found, targetDir1)
		require.Contains(t, found, targetDir2)
	})

	t.Run("文件和目录混合", func(t *testing.T) {
		testDir := t.TempDir()

		// 创建目标文件和目录
		targetFile := filepath.Join(testDir, "target.txt")
		targetDir := filepath.Join(testDir, "targetdir")
		err := os.WriteFile(targetFile, []byte("test"), 0o644)
		require.NoError(t, err)
		err = os.Mkdir(targetDir, 0o755)
		require.NoError(t, err)

		found, err := Lookup(testDir, "target.txt", "targetdir")
		require.NoError(t, err)
		require.Len(t, found, 2)
		require.Contains(t, found, targetFile)
		require.Contains(t, found, targetDir)
	})

	t.Run("无效的起始目录", func(t *testing.T) {
		found, err := Lookup("/invalid/path/that/does/not/exist", "target.txt")
		require.Error(t, err)
		require.Empty(t, found)
	})

	t.Run("相对路径处理", func(t *testing.T) {
		// 在当前目录中创建目标文件
		require.NoError(t, os.WriteFile("target1.txt", []byte("test1"), 0o644))
		require.NoError(t, os.WriteFile("target2.txt", []byte("test2"), 0o644))

		// 使用相对路径进行搜索
		found, err := Lookup(".", "target1.txt", "target2.txt")
		require.NoError(t, err)
		require.Len(t, found, 2)

		// 解析符号链接以处理 macOS /private/var 与 /var 的差异
		expectedPath1, err := filepath.EvalSymlinks(filepath.Join(tempDir, "target1.txt"))
		require.NoError(t, err)
		expectedPath2, err := filepath.EvalSymlinks(filepath.Join(tempDir, "target2.txt"))
		require.NoError(t, err)

		// 检查找到的路径是否与预期路径匹配（顺序可能不同）
		foundEvalSymlinks := make([]string, len(found))
		for i, path := range found {
			evalPath, err := filepath.EvalSymlinks(path)
			require.NoError(t, err)
			foundEvalSymlinks[i] = evalPath
		}

		require.Contains(t, foundEvalSymlinks, expectedPath1)
		require.Contains(t, foundEvalSymlinks, expectedPath2)
	})
}

func TestProbeEnt(t *testing.T) {
	t.Run("存在且所有者正确的文件", func(t *testing.T) {
		tempDir := t.TempDir()

		// 创建测试文件
		testFile := filepath.Join(tempDir, "test.txt")
		err := os.WriteFile(testFile, []byte("test"), 0o644)
		require.NoError(t, err)

		// 获取临时目录的所有者
		owner, err := Owner(tempDir)
		require.NoError(t, err)

		// 使用正确的所有者测试 probeEnt
		err = probeEnt(testFile, owner)
		require.NoError(t, err)
	})

	t.Run("存在且所有者正确的目录", func(t *testing.T) {
		tempDir := t.TempDir()

		// 创建测试目录
		testDir := filepath.Join(tempDir, "testdir")
		err := os.Mkdir(testDir, 0o755)
		require.NoError(t, err)

		// 获取临时目录的所有者
		owner, err := Owner(tempDir)
		require.NoError(t, err)

		// 使用正确的所有者测试 probeEnt
		err = probeEnt(testDir, owner)
		require.NoError(t, err)
	})

	t.Run("不存在的文件", func(t *testing.T) {
		tempDir := t.TempDir()

		nonexistentFile := filepath.Join(tempDir, "nonexistent.txt")
		owner, err := Owner(tempDir)
		require.NoError(t, err)

		err = probeEnt(nonexistentFile, owner)
		require.Error(t, err)
		require.True(t, errors.Is(err, os.ErrNotExist))
	})

	t.Run("不存在目录中的不存在的文件", func(t *testing.T) {
		nonexistentFile := "/this/directory/does/not/exists/nonexistent.txt"

		err := probeEnt(nonexistentFile, -1)
		require.Error(t, err)
		require.True(t, errors.Is(err, os.ErrNotExist))
	})

	t.Run("使用 -1 跳过所有权检查", func(t *testing.T) {
		tempDir := t.TempDir()

		// 创建测试文件
		testFile := filepath.Join(tempDir, "test.txt")
		err := os.WriteFile(testFile, []byte("test"), 0o644)
		require.NoError(t, err)

		// 使用 -1 测试 probeEnt（跳过所有权检查）
		err = probeEnt(testFile, -1)
		require.NoError(t, err)
	})

	t.Run("所有权不匹配返回权限错误", func(t *testing.T) {
		tempDir := t.TempDir()

		// 创建测试文件
		testFile := filepath.Join(tempDir, "test.txt")
		err := os.WriteFile(testFile, []byte("test"), 0o644)
		require.NoError(t, err)

		// 使用不同的所有者测试 probeEnt（使用 9999，不太可能是实际所有者）
		err = probeEnt(testFile, 9999)
		require.Error(t, err)
		require.True(t, errors.Is(err, os.ErrPermission))
	})
}
