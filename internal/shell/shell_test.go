package shell

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// 基准测试以测量CPU效率
func BenchmarkShellQuickCommands(b *testing.B) {
	shell := NewShell(&Options{WorkingDir: b.TempDir()})

	b.ReportAllocs()

	for b.Loop() {
		_, _, err := shell.Exec(b.Context(), "echo test")
		exitCode := ExitCode(err)
		if err != nil || exitCode != 0 {
			b.Fatalf("命令执行失败: %v, 退出代码: %d", err, exitCode)
		}
	}
}

func TestTestTimeout(t *testing.T) {
	// XXX(@andreynering): 这在Windows上会失败。如果可能的话，请解决此问题。
	if runtime.GOOS == "windows" {
		t.Skip("在Windows上跳过测试")
	}

	ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
	t.Cleanup(cancel)

	shell := NewShell(&Options{WorkingDir: t.TempDir()})
	_, _, err := shell.Exec(ctx, "sleep 10")
	if status := ExitCode(err); status == 0 {
		t.Fatalf("预期非零退出状态，但得到 %d", status)
	}
	if !IsInterrupt(err) {
		t.Fatalf("预期命令被中断，但实际未被中断")
	}
	if err == nil {
		t.Fatalf("预期由于超时产生错误，但没有错误")
	}
}

func TestTestCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // 立即取消上下文

	shell := NewShell(&Options{WorkingDir: t.TempDir()})
	_, _, err := shell.Exec(ctx, "sleep 10")
	if status := ExitCode(err); status == 0 {
		t.Fatalf("预期非零退出状态，但得到 %d", status)
	}
	if !IsInterrupt(err) {
		t.Fatalf("预期命令被中断，但实际未被中断")
	}
	if err == nil {
		t.Fatalf("预期由于取消产生错误，但没有错误")
	}
}

func TestRunCommandError(t *testing.T) {
	shell := NewShell(&Options{WorkingDir: t.TempDir()})
	_, _, err := shell.Exec(t.Context(), "nopenopenope")
	if status := ExitCode(err); status == 0 {
		t.Fatalf("预期非零退出状态，但得到 %d", status)
	}
	if IsInterrupt(err) {
		t.Fatalf("预期命令不被中断，但实际被中断了")
	}
	if err == nil {
		t.Fatalf("预期产生错误，但得到nil")
	}
}

func TestRunContinuity(t *testing.T) {
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	shell := NewShell(&Options{WorkingDir: tempDir1})
	if _, _, err := shell.Exec(t.Context(), "export FOO=bar"); err != nil {
		t.Fatalf("设置环境变量失败: %v", err)
	}
	if _, _, err := shell.Exec(t.Context(), "cd "+filepath.ToSlash(tempDir2)); err != nil {
		t.Fatalf("更改目录失败: %v", err)
	}
	out, _, err := shell.Exec(t.Context(), "echo $FOO ; pwd")
	if err != nil {
		t.Fatalf("执行echo命令失败: %v", err)
	}
	expect := "bar\n" + tempDir2 + "\n"
	if out != expect {
		t.Fatalf("预期输出 %q，但得到 %q", expect, out)
	}
}

func TestCrossPlatformExecution(t *testing.T) {
	shell := NewShell(&Options{WorkingDir: "."})
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	// 测试一个在所有平台上都应该能运行的简单命令
	stdout, stderr, err := shell.Exec(ctx, "echo hello")
	if err != nil {
		t.Fatalf("Echo命令执行失败: %v, 标准错误输出: %s", err, stderr)
	}

	if stdout == "" {
		t.Error("Echo命令没有产生输出")
	}

	// 无论在哪个平台上，输出都应该包含"hello"
	if !strings.Contains(strings.ToLower(stdout), "hello") {
		t.Errorf("Echo输出应该包含'hello'，但得到: %q", stdout)
	}
}
