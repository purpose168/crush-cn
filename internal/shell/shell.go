// Package shell 提供跨平台的 shell 执行能力。
//
// 本包提供 Shell 实例用于执行命令,每个实例拥有独立的工作目录和环境变量。
// 每次 shell 执行都是相互独立的。
//
// WINDOWS 兼容性:
// 本实现即使在 Windows 上也提供 POSIX shell 仿真(mvdan.cc/sh/v3)。
// 命令应使用正斜杠(/)作为路径分隔符,以确保在所有平台上正常工作。
package shell

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/charmbracelet/x/exp/slice"
	"mvdan.cc/sh/moreinterp/coreutils"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// ShellType 表示要使用的 shell 类型
type ShellType int

const (
	ShellTypePOSIX ShellType = iota // POSIX shell
	ShellTypeCmd                    // Windows CMD
	ShellTypePowerShell             // PowerShell
)

// Logger 接口用于可选的日志记录
type Logger interface {
	InfoPersist(msg string, keysAndValues ...any)
}

// noopLogger 是一个不执行任何操作的日志记录器
type noopLogger struct{}

func (noopLogger) InfoPersist(msg string, keysAndValues ...any) {}

// BlockFunc 是一个函数,用于确定是否应该阻止某个命令执行
type BlockFunc func(args []string) bool

// Shell 提供跨平台的 shell 执行功能,支持可选的状态持久化
type Shell struct {
	env        []string // 环境变量
	cwd        string   // 当前工作目录
	mu         sync.Mutex // 互斥锁,用于保护并发访问
	logger     Logger   // 日志记录器
	blockFuncs []BlockFunc // 命令阻止函数列表
}

// Options 用于创建新的 shell 实例的配置选项
type Options struct {
	WorkingDir string   // 工作目录
	Env        []string // 环境变量
	Logger     Logger   // 日志记录器
	BlockFuncs []BlockFunc // 命令阻止函数列表
}

// NewShell 使用给定的选项创建一个新的 shell 实例
func NewShell(opts *Options) *Shell {
	if opts == nil {
		opts = &Options{}
	}

	cwd := opts.WorkingDir
	if cwd == "" {
		// 如果未指定工作目录,使用当前目录
		cwd, _ = os.Getwd()
	}

	env := opts.Env
	if env == nil {
		// 如果未指定环境变量,使用系统环境变量
		env = os.Environ()
	}

	logger := opts.Logger
	if logger == nil {
		// 如果未指定日志记录器,使用空日志记录器
		logger = noopLogger{}
	}

	return &Shell{
		cwd:        cwd,
		env:        env,
		logger:     logger,
		blockFuncs: opts.BlockFuncs,
	}
}

// Exec 在 shell 中执行命令,返回标准输出、标准错误和错误信息
func (s *Shell) Exec(ctx context.Context, command string) (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.exec(ctx, command)
}

// ExecStream 在 shell 中执行命令,并将输出流式传输到提供的写入器
func (s *Shell) ExecStream(ctx context.Context, command string, stdout, stderr io.Writer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.execStream(ctx, command, stdout, stderr)
}

// GetWorkingDir 返回当前工作目录
func (s *Shell) GetWorkingDir() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cwd
}

// SetWorkingDir 设置工作目录
func (s *Shell) SetWorkingDir(dir string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 验证目录是否存在
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("目录不存在: %w", err)
	}

	s.cwd = dir
	return nil
}

// GetEnv 返回环境变量的副本
func (s *Shell) GetEnv() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	env := make([]string, len(s.env))
	copy(env, s.env)
	return env
}

// SetEnv 设置一个环境变量
func (s *Shell) SetEnv(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 更新或添加环境变量
	keyPrefix := key + "="
	for i, env := range s.env {
		if strings.HasPrefix(env, keyPrefix) {
			s.env[i] = keyPrefix + value
			return
		}
	}
	s.env = append(s.env, keyPrefix+value)
}

// SetBlockFuncs 设置 shell 的命令阻止函数
func (s *Shell) SetBlockFuncs(blockFuncs []BlockFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blockFuncs = blockFuncs
}

// CommandsBlocker 创建一个 BlockFunc,用于阻止精确匹配的命令
func CommandsBlocker(cmds []string) BlockFunc {
	bannedSet := make(map[string]struct{})
	for _, cmd := range cmds {
		bannedSet[cmd] = struct{}{}
	}

	return func(args []string) bool {
		if len(args) == 0 {
			return false
		}
		_, ok := bannedSet[args[0]]
		return ok
	}
}

