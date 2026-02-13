// Package home 提供主目录路径处理功能的单元测试
package home

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDir 测试 Dir 函数
// 验证 Dir() 函数能正确返回用户主目录路径，且路径不为空
func TestDir(t *testing.T) {
	require.NotEmpty(t, Dir())
}

// TestShort 测试 Short 函数
// 验证 Short() 函数能正确将绝对路径转换为短路径格式：
// - 主目录下的路径应转换为 ~ 开头的短路径格式
// - 非主目录下的绝对路径应保持原样不变
func TestShort(t *testing.T) {
	// 测试主目录下的文件路径转换
	d := filepath.Join(Dir(), "documents", "file.txt")
	require.Equal(t, filepath.FromSlash("~/documents/file.txt"), Short(d))

	// 测试非主目录的绝对路径（应保持原样）
	ad := filepath.FromSlash("/absolute/path/file.txt")
	require.Equal(t, ad, Short(ad))
}

// TestLong 测试 Long 函数
// 验证 Long() 函数能正确将短路径转换为完整绝对路径：
// - ~ 开头的短路径应展开为完整的主目录路径
// - 非短路径格式的绝对路径应保持原样不变
func TestLong(t *testing.T) {
	// 测试短路径转换为完整路径
	d := filepath.FromSlash("~/documents/file.txt")
	require.Equal(t, filepath.Join(Dir(), "documents", "file.txt"), Long(d))

	// 测试非短路径的绝对路径（应保持原样）
	ad := filepath.FromSlash("/absolute/path/file.txt")
	require.Equal(t, ad, Long(ad))
}
