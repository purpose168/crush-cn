package common

import (
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	xstrings "github.com/charmbracelet/x/exp/strings"
)

// Capabilities 定义支持的不同终端能力。
type Capabilities struct {
	// Profile 是用于确定如何渲染颜色的终端颜色配置文件。
	Profile colorprofile.Profile
	// Columns 是终端中的字符列数。
	Columns int
	// Rows 是终端中的字符行数。
	Rows int
	// PixelX 是终端的宽度（像素）。
	PixelX int
	// PixelY 是终端的高度（像素）。
	PixelY int
	// KittyGraphics 指示终端是否支持 Kitty 图形协议。
	KittyGraphics bool
	// SixelGraphics 指示终端是否支持 Sixel 图形。
	SixelGraphics bool
	// Env 是终端环境变量。
	Env uv.Environ
	// TerminalVersion 是终端版本字符串。
	TerminalVersion string
	// ReportFocusEvents 指示终端是否支持焦点事件。
	ReportFocusEvents bool
}

// Update 根据给定消息更新能力。
func (c *Capabilities) Update(msg any) {
	switch m := msg.(type) {
	case tea.EnvMsg:
		c.Env = uv.Environ(m)
	case tea.ColorProfileMsg:
		c.Profile = m.Profile
	case tea.WindowSizeMsg:
		c.Columns = m.Width
		c.Rows = m.Height
	case uv.PixelSizeEvent:
		c.PixelX = m.Width
		c.PixelY = m.Height
	case uv.KittyGraphicsEvent:
		c.KittyGraphics = true
	case uv.PrimaryDeviceAttributesEvent:
		if slices.Contains(m, 4) {
			c.SixelGraphics = true
		}
	case tea.TerminalVersionMsg:
		c.TerminalVersion = m.Name
	case uv.ModeReportEvent:
		switch m.Mode {
		case ansi.ModeFocusEvent:
			c.ReportFocusEvents = modeSupported(m.Value)
		}
	}
}

// QueryCmd 返回一个 [tea.Cmd]，用于查询终端的不同能力。
func QueryCmd(env uv.Environ) tea.Cmd {
	var sb strings.Builder
	sb.WriteString(ansi.RequestPrimaryDeviceAttributes)
	sb.WriteString(ansi.QueryModifyOtherKeys)

	// 仅应发送到"智能"普通终端的查询。
	shouldQueryFor := shouldQueryCapabilities(env)
	if shouldQueryFor {
		sb.WriteString(ansi.RequestNameVersion)
		// sb.WriteString(ansi.RequestModeFocusEvent) // TODO: 当我们需要通知时重新启用。
		sb.WriteString(ansi.WindowOp(14)) // 窗口大小（像素）
		kittyReq := ansi.KittyGraphics([]byte("AAAA"), "i=31", "s=1", "v=1", "a=q", "t=d", "f=24")
		if _, isTmux := env.LookupEnv("TMUX"); isTmux {
			kittyReq = ansi.TmuxPassthrough(kittyReq)
		}
		sb.WriteString(kittyReq)
	}

	return tea.Raw(sb.String())
}

// SupportsTrueColor 如果终端支持真彩色则返回 true。
func (c Capabilities) SupportsTrueColor() bool {
	return c.Profile == colorprofile.TrueColor
}

// SupportsKittyGraphics 如果终端支持 Kitty 图形则返回 true。
func (c Capabilities) SupportsKittyGraphics() bool {
	return c.KittyGraphics
}

// SupportsSixelGraphics 如果终端支持 Sixel 图形则返回 true。
func (c Capabilities) SupportsSixelGraphics() bool {
	return c.SixelGraphics
}

// CellSize 返回单个终端单元格的像素大小。
func (c Capabilities) CellSize() (width, height int) {
	if c.Columns == 0 || c.Rows == 0 {
		return 0, 0
	}
	return c.PixelX / c.Columns, c.PixelY / c.Rows
}

func modeSupported(v ansi.ModeSetting) bool {
	return v.IsSet() || v.IsReset()
}

// kittyTerminals 定义支持查询能力的终端。
var kittyTerminals = []string{"alacritty", "ghostty", "kitty", "rio", "wezterm"}

func shouldQueryCapabilities(env uv.Environ) bool {
	const osVendorTypeApple = "Apple"
	termType := env.Getenv("TERM")
	termProg, okTermProg := env.LookupEnv("TERM_PROGRAM")
	_, okSSHTTY := env.LookupEnv("SSH_TTY")
	if okTermProg && strings.Contains(termProg, osVendorTypeApple) {
		return false
	}
	return (!okTermProg && !okSSHTTY) ||
		(!strings.Contains(termProg, osVendorTypeApple) && !okSSHTTY) ||
		// 支持 XTVERSION 的终端。
		xstrings.ContainsAnyOf(termType, kittyTerminals...)
}
