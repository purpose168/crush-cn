package model

import (
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/agent"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/ultraviolet/layout"
)

// selectedLargeModel 返回从代理协调器中当前选中的大语言模型（如果存在）。
func (m *UI) selectedLargeModel() *agent.Model {
	if m.com.App.AgentCoordinator != nil {
		model := m.com.App.AgentCoordinator.Model()
		return &model
	}
	return nil
}

// landingView 渲染登录页面视图，在两列布局中显示当前工作目录、模型信息和LSP/MCP状态。
func (m *UI) landingView() string {
	t := m.com.Styles
	width := m.layout.main.Dx()
	cwd := common.PrettyPath(t, m.com.Config().WorkingDir(), width)

	parts := []string{
		cwd,
	}

	parts = append(parts, "", m.modelInfo(width))
	infoSection := lipgloss.JoinVertical(lipgloss.Left, parts...)

	_, remainingHeightArea := layout.SplitVertical(m.layout.main, layout.Fixed(lipgloss.Height(infoSection)+1))

	mcpLspSectionWidth := min(30, (width-1)/2)

	lspSection := m.lspInfo(mcpLspSectionWidth, max(1, remainingHeightArea.Dy()), false)
	mcpSection := m.mcpInfo(mcpLspSectionWidth, max(1, remainingHeightArea.Dy()), false)

	content := lipgloss.JoinHorizontal(lipgloss.Left, lspSection, " ", mcpSection)

	return lipgloss.NewStyle().
		Width(width).
		Height(m.layout.main.Dy() - 1).
		PaddingTop(1).
		Render(
			lipgloss.JoinVertical(lipgloss.Left, infoSection, "", content),
		)
}
