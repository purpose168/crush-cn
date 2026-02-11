package model

import (
	"time"

	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/util"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// DefaultStatusTTL 是状态消息的默认生存时间。
const DefaultStatusTTL = 5 * time.Second

// Status 是状态栏和帮助模型。
type Status struct {
	com      *common.Common
	hideHelp bool
	help     help.Model
	helpKm   help.KeyMap
	msg      util.InfoMsg
}

// NewStatus 创建一个新的状态栏和帮助模型。
func NewStatus(com *common.Common, km help.KeyMap) *Status {
	s := new(Status)
	s.com = com
	s.help = help.New()
	s.help.Styles = com.Styles.Help
	s.helpKm = km
	return s
}

// SetInfoMsg 设置状态信息消息。
func (s *Status) SetInfoMsg(msg util.InfoMsg) {
	s.msg = msg
}

// ClearInfoMsg 清除状态信息消息。
func (s *Status) ClearInfoMsg() {
	s.msg = util.InfoMsg{}
}

// SetWidth 设置状态栏和帮助视图的宽度。
func (s *Status) SetWidth(width int) {
	s.help.SetWidth(width)
}

// ShowingAll 返回是否显示完整的帮助视图。
func (s *Status) ShowingAll() bool {
	return s.help.ShowAll
}

// ToggleHelp 切换完整的帮助视图。
func (s *Status) ToggleHelp() {
	s.help.ShowAll = !s.help.ShowAll
}

// SetHideHelp 设置应用程序是否处于引导流程中。
func (s *Status) SetHideHelp(hideHelp bool) {
	s.hideHelp = hideHelp
}

// Draw 将状态栏绘制到屏幕上。
func (s *Status) Draw(scr uv.Screen, area uv.Rectangle) {
	if !s.hideHelp {
		helpView := s.com.Styles.Status.Help.Render(s.help.View(s.helpKm))
		uv.NewStyledString(helpView).Draw(scr, area)
	}

	// 渲染通知
	if s.msg.IsEmpty() {
		return
	}

	var indStyle lipgloss.Style
	var msgStyle lipgloss.Style
	switch s.msg.Type {
	case util.InfoTypeError:
		indStyle = s.com.Styles.Status.ErrorIndicator
		msgStyle = s.com.Styles.Status.ErrorMessage
	case util.InfoTypeWarn:
		indStyle = s.com.Styles.Status.WarnIndicator
		msgStyle = s.com.Styles.Status.WarnMessage
	case util.InfoTypeUpdate:
		indStyle = s.com.Styles.Status.UpdateIndicator
		msgStyle = s.com.Styles.Status.UpdateMessage
	case util.InfoTypeInfo:
		indStyle = s.com.Styles.Status.InfoIndicator
		msgStyle = s.com.Styles.Status.InfoMessage
	case util.InfoTypeSuccess:
		indStyle = s.com.Styles.Status.SuccessIndicator
		msgStyle = s.com.Styles.Status.SuccessMessage
	}

	ind := indStyle.String()
	messageWidth := area.Dx() - lipgloss.Width(ind)
	msg := ansi.Truncate(s.msg.Msg, messageWidth, "…")
	info := msgStyle.Width(messageWidth).Render(msg)

	// 在帮助视图上绘制信息消息
	uv.NewStyledString(ind+info).Draw(scr, area)
}

// clearInfoMsgCmd 返回一个命令，在给定的TTL之后清除信息消息。
func clearInfoMsgCmd(ttl time.Duration) tea.Cmd {
	return tea.Tick(ttl, func(time.Time) tea.Msg {
		return util.ClearStatusMsg{}
	})
}
