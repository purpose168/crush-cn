package tools

import (
	"context"
	"testing"
	"time"

	"github.com/purpose168/crush-cn/internal/shell"
	"github.com/stretchr/testify/require"
)

func TestBackgroundShell_Integration(t *testing.T) {
	t.Parallel()

	workingDir := t.TempDir()
	ctx := context.Background()

	// 启动一个后台 shell
	bgManager := shell.GetBackgroundShellManager()
	bgShell, err := bgManager.Start(ctx, workingDir, nil, "echo 'hello background' && echo 'done'", "")
	require.NoError(t, err)
	require.NotEmpty(t, bgShell.ID)

	// 等待完成
	bgShell.Wait()

	// 检查最终输出
	stdout, stderr, done, err := bgShell.GetOutput()
	require.NoError(t, err)
	require.Contains(t, stdout, "hello background")
	require.Contains(t, stdout, "done")
	require.True(t, done)
	require.Empty(t, stderr)

	// 清理
	bgManager.Kill(bgShell.ID)
}

func TestBackgroundShell_Kill(t *testing.T) {
	t.Parallel()

	workingDir := t.TempDir()
	ctx := context.Background()

	// 启动一个长时间运行的后台 shell
	bgManager := shell.GetBackgroundShellManager()
	bgShell, err := bgManager.Start(ctx, workingDir, nil, "sleep 100", "")
	require.NoError(t, err)

	// 终止它
	err = bgManager.Kill(bgShell.ID)
	require.NoError(t, err)

	// 验证它已经不存在
	_, ok := bgManager.Get(bgShell.ID)
	require.False(t, ok)

	// 验证 shell 已经完成
	require.True(t, bgShell.IsDone())
}

func TestBackgroundShell_MultipleOutputCalls(t *testing.T) {
	t.Parallel()

	workingDir := t.TempDir()
	ctx := context.Background()

	// 启动一个后台 shell
	bgManager := shell.GetBackgroundShellManager()
	bgShell, err := bgManager.Start(ctx, workingDir, nil, "echo 'step 1' && echo 'step 2' && echo 'step 3'", "")
	require.NoError(t, err)
	defer bgManager.Kill(bgShell.ID)

	// 检查我们可以在运行时多次调用 GetOutput
	for range 5 {
		_, _, done, _ := bgShell.GetOutput()
		if done {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// 等待完成
	bgShell.Wait()

	// 完成后多次调用应该返回相同的结果
	stdout1, _, done1, _ := bgShell.GetOutput()
	require.True(t, done1)
	require.Contains(t, stdout1, "step 1")
	require.Contains(t, stdout1, "step 2")
	require.Contains(t, stdout1, "step 3")

	stdout2, _, done2, _ := bgShell.GetOutput()
	require.True(t, done2)
	require.Equal(t, stdout1, stdout2, "多次 GetOutput 调用应该返回相同的结果")
}

func TestBackgroundShell_EmptyOutput(t *testing.T) {
	t.Parallel()

	workingDir := t.TempDir()
	ctx := context.Background()

	// 启动一个无输出的后台 shell
	bgManager := shell.GetBackgroundShellManager()
	bgShell, err := bgManager.Start(ctx, workingDir, nil, "sleep 0.1", "")
	require.NoError(t, err)
	defer bgManager.Kill(bgShell.ID)

	// 等待完成
	bgShell.Wait()

	stdout, stderr, done, err := bgShell.GetOutput()
	require.NoError(t, err)
	require.Empty(t, stdout)
	require.Empty(t, stderr)
	require.True(t, done)
}

func TestBackgroundShell_ExitCode(t *testing.T) {
	t.Parallel()

	workingDir := t.TempDir()
	ctx := context.Background()

	// 启动一个以非零代码退出的后台 shell
	bgManager := shell.GetBackgroundShellManager()
	bgShell, err := bgManager.Start(ctx, workingDir, nil, "echo 'failing' && exit 42", "")
	require.NoError(t, err)
	defer bgManager.Kill(bgShell.ID)

	// 等待完成
	bgShell.Wait()

	stdout, _, done, execErr := bgShell.GetOutput()
	require.True(t, done)
	require.Contains(t, stdout, "failing")
	require.Error(t, execErr)

	exitCode := shell.ExitCode(execErr)
	require.Equal(t, 42, exitCode)
}

func TestBackgroundShell_WithBlockFuncs(t *testing.T) {
	t.Parallel()

	workingDir := t.TempDir()
	ctx := context.Background()

	blockFuncs := []shell.BlockFunc{
		shell.CommandsBlocker([]string{"curl", "wget"}),
	}

	// 启动一个包含被阻止命令的后台 shell
	bgManager := shell.GetBackgroundShellManager()
	bgShell, err := bgManager.Start(ctx, workingDir, blockFuncs, "curl example.com", "")
	require.NoError(t, err)
	defer bgManager.Kill(bgShell.ID)

	// 等待完成
	bgShell.Wait()

	stdout, stderr, done, execErr := bgShell.GetOutput()
	require.True(t, done)

	// 命令应该已被阻止，检查 stderr 或错误
	if execErr != nil {
		// 错误可能包含消息
		require.Contains(t, execErr.Error(), "not allowed")
	} else {
		// 或者消息可能在 stderr 中
		output := stdout + stderr
		require.Contains(t, output, "not allowed")
	}
}

func TestBackgroundShell_StdoutAndStderr(t *testing.T) {
	t.Parallel()

	workingDir := t.TempDir()
	ctx := context.Background()

	// 启动一个同时有 stdout 和 stderr 输出的后台 shell
	bgManager := shell.GetBackgroundShellManager()
	bgShell, err := bgManager.Start(ctx, workingDir, nil, "echo 'stdout message' && echo 'stderr message' >&2", "")
	require.NoError(t, err)
	defer bgManager.Kill(bgShell.ID)

	// 等待完成
	bgShell.Wait()

	stdout, stderr, done, err := bgShell.GetOutput()
	require.NoError(t, err)
	require.True(t, done)
	require.Contains(t, stdout, "stdout message")
	require.Contains(t, stderr, "stderr message")
}

func TestBackgroundShell_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	workingDir := t.TempDir()
	ctx := context.Background()

	// 启动一个后台 shell
	bgManager := shell.GetBackgroundShellManager()
	bgShell, err := bgManager.Start(ctx, workingDir, nil, "for i in 1 2 3 4 5; do echo \"line $i\"; sleep 0.05; done", "")
	require.NoError(t, err)
	defer bgManager.Kill(bgShell.ID)

	// 从多个 goroutine 并发访问输出
	done := make(chan struct{})
	errors := make(chan error, 10)

	for range 10 {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					_, _, _, err := bgShell.GetOutput()
					if err != nil {
						errors <- err
					}
					dir := bgShell.WorkingDir
					if dir == "" {
						errors <- err
					}
					time.Sleep(10 * time.Millisecond)
				}
			}
		}()
	}

	// 让它运行一段时间
	time.Sleep(300 * time.Millisecond)
	close(done)

	// 检查是否有任何错误
	select {
	case err := <-errors:
		t.Fatalf("并发访问导致错误: %v", err)
	case <-time.After(100 * time.Millisecond):
		// 无错误 - 成功
	}
}

