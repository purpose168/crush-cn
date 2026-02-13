package shell

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBackgroundShellManager_Start(t *testing.T) {
	t.Skip("在我弄清楚为什么这个测试不稳定之前跳过")
	t.Parallel()

	ctx := t.Context()
	workingDir := t.TempDir()
	manager := newBackgroundShellManager()

	bgShell, err := manager.Start(ctx, workingDir, nil, "echo 'hello world'", "")
	if err != nil {
		t.Fatalf("启动后台shell失败: %v", err)
	}

	if bgShell.ID == "" {
		t.Error("期望shell ID不为空")
	}

	// 等待命令完成
	bgShell.Wait()

	stdout, stderr, done, err := bgShell.GetOutput()
	if !done {
		t.Error("期望shell已完成")
	}

	if err != nil {
		t.Errorf("期望无错误，但得到: %v", err)
	}

	if !strings.Contains(stdout, "hello world") {
		t.Errorf("期望stdout包含'hello world'，但得到: %s", stdout)
	}

	if stderr != "" {
		t.Errorf("期望stderr为空，但得到: %s", stderr)
	}
}

func TestBackgroundShellManager_Get(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	workingDir := t.TempDir()
	manager := newBackgroundShellManager()

	bgShell, err := manager.Start(ctx, workingDir, nil, "echo 'test'", "")
	if err != nil {
		t.Fatalf("启动后台shell失败: %v", err)
	}

	// 获取shell
	retrieved, ok := manager.Get(bgShell.ID)
	if !ok {
		t.Error("期望找到后台shell")
	}

	if retrieved.ID != bgShell.ID {
		t.Errorf("期望shell ID为%s，但得到%s", bgShell.ID, retrieved.ID)
	}

	// 清理
	manager.Kill(bgShell.ID)
}

func TestBackgroundShellManager_Kill(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	workingDir := t.TempDir()
	manager := newBackgroundShellManager()

	// 启动一个长时间运行的命令
	bgShell, err := manager.Start(ctx, workingDir, nil, "sleep 10", "")
	if err != nil {
		t.Fatalf("启动后台shell失败: %v", err)
	}

	// 终止它
	err = manager.Kill(bgShell.ID)
	if err != nil {
		t.Errorf("终止后台shell失败: %v", err)
	}

	// 验证它已不在管理器中
	_, ok := manager.Get(bgShell.ID)
	if ok {
		t.Error("期望shell在终止后被移除")
	}

	// 验证shell已完成
	if !bgShell.IsDone() {
		t.Error("期望shell在终止后已完成")
	}
}

func TestBackgroundShellManager_KillNonExistent(t *testing.T) {
	t.Parallel()

	manager := newBackgroundShellManager()

	err := manager.Kill("non-existent-id")
	if err == nil {
		t.Error("期望终止不存在的shell时返回错误")
	}
}

func TestBackgroundShell_IsDone(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	workingDir := t.TempDir()
	manager := newBackgroundShellManager()

	bgShell, err := manager.Start(ctx, workingDir, nil, "echo 'quick'", "")
	if err != nil {
		t.Fatalf("启动后台shell失败: %v", err)
	}

	// 稍等一下让命令完成
	time.Sleep(100 * time.Millisecond)

	if !bgShell.IsDone() {
		t.Error("期望shell已完成")
	}

	// 清理
	manager.Kill(bgShell.ID)
}

func TestBackgroundShell_WithBlockFuncs(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	workingDir := t.TempDir()
	manager := newBackgroundShellManager()

	blockFuncs := []BlockFunc{
		CommandsBlocker([]string{"curl", "wget"}),
	}

	bgShell, err := manager.Start(ctx, workingDir, blockFuncs, "curl example.com", "")
	if err != nil {
		t.Fatalf("启动后台shell失败: %v", err)
	}

	// 等待命令完成
	bgShell.Wait()

	stdout, stderr, done, execErr := bgShell.GetOutput()
	if !done {
		t.Error("期望shell已完成")
	}

	// 命令应该被阻止
	output := stdout + stderr
	if !strings.Contains(output, "not allowed") && execErr == nil {
		t.Errorf("期望命令被阻止，得到stdout: %s, stderr: %s, err: %v", stdout, stderr, execErr)
	}

	// 清理
	manager.Kill(bgShell.ID)
}

