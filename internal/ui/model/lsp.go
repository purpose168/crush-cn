package model

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/powernap/pkg/lsp/protocol"
	"github.com/purpose168/crush-cn/internal/app"
	"github.com/purpose168/crush-cn/internal/lsp"
	"github.com/purpose168/crush-cn/internal/ui/common"
	"github.com/purpose168/crush-cn/internal/ui/styles"
)

// LSPInfo 包装LSP客户端信息，按严重程度分类的诊断计数。
type LSPInfo struct {
	app.LSPClientInfo
	Diagnostics map[protocol.DiagnosticSeverity]int
}

// lspInfo 渲染LSP状态部分，显示活动的LSP客户端及其诊断计数。
func (m *UI) lspInfo(width, maxItems int, isSection bool) string {
	t := m.com.Styles

	states := slices.SortedFunc(maps.Values(m.lspStates), func(a, b app.LSPClientInfo) int {
		return strings.Compare(a.Name, b.Name)
	})

	var lsps []LSPInfo
	for _, state := range states {
		client, ok := m.com.App.LSPManager.Clients().Get(state.Name)
		if !ok {
			continue
		}
		counts := client.GetDiagnosticCounts()
		lspErrs := map[protocol.DiagnosticSeverity]int{
			protocol.SeverityError:       counts.Error,
			protocol.SeverityWarning:     counts.Warning,
			protocol.SeverityHint:        counts.Hint,
			protocol.SeverityInformation: counts.Information,
		}

		lsps = append(lsps, LSPInfo{LSPClientInfo: state, Diagnostics: lspErrs})
	}

	title := t.Subtle.Render("语言服务器")
	if isSection {
		title = common.Section(t, title, width)
	}
	list := t.Subtle.Render("无")
	if len(lsps) > 0 {
		list = lspList(t, lsps, width, maxItems)
	}

	return lipgloss.NewStyle().Width(width).Render(fmt.Sprintf("%s\n\n%s", title, list))
}

// lspDiagnostics 使用适当的图标和颜色格式化诊断计数。
func lspDiagnostics(t *styles.Styles, diagnostics map[protocol.DiagnosticSeverity]int) string {
	var errs []string
	if diagnostics[protocol.SeverityError] > 0 {
		errs = append(errs, t.LSP.ErrorDiagnostic.Render(fmt.Sprintf("%s%d", styles.LSPErrorIcon, diagnostics[protocol.SeverityError])))
	}
	if diagnostics[protocol.SeverityWarning] > 0 {
		errs = append(errs, t.LSP.WarningDiagnostic.Render(fmt.Sprintf("%s%d", styles.LSPWarningIcon, diagnostics[protocol.SeverityWarning])))
	}
	if diagnostics[protocol.SeverityHint] > 0 {
		errs = append(errs, t.LSP.HintDiagnostic.Render(fmt.Sprintf("%s%d", styles.LSPHintIcon, diagnostics[protocol.SeverityHint])))
	}
	if diagnostics[protocol.SeverityInformation] > 0 {
		errs = append(errs, t.LSP.InfoDiagnostic.Render(fmt.Sprintf("%s%d", styles.LSPInfoIcon, diagnostics[protocol.SeverityInformation])))
	}
	return strings.Join(errs, " ")
}

// lspList 渲染LSP客户端列表，显示其状态和诊断信息，
// 如有需要则截断至maxItems项。
func lspList(t *styles.Styles, lsps []LSPInfo, width, maxItems int) string {
	if maxItems <= 0 {
		return ""
	}
	var renderedLsps []string
	for _, l := range lsps {
		var icon string
		title := l.Name
		var description string
		var diagnostics string
		switch l.State {
		case lsp.StateStopped:
			icon = t.ItemOfflineIcon.Foreground(t.Muted.GetBackground()).String()
			description = t.Subtle.Render("已停止")
		case lsp.StateStarting:
			icon = t.ItemBusyIcon.String()
			description = t.Subtle.Render("启动中...")
		case lsp.StateReady:
			icon = t.ItemOnlineIcon.String()
			diagnostics = lspDiagnostics(t, l.Diagnostics)
		case lsp.StateError:
			icon = t.ItemErrorIcon.String()
			description = t.Subtle.Render("错误")
			if l.Error != nil {
				description = t.Subtle.Render(fmt.Sprintf("错误: %s", l.Error.Error()))
			}
		case lsp.StateDisabled:
			icon = t.ItemOfflineIcon.Foreground(t.Muted.GetBackground()).String()
			description = t.Subtle.Render("已禁用")
		default:
			icon = t.ItemOfflineIcon.String()
		}
		renderedLsps = append(renderedLsps, common.Status(t, common.StatusOpts{
			Icon:         icon,
			Title:        title,
			Description:  description,
			ExtraContent: diagnostics,
		}, width))
	}

	if len(renderedLsps) > maxItems {
		visibleItems := renderedLsps[:maxItems-1]
		remaining := len(renderedLsps) - maxItems
		visibleItems = append(visibleItems, t.Subtle.Render(fmt.Sprintf("以及其余 %d 项", remaining)))
		return lipgloss.JoinVertical(lipgloss.Left, visibleItems...)
	}
	return lipgloss.JoinVertical(lipgloss.Left, renderedLsps...)
}
