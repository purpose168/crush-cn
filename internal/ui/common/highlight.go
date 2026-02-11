package common

import (
	"bytes"
	"image/color"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	chromastyles "github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/crush/internal/ui/styles"
)

// SyntaxHighlight 根据文件名和背景色对给定的源代码应用语法高亮。
// 它返回高亮代码作为字符串。
func SyntaxHighlight(st *styles.Styles, source, fileName string, bg color.Color) (string, error) {
	// 确定要使用的语言词法分析器
	l := lexers.Match(fileName)
	if l == nil {
		l = lexers.Analyse(source)
	}
	if l == nil {
		l = lexers.Fallback
	}
	l = chroma.Coalesce(l)

	// 获取格式化器
	f := formatters.Get("terminal16m")
	if f == nil {
		f = formatters.Fallback
	}

	style := chroma.MustNewStyle("crush", st.ChromaTheme())

	// 修改样式以使用提供的背景
	s, err := style.Builder().Transform(
		func(t chroma.StyleEntry) chroma.StyleEntry {
			r, g, b, _ := bg.RGBA()
			t.Background = chroma.NewColour(uint8(r>>8), uint8(g>>8), uint8(b>>8))
			return t
		},
	).Build()
	if err != nil {
		s = chromastyles.Fallback
	}

	// 标记化和格式化
	it, err := l.Tokenise(nil, source)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = f.Format(&buf, s, it)
	return buf.String(), err
}
