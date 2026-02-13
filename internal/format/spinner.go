package format

import (
	"context"
	"errors"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/purpose168/crush-cn/internal/ui/anim"
)

// Spinner 封装了 bubbles spinner，用于非交互模式
// done: 用于通知动画完成的通道
// prog: BubbleTea 程序实例
type Spinner struct {
	done chan struct{} // 完成信号通道
	prog *tea.Program  // BubbleTea 程序实例
}

// model 定义了 spinner 的内部模型
type model struct {
	cancel context.CancelFunc // 取消函数，用于中断操作
	anim   *anim.Anim         // 动画实例
}

// Init 初始化 spinner，启动动画
func (m model) Init() tea.Cmd { return m.anim.Start() }

// View 渲染 spinner 的视图
func (m model) View() tea.View { return tea.NewView(m.anim.Render()) }

// Update 实现 tea.Model 接口，处理消息更新
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// 处理按键消息
		switch msg.String() {
		case "ctrl+c", "esc":
			// 用户按下 Ctrl+C 或 ESC 键，取消操作并退出
			m.cancel()
			return m, tea.Quit
		}
	case anim.StepMsg:
		// 处理动画步进消息
		cmd := m.anim.Animate(msg)
		return m, cmd
	}
	return m, nil
}

// NewSpinner 创建一个新的 spinner 实例
// 参数:
//   - ctx: 上下文，用于控制生命周期
//   - cancel: 取消函数，用于中断操作
//   - animSettings: 动画配置参数
// 返回:
//   - *Spinner: 初始化后的 Spinner 实例
func NewSpinner(ctx context.Context, cancel context.CancelFunc, animSettings anim.Settings) *Spinner {
	m := model{
		anim:   anim.New(animSettings),
		cancel: cancel,
	}

	// 创建 BubbleTea 程序，输出到标准错误流
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithContext(ctx))

	return &Spinner{
		prog: p,
		done: make(chan struct{}, 1),
	}
}

// Start 启动 spinner 动画
func (s *Spinner) Start() {
	go func() {
		defer close(s.done)
		_, err := s.prog.Run()
		// 确保清除当前行
		fmt.Fprint(os.Stderr, ansi.EraseEntireLine)
		// 处理错误，忽略取消和中断错误
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, tea.ErrInterrupted) {
			fmt.Fprintf(os.Stderr, "运行 spinner 时出错: %v\n", err)
		}
	}()
}

// Stop 停止 spinner 动画
func (s *Spinner) Stop() {
	s.prog.Quit()
	<-s.done // 等待动画完全停止
}
