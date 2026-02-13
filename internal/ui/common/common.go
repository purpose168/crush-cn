package common

import (
	"fmt"
	"image"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/purpose168/crush-cn/internal/app"
	"github.com/purpose168/crush-cn/internal/config"
	"github.com/purpose168/crush-cn/internal/ui/styles"
	"github.com/purpose168/crush-cn/internal/ui/util"
)

// MaxAttachmentSize 定义文件附件的最大允许大小（5 MB）。
const MaxAttachmentSize = int64(5 * 1024 * 1024)

// AllowedImageTypes 定义允许的图像文件类型。
var AllowedImageTypes = []string{".jpg", ".jpeg", ".png"}

// Common 定义通用 UI 选项和配置。
type Common struct {
	App    *app.App
	Styles *styles.Styles
}

// Config 返回与此 [Common] 实例关联的配置。
func (c *Common) Config() *config.Config {
	return c.App.Config()
}

// DefaultCommon 返回默认的通用 UI 配置。
func DefaultCommon(app *app.App) *Common {
	s := styles.DefaultStyles()
	return &Common{
		App:    app,
		Styles: &s,
	}
}

// CenterRect 返回一个在给定区域内居中的新 [Rectangle]，具有指定的宽度和高度。
func CenterRect(area uv.Rectangle, width, height int) uv.Rectangle {
	centerX := area.Min.X + area.Dx()/2
	centerY := area.Min.Y + area.Dy()/2
	minX := centerX - width/2
	minY := centerY - height/2
	maxX := minX + width
	maxY := minY + height
	return image.Rect(minX, minY, maxX, maxY)
}

// BottomLeftRect 返回一个在给定区域内左下角定位的新 [Rectangle]，具有指定的宽度和高度。
func BottomLeftRect(area uv.Rectangle, width, height int) uv.Rectangle {
	minX := area.Min.X
	maxX := minX + width
	maxY := area.Max.Y
	minY := maxY - height
	return image.Rect(minX, minY, maxX, maxY)
}

// IsFileTooBig 检查给定路径的文件是否超过指定的大小限制。
func IsFileTooBig(filePath string, sizeLimit int64) (bool, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false, fmt.Errorf("获取文件信息失败: %w", err)
	}

	if fileInfo.Size() > sizeLimit {
		return true, nil
	}

	return false, nil
}

// CopyToClipboard 使用 OSC 52（终端转义序列）和本机剪贴板将给定文本复制到剪贴板，以实现最大兼容性。
// 返回一个向用户报告成功的命令，并附带指定的消息。
func CopyToClipboard(text, successMessage string) tea.Cmd {
	return CopyToClipboardWithCallback(text, successMessage, nil)
}

// CopyToClipboardWithCallback 将文本复制到剪贴板，并在显示成功消息之前执行回调。
// 这在需要执行额外操作（如清除 UI 状态）时非常有用。
func CopyToClipboardWithCallback(text, successMessage string, callback tea.Cmd) tea.Cmd {
	return tea.Sequence(
		tea.SetClipboard(text),
		func() tea.Msg {
			_ = clipboard.WriteAll(text)
			return nil
		},
		callback,
		util.ReportInfo(successMessage),
	)
}
