package model

import (
	"cmp"
	"fmt"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/logo"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/ultraviolet/layout"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// modelInfo 渲染当前模型信息，包括推理
// 设置和上下文使用情况/成本，用于侧边栏。
func (m *UI) modelInfo(width int) string {
	model := m.selectedLargeModel()
	reasoningInfo := ""
	providerName := ""

	if model != nil {
		// 首先获取提供商名称
		providerConfig, ok := m.com.Config().Providers.Get(model.ModelCfg.Provider)
		if ok {
			providerName = providerConfig.Name

			// 仅当模型支持推理时才检查推理设置
			if model.CatwalkCfg.CanReason {
				if len(model.CatwalkCfg.ReasoningLevels) == 0 {
					if model.ModelCfg.Think {
						reasoningInfo = "思考开启"
					} else {
						reasoningInfo = "思考关闭"
					}
				} else {
					formatter := cases.Title(language.English, cases.NoLower)
					reasoningEffort := cmp.Or(model.ModelCfg.ReasoningEffort, model.CatwalkCfg.DefaultReasoningEffort)
					reasoningInfo = formatter.String(fmt.Sprintf("推理强度 %s", reasoningEffort))
				}
			}
		}
	}

	var modelContext *common.ModelContextInfo
	if model != nil && m.session != nil {
		modelContext = &common.ModelContextInfo{
			ContextUsed:  m.session.CompletionTokens + m.session.PromptTokens,
			Cost:         m.session.Cost,
			ModelContext: model.CatwalkCfg.ContextWindow,
		}
	}
	return common.ModelInfo(m.com.Styles, model.CatwalkCfg.Name, providerName, reasoningInfo, modelContext, width)
}

// getDynamicHeightLimits 根据高度返回每个部分要显示的项目数量，
// 因为某些项目比其他项目更重要。
func getDynamicHeightLimits(availableHeight int) (maxFiles, maxLSPs, maxMCPs int) {
	const (
		minItemsPerSection      = 2
		defaultMaxFilesShown    = 10
		defaultMaxLSPsShown     = 8
		defaultMaxMCPsShown     = 8
		minAvailableHeightLimit = 10
	)

	// 如果空间很小，使用最小值
	if availableHeight < minAvailableHeightLimit {
		return minItemsPerSection, minItemsPerSection, minItemsPerSection
	}

	// 在三个部分之间分配可用高度
	// 优先文件，然后是LSP，最后是MCP
	totalSections := 3
	heightPerSection := availableHeight / totalSections

	// 计算每个部分的限制，确保最小值
	maxFiles = max(minItemsPerSection, min(defaultMaxFilesShown, heightPerSection))
	maxLSPs = max(minItemsPerSection, min(defaultMaxLSPsShown, heightPerSection))
	maxMCPs = max(minItemsPerSection, min(defaultMaxMCPsShown, heightPerSection))

	// 如果有多余的空间，首先给文件
	remainingHeight := availableHeight - (maxFiles + maxLSPs + maxMCPs)
	if remainingHeight > 0 {
		extraForFiles := min(remainingHeight, defaultMaxFilesShown-maxFiles)
		maxFiles += extraForFiles
		remainingHeight -= extraForFiles

		if remainingHeight > 0 {
			extraForLSPs := min(remainingHeight, defaultMaxLSPsShown-maxLSPs)
			maxLSPs += extraForLSPs
			remainingHeight -= extraForLSPs

			if remainingHeight > 0 {
				maxMCPs += min(remainingHeight, defaultMaxMCPsShown-maxMCPs)
			}
		}
	}

	return maxFiles, maxLSPs, maxMCPs
}

// sidebar 渲染聊天侧边栏，包含会话标题、工作
// 目录、模型信息、文件列表、LSP状态和MCP状态。
func (m *UI) drawSidebar(scr uv.Screen, area uv.Rectangle) {
	if m.session == nil {
		return
	}

	const logoHeightBreakpoint = 30

	t := m.com.Styles
	width := area.Dx()
	height := area.Dy()

	title := t.Muted.Width(width).MaxHeight(2).Render(m.session.Title)
	cwd := common.PrettyPath(t, m.com.Config().WorkingDir(), width)
	sidebarLogo := m.sidebarLogo
	if height < logoHeightBreakpoint {
		sidebarLogo = logo.SmallRender(m.com.Styles, width)
	}
	blocks := []string{
		sidebarLogo,
		title,
		"",
		cwd,
		"",
		m.modelInfo(width),
		"",
	}

	sidebarHeader := lipgloss.JoinVertical(
		lipgloss.Left,
		blocks...,
	)

	_, remainingHeightArea := layout.SplitVertical(m.layout.sidebar, layout.Fixed(lipgloss.Height(sidebarHeader)))
	remainingHeight := remainingHeightArea.Dy() - 10
	maxFiles, maxLSPs, maxMCPs := getDynamicHeightLimits(remainingHeight)

	lspSection := m.lspInfo(width, maxLSPs, true)
	mcpSection := m.mcpInfo(width, maxMCPs, true)
	filesSection := m.filesInfo(m.com.Config().WorkingDir(), width, maxFiles, true)

	uv.NewStyledString(
		lipgloss.NewStyle().
			MaxWidth(width).
			MaxHeight(height).
			Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					sidebarHeader,
					filesSection,
					"",
					lspSection,
					"",
					mcpSection,
				),
			),
	).Draw(scr, area)
}
