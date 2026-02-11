package model

import (
	"fmt"
	"log/slog"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/purpose168/crush-cn/internal/agent"
	"github.com/purpose168/crush-cn/internal/config"
	"github.com/purpose168/crush-cn/internal/home"
	"github.com/purpose168/crush-cn/internal/ui/common"
	"github.com/purpose168/crush-cn/internal/ui/util"
)

// markProjectInitialized 在配置中将当前项目标记为已初始化。
func (m *UI) markProjectInitialized() tea.Msg {
	// TODO: 处理错误以便在tui页脚中显示
	err := config.MarkProjectInitialized(m.com.Config())
	if err != nil {
		slog.Error(err.Error())
	}
	return nil
}

// updateInitializeView 处理项目初始化提示的键盘输入。
func (m *UI) updateInitializeView(msg tea.KeyPressMsg) (cmds []tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Initialize.Enter):
		if m.onboarding.yesInitializeSelected {
			cmds = append(cmds, m.initializeProject())
		} else {
			cmds = append(cmds, m.skipInitializeProject())
		}
	case key.Matches(msg, m.keyMap.Initialize.Switch):
		m.onboarding.yesInitializeSelected = !m.onboarding.yesInitializeSelected
	case key.Matches(msg, m.keyMap.Initialize.Yes):
		cmds = append(cmds, m.initializeProject())
	case key.Matches(msg, m.keyMap.Initialize.No):
		cmds = append(cmds, m.skipInitializeProject())
	}
	return cmds
}

// initializeProject 开始项目初始化并跳转到启动页面视图。
func (m *UI) initializeProject() tea.Cmd {
	// 清除会话
	var cmds []tea.Cmd
	if cmd := m.newSession(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	cfg := m.com.Config()

	initialize := func() tea.Msg {
		initPrompt, err := agent.InitializePrompt(*cfg)
		if err != nil {
			return util.InfoMsg{Type: util.InfoTypeError, Msg: err.Error()}
		}
		return sendMessageMsg{Content: initPrompt}
	}
	// 将项目标记为已初始化
	cmds = append(cmds, initialize, m.markProjectInitialized)

	return tea.Sequence(cmds...)
}

// skipInitializeProject 跳过项目初始化并跳转到启动页面视图。
func (m *UI) skipInitializeProject() tea.Cmd {
	// TODO: 初始化项目
	m.setState(uiLanding, uiFocusEditor)
	// 将项目标记为已初始化
	return m.markProjectInitialized
}

// initializeView 渲染带有是/否按钮的项目初始化提示。
func (m *UI) initializeView() string {
	cfg := m.com.Config()
	s := m.com.Styles.Initialize
	cwd := home.Short(cfg.WorkingDir())
	initFile := cfg.Options.InitializeAs

	header := s.Header.Render("您要初始化这个项目吗？")
	path := s.Accent.PaddingLeft(2).Render(cwd)
	desc := s.Content.Render(fmt.Sprintf("当我初始化您的代码库时，我会检查项目并将结果放入%s文件中作为通用上下文。", initFile))
	hint := s.Content.Render("您也可以随时通过 ") + s.Accent.Render("ctrl+p") + s.Content.Render(" 进行初始化。")
	prompt := s.Content.Render("您现在要初始化吗？")

	buttons := common.ButtonGroup(m.com.Styles, []common.ButtonOpts{
		{Text: "是！", Selected: m.onboarding.yesInitializeSelected},
		{Text: "否", Selected: !m.onboarding.yesInitializeSelected},
	}, " ")

	// 最大宽度60以使文本紧凑
	width := min(m.layout.main.Dx(), 60)

	return lipgloss.NewStyle().
		Width(width).
		Height(m.layout.main.Dy()).
		PaddingBottom(1).
		AlignVertical(lipgloss.Bottom).
		Render(strings.Join(
			[]string{
				header,
				path,
				desc,
				hint,
				prompt,
				buttons,
			},
			"\n\n",
		))
}
