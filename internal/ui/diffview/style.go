package diffview

import (
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/exp/charmtone"
)

// LineStyle 定义差异视图中给定行类型的样式。
type LineStyle struct {
	LineNumber lipgloss.Style
	Symbol     lipgloss.Style
	Code       lipgloss.Style
}

// Style 定义差异视图的总体样式，包括不同行类型的样式，
// 如分隔符、缺失、相等、插入和删除行。
type Style struct {
	DividerLine LineStyle
	MissingLine LineStyle
	EqualLine   LineStyle
	InsertLine  LineStyle
	DeleteLine  LineStyle
}

// DefaultLightStyle 为差异视图提供默认的浅色主题样式。
func DefaultLightStyle() Style {
	return Style{
		DividerLine: LineStyle{
			LineNumber: lipgloss.NewStyle().
				Foreground(charmtone.Iron).
				Background(charmtone.Thunder),
			Code: lipgloss.NewStyle().
				Foreground(charmtone.Oyster).
				Background(charmtone.Anchovy),
		},
		MissingLine: LineStyle{
			LineNumber: lipgloss.NewStyle().
				Background(charmtone.Ash),
			Code: lipgloss.NewStyle().
				Background(charmtone.Ash),
		},
		EqualLine: LineStyle{
			LineNumber: lipgloss.NewStyle().
				Foreground(charmtone.Charcoal).
				Background(charmtone.Ash),
			Code: lipgloss.NewStyle().
				Foreground(charmtone.Pepper).
				Background(charmtone.Salt),
		},
		InsertLine: LineStyle{
			LineNumber: lipgloss.NewStyle().
				Foreground(charmtone.Turtle).
				Background(lipgloss.Color("#c8e6c9")),
			Symbol: lipgloss.NewStyle().
				Foreground(charmtone.Turtle).
				Background(lipgloss.Color("#e8f5e9")),
			Code: lipgloss.NewStyle().
				Foreground(charmtone.Pepper).
				Background(lipgloss.Color("#e8f5e9")),
		},
		DeleteLine: LineStyle{
			LineNumber: lipgloss.NewStyle().
				Foreground(charmtone.Cherry).
				Background(lipgloss.Color("#ffcdd2")),
			Symbol: lipgloss.NewStyle().
				Foreground(charmtone.Cherry).
				Background(lipgloss.Color("#ffebee")),
			Code: lipgloss.NewStyle().
				Foreground(charmtone.Pepper).
				Background(lipgloss.Color("#ffebee")),
		},
	}
}

// DefaultDarkStyle 为差异视图提供默认的深色主题样式。
func DefaultDarkStyle() Style {
	return Style{
		DividerLine: LineStyle{
			LineNumber: lipgloss.NewStyle().
				Foreground(charmtone.Smoke).
				Background(charmtone.Sapphire),
			Code: lipgloss.NewStyle().
				Foreground(charmtone.Smoke).
				Background(charmtone.Ox),
		},
		MissingLine: LineStyle{
			LineNumber: lipgloss.NewStyle().
				Background(charmtone.Charcoal),
			Code: lipgloss.NewStyle().
				Background(charmtone.Charcoal),
		},
		EqualLine: LineStyle{
			LineNumber: lipgloss.NewStyle().
				Foreground(charmtone.Ash).
				Background(charmtone.Charcoal),
			Code: lipgloss.NewStyle().
				Foreground(charmtone.Salt).
				Background(charmtone.Pepper),
		},
		InsertLine: LineStyle{
			LineNumber: lipgloss.NewStyle().
				Foreground(charmtone.Turtle).
				Background(lipgloss.Color("#293229")),
			Symbol: lipgloss.NewStyle().
				Foreground(charmtone.Turtle).
				Background(lipgloss.Color("#303a30")),
			Code: lipgloss.NewStyle().
				Foreground(charmtone.Salt).
				Background(lipgloss.Color("#303a30")),
		},
		DeleteLine: LineStyle{
			LineNumber: lipgloss.NewStyle().
				Foreground(charmtone.Cherry).
				Background(lipgloss.Color("#332929")),
			Symbol: lipgloss.NewStyle().
				Foreground(charmtone.Cherry).
				Background(lipgloss.Color("#3a3030")),
			Code: lipgloss.NewStyle().
				Foreground(charmtone.Salt).
				Background(lipgloss.Color("#3a3030")),
		},
	}
}
