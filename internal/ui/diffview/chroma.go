package diffview

import (
	"fmt"
	"image/color"
	"io"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/charmbracelet/crush/internal/ansiext"
)

var _ chroma.Formatter = chromaFormatter{}

// chromaFormatter 是一个用于Chroma的自定义格式化器，它使用Lip Gloss进行
// 前景样式设置，同时保持强制背景色。
type chromaFormatter struct {
	bgColor color.Color
}

// Format 实现chroma.Formatter接口。
func (c chromaFormatter) Format(w io.Writer, style *chroma.Style, it chroma.Iterator) error {
	for token := it(); token != chroma.EOF; token = it() {
		value := strings.TrimRight(token.Value, "\n")
		value = ansiext.Escape(value)

		entry := style.Get(token.Type)
		if entry.IsZero() {
			if _, err := fmt.Fprint(w, value); err != nil {
				return err
			}
			continue
		}

		s := lipgloss.NewStyle().
			Background(c.bgColor)

		if entry.Bold == chroma.Yes {
			s = s.Bold(true)
		}
		if entry.Underline == chroma.Yes {
			s = s.Underline(true)
		}
		if entry.Italic == chroma.Yes {
			s = s.Italic(true)
		}
		if entry.Colour.IsSet() {
			s = s.Foreground(lipgloss.Color(entry.Colour.String()))
		}

		if _, err := fmt.Fprint(w, s.Render(value)); err != nil {
			return err
		}
	}
	return nil
}