// ArgumentsBlocker 创建一个 BlockFunc,用于阻止特定的子命令
func ArgumentsBlocker(cmd string, args []string, flags []string) BlockFunc {
	return func(parts []string) bool {
		if len(parts) == 0 || parts[0] != cmd {
			return false
		}

		argParts, flagParts := splitArgsFlags(parts[1:])
		if len(argParts) < len(args) || len(flagParts) < len(flags) {
			return false
		}

		argsMatch := slices.Equal(argParts[:len(args)], args)
		flagsMatch := slice.IsSubset(flags, flagParts)

		return argsMatch && flagsMatch
	}
}

func splitArgsFlags(parts []string) (args []string, flags []string) {
	args = make([]string, 0, len(parts))
	flags = make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.HasPrefix(part, "-") {
			// 如果存在等号,提取等号前的标志名称
			flag := part
			if idx := strings.IndexByte(part, '='); idx != -1 {
				flag = part[:idx]
			}
			flags = append(flags, flag)
		} else {
			args = append(args, part)
		}
	}
	return args, flags
}

// blockHandler 返回一个命令执行处理器,用于阻止不安全的命令
func (s *Shell) blockHandler() func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
		return func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return next(ctx, args)
			}

			// 检查所有阻止函数,如果任一函数返回 true,则阻止命令执行
			for _, blockFunc := range s.blockFuncs {
				if blockFunc(args) {
					return fmt.Errorf("出于安全原因,不允许执行该命令: %q", args[0])
				}
			}

			return next(ctx, args)
		}
	}
}

// newInterp 使用当前 shell 状态创建一个新的解释器
func (s *Shell) newInterp(stdout, stderr io.Writer) (*interp.Runner, error) {
	return interp.New(
		interp.StdIO(nil, stdout, stderr),
		interp.Interactive(false),
		interp.Env(expand.ListEnviron(s.env...)),
		interp.Dir(s.cwd),
		interp.ExecHandlers(s.execHandlers()...),
	)
}

// updateShellFromRunner 在执行后从解释器更新 shell 状态
func (s *Shell) updateShellFromRunner(runner *interp.Runner) {
	s.cwd = runner.Dir
	s.env = s.env[:0]
	// 从解释器中提取导出的环境变量
	for name, vr := range runner.Vars {
		if vr.Exported {
			s.env = append(s.env, name+"="+vr.Str)
		}
	}
}

// execCommon 是执行命令的共享实现
func (s *Shell) execCommon(ctx context.Context, command string, stdout, stderr io.Writer) error {
	line, err := syntax.NewParser().Parse(strings.NewReader(command), "")
	if err != nil {
		return fmt.Errorf("无法解析命令: %w", err)
	}

	runner, err := s.newInterp(stdout, stderr)
	if err != nil {
		return fmt.Errorf("无法运行命令: %w", err)
	}

	err = runner.Run(ctx, line)
	s.updateShellFromRunner(runner)
	s.logger.InfoPersist("命令执行完成", "command", command, "err", err)
	return err
}

// exec 使用跨平台 shell 解释器执行命令
func (s *Shell) exec(ctx context.Context, command string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	err := s.execCommon(ctx, command, &stdout, &stderr)
	return stdout.String(), stderr.String(), err
}

// execStream 使用 POSIX shell 仿真执行命令,并流式输出结果
func (s *Shell) execStream(ctx context.Context, command string, stdout, stderr io.Writer) error {
	return s.execCommon(ctx, command, stdout, stderr)
}

// execHandlers 返回命令执行处理器列表
func (s *Shell) execHandlers() []func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	handlers := []func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc{
		s.blockHandler(),
	}
	if useGoCoreUtils {
		// 如果启用 Go 核心工具,添加核心工具处理器
		handlers = append(handlers, coreutils.ExecHandler)
	}
	return handlers
}

// IsInterrupt 检查错误是否由于中断引起
func IsInterrupt(err error) bool {
	return errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded)
}

// ExitCode 从错误中提取退出码
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr interp.ExitStatus
	if errors.As(err, &exitErr) {
		return int(exitErr)
	}
	return 1
}
