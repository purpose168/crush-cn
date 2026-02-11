package dialog

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/catwalk/pkg/catwalk"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/oauth"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/util"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/pkg/browser"
)

type OAuthProvider interface {
	name() string
	initiateAuth() tea.Msg
	startPolling(deviceCode string, expiresIn int) tea.Cmd
	stopPolling() tea.Msg
}

// OAuthState 表示设备流程的当前状态。
type OAuthState int

const (
	OAuthStateInitializing OAuthState = iota
	OAuthStateDisplay
	OAuthStateSuccess
	OAuthStateError
)

// OAuthID 是模型选择对话框的标识符。
const OAuthID = "oauth"

// OAuth 处理 OAuth 流程认证。
type OAuth struct {
	com          *common.Common
	isOnboarding bool

	provider      catwalk.Provider
	model         config.SelectedModel
	modelType     config.SelectedModelType
	oAuthProvider OAuthProvider

	State OAuthState

	spinner spinner.Model
	help    help.Model
	keyMap  struct {
		Copy   key.Binding
		Submit key.Binding
		Close  key.Binding
	}

	width           int
	deviceCode      string
	userCode        string
	verificationURL string
	expiresIn       int
	interval        int
	token           *oauth.Token
	cancelFunc      context.CancelFunc
}

var _ Dialog = (*OAuth)(nil)

// newOAuth 创建一个新的设备流程组件。
func newOAuth(
	com *common.Common,
	isOnboarding bool,
	provider catwalk.Provider,
	model config.SelectedModel,
	modelType config.SelectedModelType,
	oAuthProvider OAuthProvider,
) (*OAuth, tea.Cmd) {
	t := com.Styles

	m := OAuth{}
	m.com = com
	m.isOnboarding = isOnboarding
	m.provider = provider
	m.model = model
	m.modelType = modelType
	m.oAuthProvider = oAuthProvider
	m.width = 60
	m.State = OAuthStateInitializing

	m.spinner = spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(t.Base.Foreground(t.GreenLight)),
	)

	m.help = help.New()
	m.help.Styles = t.DialogHelpStyles()

	m.keyMap.Copy = key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "复制代码"),
	)
	m.keyMap.Submit = key.NewBinding(
		key.WithKeys("enter", "ctrl+y"),
		key.WithHelp("enter", "复制并打开"),
	)
	m.keyMap.Close = CloseKey

	return &m, tea.Batch(m.spinner.Tick, m.oAuthProvider.initiateAuth)
}

// ID 实现 Dialog 接口。
func (m *OAuth) ID() string {
	return OAuthID
}

// HandleMsg 处理消息和状态转换。
func (m *OAuth) HandleMsg(msg tea.Msg) Action {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		switch m.State {
		case OAuthStateInitializing, OAuthStateDisplay:
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			if cmd != nil {
				return ActionCmd{cmd}
			}
		}

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keyMap.Copy):
			cmd := m.copyCode()
			return ActionCmd{cmd}

		case key.Matches(msg, m.keyMap.Submit):
			switch m.State {
			case OAuthStateSuccess:
				return m.saveKeyAndContinue()

			default:
				cmd := m.copyCodeAndOpenURL()
				return ActionCmd{cmd}
			}

		case key.Matches(msg, m.keyMap.Close):
			switch m.State {
			case OAuthStateSuccess:
				return m.saveKeyAndContinue()

			default:
				return ActionClose{}
			}
		}

	case ActionInitiateOAuth:
		m.deviceCode = msg.DeviceCode
		m.userCode = msg.UserCode
		m.expiresIn = msg.ExpiresIn
		m.verificationURL = msg.VerificationURL
		m.interval = msg.Interval
		m.State = OAuthStateDisplay
		return ActionCmd{m.oAuthProvider.startPolling(msg.DeviceCode, msg.ExpiresIn)}

	case ActionCompleteOAuth:
		m.State = OAuthStateSuccess
		m.token = msg.Token
		return ActionCmd{m.oAuthProvider.stopPolling}

	case ActionOAuthErrored:
		m.State = OAuthStateError
		cmd := tea.Batch(m.oAuthProvider.stopPolling, util.ReportError(msg.Error))
		return ActionCmd{cmd}
	}
	return nil
}

// View 渲染设备流程对话框。
func (m *OAuth) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	var (
		t           = m.com.Styles
		dialogStyle = t.Dialog.View.Width(m.width)
	)
	if m.isOnboarding {
		view := m.dialogContent()
		DrawOnboarding(scr, area, view)
	} else {
		view := dialogStyle.Render(m.dialogContent())
		DrawCenter(scr, area, view)
	}
	return nil
}

func (m *OAuth) dialogContent() string {
	var (
		t         = m.com.Styles
		helpStyle = t.Dialog.HelpView
	)

	switch m.State {
	case OAuthStateInitializing:
		return m.innerDialogContent()

	default:
		elements := []string{
			m.headerContent(),
			m.innerDialogContent(),
			helpStyle.Render(m.help.View(m)),
		}
		return strings.Join(elements, "\n")
	}
}