func TestBackgroundShell_List(t *testing.T) {
	t.Parallel()

	workingDir := t.TempDir()
	ctx := context.Background()

	bgManager := shell.GetBackgroundShellManager()

	// 启动多个后台 shell
	shells := make([]*shell.BackgroundShell, 3)
	for i := range 3 {
		bgShell, err := bgManager.Start(ctx, workingDir, nil, "sleep 1", "")
		require.NoError(t, err)
		shells[i] = bgShell
	}

	// 获取列表
	ids := bgManager.List()

	// 验证所有 shell 都在列表中
	for _, sh := range shells {
		require.Contains(t, ids, sh.ID, "Shell %s not found in list", sh.ID)
	}

	// 清理
	for _, sh := range shells {
		bgManager.Kill(sh.ID)
	}
}

func TestBackgroundShell_AutoBackground(t *testing.T) {
	t.Parallel()

	workingDir := t.TempDir()
	ctx := context.Background()

	// 测试快速命令是否同步完成
	t.Run("quick command completes synchronously", func(t *testing.T) {
		t.Parallel()
		bgManager := shell.GetBackgroundShellManager()
		bgShell, err := bgManager.Start(ctx, workingDir, nil, "echo 'quick'", "")
		require.NoError(t, err)

		// 等待阈值时间
		time.Sleep(5 * time.Second)

		// 现在应该已完成
		stdout, stderr, done, err := bgShell.GetOutput()
		require.NoError(t, err)
		require.True(t, done, "快速命令应该已完成")
		require.Contains(t, stdout, "quick")
		require.Empty(t, stderr)

		// 清理
		bgManager.Kill(bgShell.ID)
	})

	// 测试长时间命令是否保持在后台运行
	t.Run("long command stays in background", func(t *testing.T) {
		t.Parallel()
		bgManager := shell.GetBackgroundShellManager()
		bgShell, err := bgManager.Start(ctx, workingDir, nil, "sleep 20 && echo '20 seconds completed'", "")
		require.NoError(t, err)
		defer bgManager.Kill(bgShell.ID)

		// 等待阈值时间
		time.Sleep(5 * time.Second)

		// 应该仍在运行
		stdout, stderr, done, err := bgShell.GetOutput()
		require.NoError(t, err)
		require.False(t, done, "长时间命令应该仍在运行")
		require.Empty(t, stdout, "睡眠命令尚未有输出")
		require.Empty(t, stderr)

		// 验证我们可以从管理器中获取 shell
		retrieved, ok := bgManager.Get(bgShell.ID)
		require.True(t, ok, "应该能够检索后台 shell")
		require.Equal(t, bgShell.ID, retrieved.ID)
	})
}
