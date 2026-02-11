package dialog

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"charm.land/catwalk/pkg/catwalk"
	"github.com/charmbracelet/crush/internal/commands"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/message"
	"github.com/charmbracelet/crush/internal/oauth"
	"github.com/charmbracelet/crush/internal/permission"
	"github.com/charmbracelet/crush/internal/session"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/util"
)

// ActionClose 是一个关闭当前对话框的消息。
type ActionClose struct{}

// ActionQuit 是一个退出应用程序的消息。
type ActionQuit = tea.QuitMsg

// ActionOpenDialog 是一个打开对话框的消息。
type ActionOpenDialog struct {
	DialogID string
}

// ActionSelectSession 是一个表示已选择会话的消息。
type ActionSelectSession struct {
	Session session.Session
}

// ActionSelectModel 是一个表示已选择模型的消息。
type ActionSelectModel struct {
	Provider       catwalk.Provider
	Model          config.SelectedModel
	ModelType      config.SelectedModelType
	ReAuthenticate bool
}

// 命令的消息
type (
	ActionNewSession        struct{}
	ActionToggleHelp        struct{}
	ActionToggleCompactMode struct{}
	ActionToggleThinking    struct{}
	ActionExternalEditor    struct{}
	ActionToggleYoloMode    struct{}
	// ActionInitializeProject 是一个初始化项目的消息。
	ActionInitializeProject struct{}
	ActionSummarize         struct {
		SessionID string
	}
	// ActionSelectReasoningEffort 是一个表示已选择推理强度的消息。
	ActionSelectReasoningEffort struct {
		Effort string
	}
	ActionPermissionResponse struct {
		Permission permission.PermissionRequest
		Action     PermissionAction
	}
	// ActionRunCustomCommand 是一个运行自定义命令的消息。
	ActionRunCustomCommand struct {
		Content   string
		Arguments []commands.Argument
		Args      map[string]string // 实际参数值
	}
	// ActionRunMCPPrompt 是一个运行自定义命令的消息。
	ActionRunMCPPrompt struct {
		Title       string
		Description string
		PromptID    string
		ClientID    string
		Arguments   []commands.Argument
		Args        map[string]string // 实际参数值
	}
)

// API 密钥输入对话框的消息。
type (
	ActionChangeAPIKeyState struct {
		State APIKeyInputState
	}
)

// OAuth2 设备流程对话框的消息。
type (
	// ActionInitiateOAuth 在设备认证成功启动时发送。
	ActionInitiateOAuth struct {
		DeviceCode      string
		UserCode        string
		ExpiresIn       int
		VerificationURL string
		Interval        int
	}

	// ActionCompleteOAuth 在设备流程成功完成时发送。
	ActionCompleteOAuth struct {
		Token *oauth.Token
	}

	// ActionOAuthErrored 在设备流程遇到错误时发送。
	ActionOAuthErrored struct {
		Error error
	}
)

// ActionCmd 表示一个携带 [tea.Cmd] 的操作，该命令将被传递到 Bubble Tea 程序循环。
type ActionCmd struct {
	Cmd tea.Cmd
}

// ActionFilePickerSelected 是一个表示在文件选择器对话框中已选择文件的消息。
type ActionFilePickerSelected struct {
	Path string
}

// Cmd 返回一个命令，该命令读取指定路径的文件，并向程序发送 [message.Attachment] 消息。
func (a ActionFilePickerSelected) Cmd() tea.Cmd {
	path := a.Path
	if path == "" {
		return nil
	}
	return func() tea.Msg {
		isFileLarge, err := common.IsFileTooBig(path, common.MaxAttachmentSize)
		if err != nil {
			return util.InfoMsg{
				Type: util.InfoTypeError,
				Msg:  fmt.Sprintf("无法读取图像：%v", err),
			}
		}
		if isFileLarge {
			return util.InfoMsg{
				Type: util.InfoTypeError,
				Msg:  "文件过大，最大 5MB",
			}
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return util.InfoMsg{
				Type: util.InfoTypeError,
				Msg:  fmt.Sprintf("无法读取图像：%v", err),
			}
		}

		mimeBufferSize := min(512, len(content))
		mimeType := http.DetectContentType(content[:mimeBufferSize])
		fileName := filepath.Base(path)

		return message.Attachment{
			FilePath: path,
			FileName: fileName,
			MimeType: mimeType,
			Content:  content,
		}
	}
}
