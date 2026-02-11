package dialog

import (
	"fmt"
	"image"
	_ "image/jpeg" // register JPEG format
	_ "image/png"  // register PNG format
	"os"
	"strings"
	"sync"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/purpose168/crush-cn/internal/home"
	"github.com/purpose168/crush-cn/internal/ui/common"
	fimage "github.com/purpose168/crush-cn/internal/ui/image"
)

// FilePickerID 是 FilePicker 对话框的标识符。
const FilePickerID = "filepicker"

// FilePicker 是一个允许用户选择文件或目录的对话框。
type FilePicker struct {
	com *common.Common

	imgEnc                      fimage.Encoding
	imgPrevWidth, imgPrevHeight int
	cellSizeW, cellSizeH        int

	fp              filepicker.Model
	help            help.Model
	previewingImage bool // 指示是否正在预览图像
	isTmux          bool

	km struct {
		Select,
		Down,
		Up,
		Forward,
		Backward,
		Navigate,
		Close key.Binding
	}
}

// CellSize 返回用于图像渲染的单元格大小。
func (f *FilePicker) CellSize() fimage.CellSize {
	return fimage.CellSize{
		Width:  f.cellSizeW,
		Height: f.cellSizeH,
	}
}

var _ Dialog = (*FilePicker)(nil)

// NewFilePicker 创建一个新的 [FilePicker] 对话框。
func NewFilePicker(com *common.Common) (*FilePicker, tea.Cmd) {
	f := new(FilePicker)
	f.com = com

	help := help.New()
	help.Styles = com.Styles.DialogHelpStyles()

	f.help = help

	f.km.Select = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "接受"),
	)
	f.km.Down = key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("down/j", "向下移动"),
	)
	f.km.Up = key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("up/k", "向上移动"),
	)
	f.km.Forward = key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("right/l", "向前移动"),
	)
	f.km.Backward = key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("left/h", "向后移动"),
	)
	f.km.Navigate = key.NewBinding(
		key.WithKeys("right", "l", "left", "h", "up", "k", "down", "j"),
		key.WithHelp("↑↓←→", "导航"),
	)
	f.km.Close = key.NewBinding(
		key.WithKeys("esc", "alt+esc"),
		key.WithHelp("esc", "关闭/退出"),
	)

	fp := filepicker.New()
	fp.AllowedTypes = common.AllowedImageTypes
	fp.ShowPermissions = false
	fp.ShowSize = false
	fp.AutoHeight = false
	fp.Styles = com.Styles.FilePicker
	fp.Cursor = ""
	fp.CurrentDirectory = f.WorkingDir()

	f.fp = fp

	return f, f.fp.Init()
}

// SetImageCapabilities 设置 [FilePicker] 的图像功能。
func (f *FilePicker) SetImageCapabilities(caps *common.Capabilities) {
	if caps != nil {
		if caps.SupportsKittyGraphics() {
			f.imgEnc = fimage.EncodingKitty
		}
		f.cellSizeW, f.cellSizeH = caps.CellSize()
		_, f.isTmux = caps.Env.LookupEnv("TMUX")
	}
}

// WorkingDir 返回 [FilePicker] 的当前工作目录。
func (f *FilePicker) WorkingDir() string {
	wd := f.com.Config().WorkingDir()
	if len(wd) > 0 {
		return wd
	}

	cwd, err := os.Getwd()
	if err != nil {
		return home.Dir()
	}

	return cwd
}

// ShortHelp 返回 [FilePicker] 对话框的简短帮助键绑定。
func (f *FilePicker) ShortHelp() []key.Binding {
	return []key.Binding{
		f.km.Navigate,
		f.km.Select,
		f.km.Close,
	}
}

// FullHelp 返回 [FilePicker] 对话框的完整帮助键绑定。
func (f *FilePicker) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			f.km.Select,
			f.km.Down,
			f.km.Up,
			f.km.Forward,
		},
		{
			f.km.Backward,
			f.km.Close,
		},
	}
}

// ID 返回 [FilePicker] 对话框的标识符。
func (f *FilePicker) ID() string {
	return FilePickerID
}

