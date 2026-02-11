package model

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/purpose168/crush-cn/internal/agent/tools/mcp"
	"github.com/purpose168/crush-cn/internal/ui/common"
	"github.com/purpose168/crush-cn/internal/ui/styles"
)

// mcpInfo 渲染MCP状态部分，显示活动的MCP客户端及其工具/提示计数。
func (m *UI) mcpInfo(width, maxItems int, isSection bool) string {
	var mcps []mcp.ClientInfo
	t := m.com.Styles

	for _, mcp := range m.com.Config().MCP.Sorted() {
		if state, ok := m.mcpStates[mcp.Name]; ok {
			mcps = append(mcps, state)
		}
	}

	title := t.Subtle.Render("MCP")
	if isSection {
		title = common.Section(t, title, width)
	}
	list := t.Subtle.Render("无")
	if len(mcps) > 0 {
		list = mcpList(t, mcps, width, maxItems)
	}

	return lipgloss.NewStyle().Width(width).Render(fmt.Sprintf("%s\n\n%s", title, list))
}

// mcpCounts 格式化工具、提示和资源的计数以便显示。
func mcpCounts(t *styles.Styles, counts mcp.Counts) string {
	var parts []string
	if counts.Tools > 0 {
		parts = append(parts, t.Subtle.Render(fmt.Sprintf("%d 个工具", counts.Tools)))
	}
	if counts.Prompts > 0 {
		parts = append(parts, t.Subtle.Render(fmt.Sprintf("%d 个提示", counts.Prompts)))
	}
	if counts.Resources > 0 {
		parts = append(parts, t.Subtle.Render(fmt.Sprintf("%d 个资源", counts.Resources)))
	}
	return strings.Join(parts, " ")
}

// mcpList 渲染MCP客户端列表，显示其状态和计数，
// 如有需要则截断至maxItems项。
func mcpList(t *styles.Styles, mcps []mcp.ClientInfo, width, maxItems int) string {
	if maxItems <= 0 {
		return ""
	}
	var renderedMcps []string

	for _, m := range mcps {
		var icon string
		title := m.Name
		var description string
		var extraContent string

		switch m.State {
		case mcp.StateStarting:
			icon = t.ItemBusyIcon.String()
			description = t.Subtle.Render("启动中...")
		case mcp.StateConnected:
			icon = t.ItemOnlineIcon.String()
			extraContent = mcpCounts(t, m.Counts)
		case mcp.StateError:
			icon = t.ItemErrorIcon.String()
			description = t.Subtle.Render("错误")
			if m.Error != nil {
				description = t.Subtle.Render(fmt.Sprintf("错误: %s", m.Error.Error()))
			}
		case mcp.StateDisabled:
			icon = t.ItemOfflineIcon.Foreground(t.Muted.GetBackground()).String()
			description = t.Subtle.Render("已禁用")
		default:
			icon = t.ItemOfflineIcon.String()
		}

		renderedMcps = append(renderedMcps, common.Status(t, common.StatusOpts{
			Icon:         icon,
			Title:        title,
			Description:  description,
			ExtraContent: extraContent,
		}, width))
	}

	if len(renderedMcps) > maxItems {
		visibleItems := renderedMcps[:maxItems-1]
		remaining := len(renderedMcps) - maxItems
		visibleItems = append(visibleItems, t.Subtle.Render(fmt.Sprintf("以及其余 %d 项", remaining)))
		return lipgloss.JoinVertical(lipgloss.Left, visibleItems...)
	}
	return lipgloss.JoinVertical(lipgloss.Left, renderedMcps...)
}
