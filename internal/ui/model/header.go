package model

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/csync"
	"github.com/charmbracelet/crush/internal/fsext"
	"github.com/charmbracelet/crush/internal/lsp"
	"github.com/charmbracelet/crush/internal/session"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/styles"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

const (
	headerDiag     = "╱" // 头部斜杠字符
	minHeaderDiags = 3   // 头部斜杠的最小数量
	leftPadding    = 1   // 左侧填充
	rightPadding   = 1   // 右侧填充
)

// header 表示应用程序的头部模型
type header struct {
	// cached logo and compact logo 缓存的标志和紧凑标志
	logo        string // 标志
	compactLogo string // 紧凑标志

	com     *common.Common // 公共组件
	width   int            // 宽度
	compact bool           // 是否紧凑
}

// newHeader 创建一个新的头部模型
func newHeader(com *common.Common) *header {
	h := &header{
		com: com,
	}
	t := com.Styles
	h.compactLogo = t.Header.Charm.Render("Charm™") + " " +
		styles.ApplyBoldForegroundGrad(t, "CRUSH", t.Secondary, t.Primary) + " "
	return h
}

// drawHeader 为给定会话绘制头部
func (h *header) drawHeader(
	scr uv.Screen,
	area uv.Rectangle,
	session *session.Session,
	compact bool,
	detailsOpen bool,
	width int,
) {
	t := h.com.Styles
	if width != h.width || compact != h.compact {
		h.logo = renderLogo(h.com.Styles, compact, width)
	}

	h.width = width
	h.compact = compact

	if !compact || session == nil || h.com.App == nil {
		uv.NewStyledString(h.logo).Draw(scr, area)
		return
	}

	if session.ID == "" {
		return
	}

	var b strings.Builder
	b.WriteString(h.compactLogo)

	availDetailWidth := width - leftPadding - rightPadding - lipgloss.Width(b.String()) - minHeaderDiags
	details := renderHeaderDetails(
		h.com,
		session,
		h.com.App.LSPManager.Clients(),
		detailsOpen,
		availDetailWidth,
	)

	remainingWidth := width -
		lipgloss.Width(b.String()) -
		lipgloss.Width(details) -
		leftPadding -
		rightPadding

	if remainingWidth > 0 {
		b.WriteString(t.Header.Diagonals.Render(
			strings.Repeat(headerDiag, max(minHeaderDiags, remainingWidth)),
		))
		b.WriteString(" ")
	}

	b.WriteString(details)

	view := uv.NewStyledString(
		t.Base.Padding(0, rightPadding, 0, leftPadding).Render(b.String()))
	view.Draw(scr, area)
}

// renderHeaderDetails 渲染头部的详细信息部分
func renderHeaderDetails(
	com *common.Common,
	session *session.Session,
	lspClients *csync.Map[string, *lsp.Client],
	detailsOpen bool,
	availWidth int,
) string {
	t := com.Styles

	var parts []string

	errorCount := 0
	for l := range lspClients.Seq() {
		errorCount += l.GetDiagnosticCounts().Error
	}

	if errorCount > 0 {
		parts = append(parts, t.LSP.ErrorDiagnostic.Render(fmt.Sprintf("%s%d", styles.LSPErrorIcon, errorCount)))
	}

	agentCfg := com.Config().Agents[config.AgentCoder]
	model := com.Config().GetModelByType(agentCfg.Model)
	percentage := (float64(session.CompletionTokens+session.PromptTokens) / float64(model.ContextWindow)) * 100
	formattedPercentage := t.Header.Percentage.Render(fmt.Sprintf("%d%%", int(percentage)))
	parts = append(parts, formattedPercentage)

	const keystroke = "ctrl+d"
	if detailsOpen {
		parts = append(parts, t.Header.Keystroke.Render(keystroke)+t.Header.KeystrokeTip.Render(" close"))
	} else {
		parts = append(parts, t.Header.Keystroke.Render(keystroke)+t.Header.KeystrokeTip.Render(" open "))
	}

	dot := t.Header.Separator.Render(" • ")
	metadata := strings.Join(parts, dot)
	metadata = dot + metadata

	const dirTrimLimit = 4
	cfg := com.Config()
	cwd := fsext.DirTrim(fsext.PrettyPath(cfg.WorkingDir()), dirTrimLimit)
	cwd = ansi.Truncate(cwd, max(0, availWidth-lipgloss.Width(metadata)), "…")
	cwd = t.Header.WorkingDir.Render(cwd)

	return cwd + metadata
}
