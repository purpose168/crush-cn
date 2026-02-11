package dialog

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/purpose168/crush-cn/internal/ui/common"
)

// Dialog sizing constants.
const (
	// defaultDialogMaxWidth 是标准对话框的最大宽度。
	defaultDialogMaxWidth = 120
	// defaultDialogHeight 是标准对话框的默认高度。
	defaultDialogHeight = 30
	// titleContentHeight 是标题内容行的高度。
	titleContentHeight = 1
	// inputContentHeight 是输入内容行的高度。
	inputContentHeight = 1
)

// CloseKey 是关闭对话框的默认键绑定。
var CloseKey = key.NewBinding(
	key.WithKeys("esc", "alt+esc"),
	key.WithHelp("esc", "退出"),
)

// Action 表示在对话框中处理消息后执行的操作。
type Action any

// Dialog 是一个可以显示在 UI 顶部的组件。
type Dialog interface {
	// ID 返回对话框的唯一标识符。
	ID() string
	// HandleMsg 处理消息并返回操作。[Action] 可以是任何内容，
	// 调用者负责适当地处理它。
	HandleMsg(msg tea.Msg) Action
	// Draw 在提供的屏幕上绘制对话框，在指定区域内，
	// 并返回屏幕上所需的光标位置。
	Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor
}

// LoadingDialog 是一个可以显示加载状态的对话框。
type LoadingDialog interface {
	StartLoading() tea.Cmd
	StopLoading()
}

// Overlay 将多个对话框作为覆盖层进行管理。
type Overlay struct {
	dialogs []Dialog
}

// NewOverlay 创建一个新的 [Overlay] 实例。
func NewOverlay(dialogs ...Dialog) *Overlay {
	return &Overlay{
		dialogs: dialogs,
	}
}

// HasDialogs 检查是否有任何活动对话框。
func (d *Overlay) HasDialogs() bool {
	return len(d.dialogs) > 0
}

// ContainsDialog 检查是否存在具有指定 ID 的对话框。
func (d *Overlay) ContainsDialog(dialogID string) bool {
	for _, dialog := range d.dialogs {
		if dialog.ID() == dialogID {
			return true
		}
	}
	return false
}

// OpenDialog 向堆栈中打开一个新对话框。
func (d *Overlay) OpenDialog(dialog Dialog) {
	d.dialogs = append(d.dialogs, dialog)
}

// CloseDialog 从堆栈中关闭具有指定 ID 的对话框。
func (d *Overlay) CloseDialog(dialogID string) {
	for i, dialog := range d.dialogs {
		if dialog.ID() == dialogID {
			d.removeDialog(i)
			return
		}
	}
}

// CloseFrontDialog 关闭堆栈中的前面对话框。
func (d *Overlay) CloseFrontDialog() {
	if len(d.dialogs) == 0 {
		return
	}
	d.removeDialog(len(d.dialogs) - 1)
}

// Dialog 返回具有指定 ID 的对话框，如果未找到则返回 nil。
func (d *Overlay) Dialog(dialogID string) Dialog {
	for _, dialog := range d.dialogs {
		if dialog.ID() == dialogID {
			return dialog
		}
	}
	return nil
}

// DialogLast 返回前面对话框，如果没有对话框则返回 nil。
func (d *Overlay) DialogLast() Dialog {
	if len(d.dialogs) == 0 {
		return nil
	}
	return d.dialogs[len(d.dialogs)-1]
}

// BringToFront 将具有指定 ID 的对话框带到前面。
func (d *Overlay) BringToFront(dialogID string) {
	for i, dialog := range d.dialogs {
		if dialog.ID() == dialogID {
			// 将对话框移动到切片的末尾
			d.dialogs = append(d.dialogs[:i], d.dialogs[i+1:]...)
			d.dialogs = append(d.dialogs, dialog)
			return
		}
	}
}

// Update 处理对话框更新。
func (d *Overlay) Update(msg tea.Msg) tea.Msg {
	if len(d.dialogs) == 0 {
		return nil
	}

	idx := len(d.dialogs) - 1 // 活动对话框是最后一个
	dialog := d.dialogs[idx]
	if dialog == nil {
		return nil
	}

	return dialog.HandleMsg(msg)
}

// StartLoading 为前面对话框启动加载状态（如果它实现了 [LoadingDialog]）。
func (d *Overlay) StartLoading() tea.Cmd {
	dialog := d.DialogLast()
	if ld, ok := dialog.(LoadingDialog); ok {
		return ld.StartLoading()
	}
	return nil
}

// StopLoading 为前面对话框停止加载状态（如果它实现了 [LoadingDialog]）。
func (d *Overlay) StopLoading() {
	dialog := d.DialogLast()
	if ld, ok := dialog.(LoadingDialog); ok {
		ld.StopLoading()
	}
}

// DrawCenterCursor 在屏幕区域中绘制居中的给定字符串视图，
// 并相应地调整光标位置。
func DrawCenterCursor(scr uv.Screen, area uv.Rectangle, view string, cur *tea.Cursor) {
	width, height := lipgloss.Size(view)
	center := common.CenterRect(area, width, height)
	if cur != nil {
		cur.X += center.Min.X
		cur.Y += center.Min.Y
	}
	uv.NewStyledString(view).Draw(scr, center)
}

// DrawCenter 在屏幕区域中绘制居中的给定字符串视图。
func DrawCenter(scr uv.Screen, area uv.Rectangle, view string) {
	DrawCenterCursor(scr, area, view, nil)
}

// DrawOnboarding 在屏幕区域中绘制居中的给定字符串视图。
func DrawOnboarding(scr uv.Screen, area uv.Rectangle, view string) {
	DrawOnboardingCursor(scr, area, view, nil)
}

// DrawOnboardingCursor 在屏幕的底部左侧区域绘制给定字符串视图。
func DrawOnboardingCursor(scr uv.Screen, area uv.Rectangle, view string, cur *tea.Cursor) {
	width, height := lipgloss.Size(view)
	bottomLeft := common.BottomLeftRect(area, width, height)
	if cur != nil {
		cur.X += bottomLeft.Min.X
		cur.Y += bottomLeft.Min.Y
	}
	uv.NewStyledString(view).Draw(scr, bottomLeft)
}

// Draw 渲染覆盖层及其对话框。
func (d *Overlay) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	var cur *tea.Cursor
	for _, dialog := range d.dialogs {
		cur = dialog.Draw(scr, area)
	}
	return cur
}

// removeDialog 从堆栈中移除对话框。
func (d *Overlay) removeDialog(idx int) {
	if idx < 0 || idx >= len(d.dialogs) {
		return
	}
	d.dialogs = append(d.dialogs[:idx], d.dialogs[idx+1:]...)
}