func TestBackgroundShellManager_List(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("在Windows上跳过不稳定的测试")
	}

	t.Parallel()

	ctx := t.Context()
	workingDir := t.TempDir()
	manager := newBackgroundShellManager()

	// 启动两个shell
	bgShell1, err := manager.Start(ctx, workingDir, nil, "sleep 1", "")
	if err != nil {
		t.Fatalf("启动第一个后台shell失败: %v", err)
	}

	bgShell2, err := manager.Start(ctx, workingDir, nil, "sleep 1", "")
	if err != nil {
		t.Fatalf("启动第二个后台shell失败: %v", err)
	}

	ids := manager.List()

	// 检查两个shell都在列表中
	found1 := false
	found2 := false
	for _, id := range ids {
		if id == bgShell1.ID {
			found1 = true
		}
		if id == bgShell2.ID {
			found2 = true
		}
	}

	if !found1 {
		t.Errorf("期望在列表中找到shell %s", bgShell1.ID)
	}
	if !found2 {
		t.Errorf("期望在列表中找到shell %s", bgShell2.ID)
	}

	// 清理
	manager.Kill(bgShell1.ID)
	manager.Kill(bgShell2.ID)
}

func TestBackgroundShellManager_KillAll(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	workingDir := t.TempDir()
	manager := newBackgroundShellManager()

	// 启动多个长时间运行的shell
	shell1, err := manager.Start(ctx, workingDir, nil, "sleep 10", "")
	if err != nil {
		t.Fatalf("启动shell 1失败: %v", err)
	}

	shell2, err := manager.Start(ctx, workingDir, nil, "sleep 10", "")
	if err != nil {
		t.Fatalf("启动shell 2失败: %v", err)
	}

	shell3, err := manager.Start(ctx, workingDir, nil, "sleep 10", "")
	if err != nil {
		t.Fatalf("启动shell 3失败: %v", err)
	}

	// 验证shell正在运行
	if shell1.IsDone() || shell2.IsDone() || shell3.IsDone() {
		t.Error("shell应该尚未完成")
	}

	// 终止所有shell
	manager.KillAll(t.Context())

	// 验证所有shell已完成
	if !shell1.IsDone() {
		t.Error("shell1在KillAll后应该已完成")
	}
	if !shell2.IsDone() {
		t.Error("shell2在KillAll后应该已完成")
	}
	if !shell3.IsDone() {
		t.Error("shell3在KillAll后应该已完成")
	}

	// 验证它们已从管理器中移除
	if _, ok := manager.Get(shell1.ID); ok {
		t.Error("shell1应该已从管理器中移除")
	}
	if _, ok := manager.Get(shell2.ID); ok {
		t.Error("shell2应该已从管理器中移除")
	}
	if _, ok := manager.Get(shell3.ID); ok {
		t.Error("shell3应该已从管理器中移除")
	}

	// 验证列表为空（或不包含我们的shell）
	ids := manager.List()
	for _, id := range ids {
		if id == shell1.ID || id == shell2.ID || id == shell3.ID {
			t.Errorf("shell %s在KillAll后不应在列表中", id)
		}
	}
}

func TestBackgroundShellManager_KillAll_Timeout(t *testing.T) {
	t.Parallel()

	// XXX: 这里不能使用synctest - 会导致--race触发。

	workingDir := t.TempDir()
	manager := newBackgroundShellManager()

	// 启动一个捕获信号并忽略取消的shell。
	_, err := manager.Start(t.Context(), workingDir, nil, "trap '' TERM INT; sleep 60", "")
	require.NoError(t, err)

	// 短超时以测试超时路径。
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	t.Cleanup(cancel)

	start := time.Now()
	manager.KillAll(ctx)

	elapsed := time.Since(start)

	// 必须在超时后立即返回，而不是挂起60秒。
	require.Less(t, elapsed, 2*time.Second)
}
