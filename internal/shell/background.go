package shell

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/purpose168/crush-cn/internal/csync"
)

const (
	// MaxBackgroundJobs 是允许的最大并发后台任务数
	MaxBackgroundJobs = 50
	// CompletedJobRetentionMinutes 是在自动清理之前保留已完成任务的时长（8小时）
	CompletedJobRetentionMinutes = 8 * 60
)

// syncBuffer 是 bytes.Buffer 的线程安全包装器
type syncBuffer struct {
	buf bytes.Buffer
	mu  sync.RWMutex
}

// Write 向缓冲区写入字节数据，使用写锁保证线程安全
func (sb *syncBuffer) Write(p []byte) (n int, err error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

// WriteString 向缓冲区写入字符串，使用写锁保证线程安全
func (sb *syncBuffer) WriteString(s string) (n int, err error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.WriteString(s)
}

// String 返回缓冲区的字符串内容，使用读锁保证线程安全
func (sb *syncBuffer) String() string {
	sb.mu.RLock()
	defer sb.mu.RUnlock()
	return sb.buf.String()
}

// BackgroundShell 表示在后台运行的 shell
type BackgroundShell struct {
	ID          string          // 任务唯一标识符
	Command     string          // 执行的命令
	Description string          // 任务描述
	Shell       *Shell          // Shell 实例
	WorkingDir  string          // 工作目录
	ctx         context.Context // 上下文，用于取消操作
	cancel      context.CancelFunc // 取消函数
	stdout      *syncBuffer     // 标准输出缓冲区
	stderr      *syncBuffer     // 标准错误输出缓冲区
	done        chan struct{}   // 完成信号通道
	exitErr     error           // 退出错误
	completedAt int64           // 任务完成的 Unix 时间戳（0 表示仍在运行）
}

// BackgroundShellManager 管理后台 shell 实例
type BackgroundShellManager struct {
	shells *csync.Map[string, *BackgroundShell] // 存储所有后台 shell 的并发安全映射
}

var (
	backgroundManager     *BackgroundShellManager
	backgroundManagerOnce sync.Once
	idCounter             atomic.Uint64
)

// newBackgroundShellManager 创建一个新的 BackgroundShellManager 实例
func newBackgroundShellManager() *BackgroundShellManager {
	return &BackgroundShellManager{
		shells: csync.NewMap[string, *BackgroundShell](),
	}
}

// GetBackgroundShellManager 返回单例的后台 shell 管理器
func GetBackgroundShellManager() *BackgroundShellManager {
	backgroundManagerOnce.Do(func() {
		backgroundManager = newBackgroundShellManager()
	})
	return backgroundManager
}

// Start 使用给定命令创建并启动一个新的后台 shell
func (m *BackgroundShellManager) Start(ctx context.Context, workingDir string, blockFuncs []BlockFunc, command string, description string) (*BackgroundShell, error) {
	// 检查任务数量限制
	if m.shells.Len() >= MaxBackgroundJobs {
		return nil, fmt.Errorf("已达到最大后台任务数（%d）。请终止或等待某些任务完成", MaxBackgroundJobs)
	}

	id := fmt.Sprintf("%03X", idCounter.Add(1))

	shell := NewShell(&Options{
		WorkingDir: workingDir,
		BlockFuncs: blockFuncs,
	})

	shellCtx, cancel := context.WithCancel(ctx)

	bgShell := &BackgroundShell{
		ID:          id,
		Command:     command,
		Description: description,
		WorkingDir:  workingDir,
		Shell:       shell,
		ctx:         shellCtx,
		cancel:      cancel,
		stdout:      &syncBuffer{},
		stderr:      &syncBuffer{},
		done:        make(chan struct{}),
	}

	m.shells.Set(id, bgShell)

	// 在 goroutine 中执行命令，避免阻塞主线程
	go func() {
		defer close(bgShell.done)

		err := shell.ExecStream(shellCtx, command, bgShell.stdout, bgShell.stderr)

		bgShell.exitErr = err
		atomic.StoreInt64(&bgShell.completedAt, time.Now().Unix())
	}()

	return bgShell, nil
}

// Get 根据 ID 获取后台 shell
func (m *BackgroundShellManager) Get(id string) (*BackgroundShell, bool) {
	return m.shells.Get(id)
}

// Remove 从管理器中移除后台 shell，但不终止它
// 当 shell 已完成且您只想清理跟踪信息时，这很有用
func (m *BackgroundShellManager) Remove(id string) error {
	_, ok := m.shells.Take(id)
	if !ok {
		return fmt.Errorf("未找到后台 shell: %s", id)
	}
	return nil
}

// Kill 根据 ID 终止后台 shell
func (m *BackgroundShellManager) Kill(id string) error {
	shell, ok := m.shells.Take(id)
	if !ok {
		return fmt.Errorf("未找到后台 shell: %s", id)
	}

	shell.cancel()
	<-shell.done
	return nil
}

// BackgroundShellInfo 包含后台 shell 的信息
type BackgroundShellInfo struct {
	ID          string // 任务 ID
	Command     string // 执行的命令
	Description string // 任务描述
}

// List 返回所有后台 shell 的 ID 列表
func (m *BackgroundShellManager) List() []string {
	ids := make([]string, 0, m.shells.Len())
	for id := range m.shells.Seq2() {
		ids = append(ids, id)
	}
	return ids
}

// Cleanup 移除已完成超过保留期的任务
func (m *BackgroundShellManager) Cleanup() int {
	now := time.Now().Unix()
	retentionSeconds := int64(CompletedJobRetentionMinutes * 60)

	var toRemove []string
	for shell := range m.shells.Seq() {
		completedAt := atomic.LoadInt64(&shell.completedAt)
		if completedAt > 0 && now-completedAt > retentionSeconds {
			toRemove = append(toRemove, shell.ID)
		}
	}

	// 移除所有过期的已完成任务
	for _, id := range toRemove {
		m.Remove(id)
	}

	return len(toRemove)
}

// KillAll 终止所有后台 shell。提供的上下文限制了函数等待每个 shell 退出的时间
func (m *BackgroundShellManager) KillAll(ctx context.Context) {
	shells := slices.Collect(m.shells.Seq())
	m.shells.Reset(map[string]*BackgroundShell{})

	var wg sync.WaitGroup
	for _, shell := range shells {
		wg.Go(func() {
			shell.cancel()
			select {
			case <-shell.done:
				// shell 已正常退出
			case <-ctx.Done():
				// 等待超时
			}
		})
	}
	wg.Wait()
}

// GetOutput 返回后台 shell 的当前输出
func (bs *BackgroundShell) GetOutput() (stdout string, stderr string, done bool, err error) {
	select {
	case <-bs.done:
		// 任务已完成，返回所有输出和错误信息
		return bs.stdout.String(), bs.stderr.String(), true, bs.exitErr
	default:
		// 任务仍在运行，返回当前输出
		return bs.stdout.String(), bs.stderr.String(), false, nil
	}
}

// IsDone 检查后台 shell 是否已完成执行
func (bs *BackgroundShell) IsDone() bool {
	select {
	case <-bs.done:
		return true
	default:
		return false
	}
}

// Wait 阻塞直到后台 shell 完成
func (bs *BackgroundShell) Wait() {
	<-bs.done
}
