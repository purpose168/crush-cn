package shell

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestShellPerformanceComparison(t *testing.T) {
	// 创建一个新的Shell实例，使用临时目录作为工作目录
	shell := NewShell(&Options{WorkingDir: t.TempDir()})

	// 测试快速命令执行性能
	start := time.Now()
	// 执行简单的echo命令，输出"hello"
	stdout, stderr, err := shell.Exec(t.Context(), "echo 'hello'")
	// 获取命令的退出码
	exitCode := ExitCode(err)
	// 计算命令执行耗时
	duration := time.Since(start)

	// 验证命令执行成功，没有错误
	require.NoError(t, err)
	// 验证退出码为0（表示成功）
	require.Equal(t, 0, exitCode)
	// 验证标准输出包含"hello"
	require.Contains(t, stdout, "hello")
	// 验证标准错误输出为空
	require.Empty(t, stderr)

	// 记录快速命令的执行耗时
	t.Logf("快速命令耗时: %v", duration)
}

// 基准测试：测量轮询期间的CPU使用情况
func BenchmarkShellPolling(b *testing.B) {
	// 创建一个新的Shell实例，使用临时目录作为工作目录
	shell := NewShell(&Options{WorkingDir: b.TempDir()})

	// 报告内存分配统计信息
	b.ReportAllocs()

	// 执行基准测试循环
	for b.Loop() {
		// 使用短时间的sleep命令来测量轮询开销
		_, _, err := shell.Exec(b.Context(), "sleep 0.02")
		// 获取命令的退出码
		exitCode := ExitCode(err)
		// 检查命令是否执行成功
		if err != nil || exitCode != 0 {
			// 如果命令失败，终止基准测试并报告错误
			b.Fatalf("命令执行失败: %v, 退出码: %d", err, exitCode)
		}
	}
}
