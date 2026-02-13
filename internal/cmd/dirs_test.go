package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// init 函数用于设置测试环境变量
// 1. 设置 XDG_CONFIG_HOME 为 /tmp/fakeconfig
// 2. 设置 XDG_DATA_HOME 为 /tmp/fakedata
// 3. 取消设置 CRUSH_GLOBAL_CONFIG 环境变量
// 4. 取消设置 CRUSH_GLOBAL_DATA 环境变量
func init() {
	os.Setenv("XDG_CONFIG_HOME", "/tmp/fakeconfig")
	os.Setenv("XDG_DATA_HOME", "/tmp/fakedata")
	os.Unsetenv("CRUSH_GLOBAL_CONFIG")
	os.Unsetenv("CRUSH_GLOBAL_DATA")
}

// TestDirs 测试 dirs 命令的输出
// 该命令应输出配置目录和数据目录的路径
func TestDirs(t *testing.T) {
	// 创建一个缓冲区来捕获命令输出
	var b bytes.Buffer
	// 设置命令的标准输出
	dirsCmd.SetOut(&b)
	// 设置命令的标准错误
	dirsCmd.SetErr(&b)
	// 设置命令的标准输入
	dirsCmd.SetIn(bytes.NewReader(nil))
	// 运行命令
	dirsCmd.Run(dirsCmd, nil)
	// 构建期望的输出
	expected := filepath.FromSlash("/tmp/fakeconfig/crush") + "\n" +
		filepath.FromSlash("/tmp/fakedata/crush") + "\n"
	// 验证命令输出是否与期望一致
	require.Equal(t, expected, b.String())
}

// TestConfigDir 测试 configDir 命令的输出
// 该命令应输出配置目录的路径
func TestConfigDir(t *testing.T) {
	// 创建一个缓冲区来捕获命令输出
	var b bytes.Buffer
	// 设置命令的标准输出
	configDirCmd.SetOut(&b)
	// 设置命令的标准错误
	configDirCmd.SetErr(&b)
	// 设置命令的标准输入
	configDirCmd.SetIn(bytes.NewReader(nil))
	// 运行命令
	configDirCmd.Run(configDirCmd, nil)
	// 构建期望的输出
	expected := filepath.FromSlash("/tmp/fakeconfig/crush") + "\n"
	// 验证命令输出是否与期望一致
	require.Equal(t, expected, b.String())
}

// TestDataDir 测试 dataDir 命令的输出
// 该命令应输出数据目录的路径
func TestDataDir(t *testing.T) {
	// 创建一个缓冲区来捕获命令输出
	var b bytes.Buffer
	// 设置命令的标准输出
	dataDirCmd.SetOut(&b)
	// 设置命令的标准错误
	dataDirCmd.SetErr(&b)
	// 设置命令的标准输入
	dataDirCmd.SetIn(bytes.NewReader(nil))
	// 运行命令
	dataDirCmd.Run(dataDirCmd, nil)
	// 构建期望的输出
	expected := filepath.FromSlash("/tmp/fakedata/crush") + "\n"
	// 验证命令输出是否与期望一致
	require.Equal(t, expected, b.String())
}
