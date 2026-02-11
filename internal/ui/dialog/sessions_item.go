package dialog

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/crush/internal/session"
	"github.com/charmbracelet/crush/internal/ui/list"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/charmbracelet/x/ansi"
	"github.com/dustin/go-humanize"
	"github.com/rivo/uniseg"
	"github.com/sahilm/fuzzy"
)

// ListItem 表示对话框列表中可选择和可搜索的项目。
type ListItem interface {
	list.FilterableItem
	list.Focusable
	list.MatchSettable

	// ID 返回项目的唯一标识符。
	ID() string
}

// SessionItem 包装一个[session.Session]以实现[ListItem]接口。
type SessionItem struct {
	session.Session
	t                *styles.Styles
	sessionsMode     sessionsMode
	m                fuzzy.Match
	cache            map[int]string
	updateTitleInput textinput.Model
	focused          bool
}

var _ ListItem = &SessionItem{}

// Filter 返回会话的可过滤值。
func (s *SessionItem) Filter() string {
	return s.Title
}

// ID 返回会话的唯一标识符。
func (s *SessionItem) ID() string {
	return s.Session.ID
}

// SetMatch 设置会话项目的模糊匹配。
func (s *SessionItem) SetMatch(m fuzzy.Match) {
	s.cache = nil
	s.m = m
}

// InputValue 返回更新的标题值
func (s *SessionItem) InputValue() string {
	return s.updateTitleInput.Value()
}

// HandleInput 将输入消息转发到更新标题输入
func (s *SessionItem) HandleInput(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	s.updateTitleInput, cmd = s.updateTitleInput.Update(msg)
	return cmd
}

// Cursor 返回更新标题输入的光标
func (s *SessionItem) Cursor() *tea.Cursor {
	return s.updateTitleInput.Cursor()
}

// Render 返回会话项目的字符串表示。
func (s *SessionItem) Render(width int) string {
	info := humanize.Time(time.Unix(s.UpdatedAt, 0))
	styles := ListItemStyles{
		ItemBlurred:     s.t.Dialog.NormalItem,
		ItemFocused:     s.t.Dialog.SelectedItem,
		InfoTextBlurred: s.t.Subtle,
		InfoTextFocused: s.t.Base,
	}

	switch s.sessionsMode {
	case sessionsModeDeleting:
		styles.ItemBlurred = s.t.Dialog.Sessions.DeletingItemBlurred
		styles.ItemFocused = s.t.Dialog.Sessions.DeletingItemFocused
	case sessionsModeUpdating:
		styles.ItemBlurred = s.t.Dialog.Sessions.RenamingItemBlurred
		styles.ItemFocused = s.t.Dialog.Sessions.RenamingingItemFocused
		if s.focused {
			inputWidth := width - styles.InfoTextFocused.GetHorizontalFrameSize()
			s.updateTitleInput.SetWidth(inputWidth)
			s.updateTitleInput.Placeholder = ansi.Truncate(s.Title, width, "…")
			return styles.ItemFocused.Render(s.updateTitleInput.View())
		}
	}

	return renderItem(styles, s.Title, info, s.focused, width, s.cache, &s.m)
}

type ListItemStyles struct {
	ItemBlurred     lipgloss.Style
	ItemFocused     lipgloss.Style
	InfoTextBlurred lipgloss.Style
	InfoTextFocused lipgloss.Style
}

func renderItem(t ListItemStyles, title string, info string, focused bool, width int, cache map[int]string, m *fuzzy.Match) string {
	if cache == nil {
		cache = make(map[int]string)
	}

	cached, ok := cache[width]
	if ok {
		return cached
	}

	style := t.ItemBlurred
	if focused {
		style = t.ItemFocused
	}

	var infoText string
	var infoWidth int
	lineWidth := width
	if len(info) > 0 {
		infoText = fmt.Sprintf(" %s ", info)
		if focused {
			infoText = t.InfoTextFocused.Render(infoText)
		} else {
			infoText = t.InfoTextBlurred.Render(infoText)
		}

		infoWidth = lipgloss.Width(infoText)
	}

	title = ansi.Truncate(title, max(0, lineWidth-infoWidth), "")
	titleWidth := lipgloss.Width(title)
	gap := strings.Repeat(" ", max(0, lineWidth-titleWidth-infoWidth))
	content := title
	if m != nil && len(m.MatchedIndexes) > 0 {
		var lastPos int
		parts := make([]string, 0)
		ranges := matchedRanges(m.MatchedIndexes)
		for _, rng := range ranges {
			start, stop := bytePosToVisibleCharPos(title, rng)
			if start > lastPos {
				parts = append(parts, ansi.Cut(title, lastPos, start))
			}
			// 注意：我们在这里使用[ansi.Style]而不是[lipgloss.Style]
			// 因为我们可以通过[ansi.AttrUnderline]和[ansi.AttrNoUnderline]
			// 更精确地控制下划线的开始和停止
			// 这些只影响下划线属性而不会干扰其他样式
			parts = append(parts,
				ansi.NewStyle().Underline(true).String(),
				ansi.Cut(title, start, stop+1),
				ansi.NewStyle().Underline(false).String(),
			)
			lastPos = stop + 1
		}
		if lastPos < ansi.StringWidth(title) {
			parts = append(parts, ansi.Cut(title, lastPos, ansi.StringWidth(title)))
		}

		content = strings.Join(parts, "")
	}

	content = style.Render(content + gap + infoText)
	cache[width] = content
	return content
}

// SetFocused 设置会话项目的焦点状态。
func (s *SessionItem) SetFocused(focused bool) {
	if s.focused != focused {
		s.cache = nil
	}
	s.focused = focused
}

// sessionItems 接受一个[session.Session]切片并将它们转换为
// [ListItem]切片。
func sessionItems(t *styles.Styles, mode sessionsMode, sessions ...session.Session) []list.FilterableItem {
	items := make([]list.FilterableItem, len(sessions))
	for i, s := range sessions {
		item := &SessionItem{Session: s, t: t, sessionsMode: mode}
		if mode == sessionsModeUpdating {
			item.updateTitleInput = textinput.New()
			item.updateTitleInput.SetVirtualCursor(false)
			item.updateTitleInput.Prompt = ""
			inputStyle := t.TextInput
			inputStyle.Focused.Placeholder = t.Dialog.Sessions.RenamingPlaceholder
			item.updateTitleInput.SetStyles(inputStyle)
			item.updateTitleInput.Focus()
		}
		items[i] = item
	}
	return items
}

func matchedRanges(in []int) [][2]int {
	if len(in) == 0 {
		return [][2]int{}
	}
	current := [2]int{in[0], in[0]}
	if len(in) == 1 {
		return [][2]int{current}
	}
	var out [][2]int
	for i := 1; i < len(in); i++ {
		if in[i] == current[1]+1 {
			current[1] = in[i]
		} else {
			out = append(out, current)
			current = [2]int{in[i], in[i]}
		}
	}
	out = append(out, current)
	return out
}

func bytePosToVisibleCharPos(str string, rng [2]int) (int, int) {
	bytePos, byteStart, byteStop := 0, rng[0], rng[1]
	pos, start, stop := 0, 0, 0
	gr := uniseg.NewGraphemes(str)
	for byteStart > bytePos {
		if !gr.Next() {
			break
		}
		bytePos += len(gr.Str())
		pos += max(1, gr.Width())
	}
	start = pos
	for byteStop > bytePos {
		if !gr.Next() {
			break
		}
		bytePos += len(gr.Str())
		pos += max(1, gr.Width())
	}
	stop = pos
	return start, stop
}