// HandleMsg 根据给定消息更新 [FilePicker] 对话框。
func (f *FilePicker) HandleMsg(msg tea.Msg) Action {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, f.km.Close):
			return ActionClose{}
		}
	}

	var cmd tea.Cmd
	f.fp, cmd = f.fp.Update(msg)
	if selFile := f.fp.HighlightedPath(); selFile != "" {
		var allowed bool
		for _, allowedExt := range f.fp.AllowedTypes {
			if strings.HasSuffix(strings.ToLower(selFile), allowedExt) {
				allowed = true
				break
			}
		}

		f.previewingImage = allowed
		if allowed && !fimage.HasTransmitted(selFile, f.imgPrevWidth, f.imgPrevHeight) {
			f.previewingImage = false
			img, err := loadImage(selFile)
			if err == nil {
				cmds = append(cmds, tea.Sequence(
					f.imgEnc.Transmit(selFile, img, f.CellSize(), f.imgPrevWidth, f.imgPrevHeight, f.isTmux),
					func() tea.Msg {
						f.previewingImage = true
						return nil
					},
				))
			}
		}
	}
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	if didSelect, path := f.fp.DidSelectFile(msg); didSelect {
		return ActionFilePickerSelected{Path: path}
	}

	return ActionCmd{tea.Batch(cmds...)}
}

const (
	filePickerMinWidth  = 70
	filePickerMinHeight = 10
)

// Draw 将 [FilePicker] 对话框渲染为字符串。
func (f *FilePicker) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	width := max(0, min(filePickerMinWidth, area.Dx()))
	height := max(0, min(10, area.Dy()))
	innerWidth := width - f.com.Styles.Dialog.View.GetHorizontalFrameSize()
	imgPrevHeight := filePickerMinHeight*2 - f.com.Styles.Dialog.ImagePreview.GetVerticalFrameSize()
	imgPrevWidth := innerWidth - f.com.Styles.Dialog.ImagePreview.GetHorizontalFrameSize()
	f.imgPrevWidth = imgPrevWidth
	f.imgPrevHeight = imgPrevHeight
	f.fp.SetHeight(height)

	styles := f.com.Styles.FilePicker
	styles.File = styles.File.Width(innerWidth)
	styles.Directory = styles.Directory.Width(innerWidth)
	styles.Selected = styles.Selected.PaddingLeft(1).Width(innerWidth)
	styles.DisabledSelected = styles.DisabledSelected.PaddingLeft(1).Width(innerWidth)
	f.fp.Styles = styles

	t := f.com.Styles
	rc := NewRenderContext(t, width)
	rc.Gap = 1
	rc.Title = "添加图像"
	rc.Help = f.help.View(f)

	imgPreview := t.Dialog.ImagePreview.Align(lipgloss.Center).Width(innerWidth).Render(f.imagePreview(imgPrevWidth, imgPrevHeight))
	rc.AddPart(imgPreview)

	files := strings.TrimSpace(f.fp.View())
	rc.AddPart(files)

	view := rc.Render()

	DrawCenter(scr, area, view)
	return nil
}

var (
	imagePreviewCache = map[string]string{}
	imagePreviewMutex sync.RWMutex
)

// imagePreview 返回 [FilePicker] 对话框的图像预览部分。
func (f *FilePicker) imagePreview(imgPrevWidth, imgPrevHeight int) string {
	if !f.previewingImage {
		key := fmt.Sprintf("%dx%d", imgPrevWidth, imgPrevHeight)
		imagePreviewMutex.RLock()
		cached, ok := imagePreviewCache[key]
		imagePreviewMutex.RUnlock()
		if ok {
			return cached
		}

		var sb strings.Builder
		for y := range imgPrevHeight {
			for range imgPrevWidth {
				sb.WriteRune('█')
			}
			if y < imgPrevHeight-1 {
				sb.WriteRune('\n')
			}
		}

		imagePreviewMutex.Lock()
		imagePreviewCache[key] = sb.String()
		imagePreviewMutex.Unlock()

		return sb.String()
	}

	if id := f.fp.HighlightedPath(); id != "" {
		r := f.imgEnc.Render(id, imgPrevWidth, imgPrevHeight)
		return r
	}

	return ""
}

func loadImage(path string) (img image.Image, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err = image.Decode(file)
	if err != nil {
		return nil, err
	}

	return img, nil
}