func (m *OAuth) headerContent() string {
	var (
		t            = m.com.Styles
		titleStyle   = t.Dialog.Title
		textStyle    = t.Dialog.PrimaryText
		dialogStyle  = t.Dialog.View.Width(m.width)
		headerOffset = titleStyle.GetHorizontalFrameSize() + dialogStyle.GetHorizontalFrameSize()
		dialogTitle  = fmt.Sprintf("使用 %s 进行身份验证", m.oAuthProvider.name())
	)
	if m.isOnboarding {
		return textStyle.Render(dialogTitle)
	}
	return common.DialogTitle(t, titleStyle.Render(dialogTitle), m.width-headerOffset, t.Primary, t.Secondary)
}

func (m *OAuth) innerDialogContent() string {
	var (
		t            = m.com.Styles
		whiteStyle   = lipgloss.NewStyle().Foreground(t.White)
		primaryStyle = lipgloss.NewStyle().Foreground(t.Primary)
		greenStyle   = lipgloss.NewStyle().Foreground(t.GreenLight)
		linkStyle    = lipgloss.NewStyle().Foreground(t.GreenDark).Underline(true)
		errorStyle   = lipgloss.NewStyle().Foreground(t.Error)
		mutedStyle   = lipgloss.NewStyle().Foreground(t.FgMuted)
	)

	switch m.State {
	case OAuthStateInitializing:
		return lipgloss.NewStyle().
			Margin(1, 1).
			Width(m.width - 2).
			Align(lipgloss.Center).
			Render(
				greenStyle.Render(m.spinner.View()) +
					mutedStyle.Render("正在初始化..."),
			)

	case OAuthStateDisplay:
		instructions := lipgloss.NewStyle().
			Margin(0, 1).
			Width(m.width - 2).
			Render(
				whiteStyle.Render("按 ") +
					primaryStyle.Render("enter") +
					whiteStyle.Render(" 复制下面的代码并打开浏览器。"),
			)

		codeBox := lipgloss.NewStyle().
			Width(m.width-2).
			Height(7).
			Align(lipgloss.Center, lipgloss.Center).
			Background(t.BgBaseLighter).
			Margin(0, 1).
			Render(
				lipgloss.NewStyle().
					Bold(true).
					Foreground(t.White).
					Render(m.userCode),
			)

		link := linkStyle.Hyperlink(m.verificationURL, "id=oauth-verify").Render(m.verificationURL)
		url := mutedStyle.
			Margin(0, 1).
			Width(m.width - 2).
			Render("浏览器没有打开？请访问\n" + link)

		waiting := lipgloss.NewStyle().
			Margin(0, 1).
			Width(m.width - 2).
			Render(
				greenStyle.Render(m.spinner.View()) + mutedStyle.Render("正在验证..."),
			)

		return lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			instructions,
			"",
			codeBox,
			"",
			url,
			"",
			waiting,
			"",
		)

	case OAuthStateSuccess:
		return greenStyle.
			Margin(1).
			Width(m.width - 2).
			Render("身份验证成功！")

	case OAuthStateError:
		return lipgloss.NewStyle().
			Margin(1).
			Width(m.width - 2).
			Render(errorStyle.Render("身份验证失败。"))

	default:
		return ""
	}
}

// FullHelp 返回完整的帮助视图。
func (m *OAuth) FullHelp() [][]key.Binding {
	return [][]key.Binding{m.ShortHelp()}
}

// ShortHelp 返回简短的帮助视图。
func (m *OAuth) ShortHelp() []key.Binding {
	switch m.State {
	case OAuthStateError:
		return []key.Binding{m.keyMap.Close}

	case OAuthStateSuccess:
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("finish", "ctrl+y", "esc"),
				key.WithHelp("enter", "完成"),
			),
		}

	default:
		return []key.Binding{
			m.keyMap.Copy,
			m.keyMap.Submit,
			m.keyMap.Close,
		}
	}
}

func (d *OAuth) copyCode() tea.Cmd {
	if d.State != OAuthStateDisplay {
		return nil
	}
	return tea.Sequence(
		tea.SetClipboard(d.userCode),
		util.ReportInfo("代码已复制到剪贴板"),
	)
}

func (d *OAuth) copyCodeAndOpenURL() tea.Cmd {
	if d.State != OAuthStateDisplay {
		return nil
	}
	return tea.Sequence(
		tea.SetClipboard(d.userCode),
		func() tea.Msg {
			if err := browser.OpenURL(d.verificationURL); err != nil {
				return ActionOAuthErrored{fmt.Errorf("无法打开浏览器: %w", err)}
			}
			return nil
		},
		util.ReportInfo("代码已复制且 URL 已打开"),
	)
}

func (m *OAuth) saveKeyAndContinue() Action {
	cfg := m.com.Config()

	err := cfg.SetProviderAPIKey(string(m.provider.ID), m.token)
	if err != nil {
		return ActionCmd{util.ReportError(fmt.Errorf("无法保存 API 密钥: %w", err))}
	}

	return ActionSelectModel{
		Provider:  m.provider,
		Model:     m.model,
		ModelType: m.modelType,
	}
}
