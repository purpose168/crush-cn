// Package logo 以风格化的方式渲染Crush标志。
package logo

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/MakeNowJust/heredoc"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/slice"
)

// letterform 表示字母形式。可以通过布尔参数将其水平拉伸
// 指定的量。
type letterform func(bool) string

const diag = `╱`

// Opts 是渲染Crush标题艺术字的选项。
type Opts struct {
	FieldColor   color.Color // 对角线
	TitleColorA  color.Color // 左侧渐变坡度点
	TitleColorB  color.Color // 右侧渐变坡度点
	CharmColor   color.Color // Charm™文本颜色
	VersionColor color.Color // 版本文本颜色
	Width        int         // 渲染的logo宽度，用于截断
}

// Render 渲染Crush标志。将参数设置为true以渲染窄版本，
// 旨在用于侧边栏。
//
// compact参数决定它是为侧边栏渲染紧凑版本，
// 还是为主面板渲染较宽版本。
func Render(s *styles.Styles, version string, compact bool, o Opts) string {
	const charm = " Charm™"

	fg := func(c color.Color, s string) string {
		return lipgloss.NewStyle().Foreground(c).Render(s)
	}

	// 标题。
	const spacing = 1
	letterforms := []letterform{
		letterC,
		letterR,
		letterU,
		letterSStylized,
		letterH,
	}
	stretchIndex := -1 // -1 means no stretching.
	if !compact {
		stretchIndex = cachedRandN(len(letterforms))
	}

	crush := renderWord(spacing, stretchIndex, letterforms...)
	crushWidth := lipgloss.Width(crush)
	b := new(strings.Builder)
	for r := range strings.SplitSeq(crush, "\n") {
		fmt.Fprintln(b, styles.ApplyForegroundGrad(s, r, o.TitleColorA, o.TitleColorB))
	}
	crush = b.String()

	// Charm和版本。
	metaRowGap := 1
	maxVersionWidth := crushWidth - lipgloss.Width(charm) - metaRowGap
	version = ansi.Truncate(version, maxVersionWidth, "…") // 如果版本太长则截断。
	gap := max(0, crushWidth-lipgloss.Width(charm)-lipgloss.Width(version))
	metaRow := fg(o.CharmColor, charm) + strings.Repeat(" ", gap) + fg(o.VersionColor, version)

	// 连接元行和大型Crush标题。
	crush = strings.TrimSpace(metaRow + "\n" + crush)

	// 窄版本。
	if compact {
		field := fg(o.FieldColor, strings.Repeat(diag, crushWidth))
		return strings.Join([]string{field, field, crush, field, ""}, "\n")
	}

	fieldHeight := lipgloss.Height(crush)

	// 左侧区域。
	const leftWidth = 6
	leftFieldRow := fg(o.FieldColor, strings.Repeat(diag, leftWidth))
	leftField := new(strings.Builder)
	for range fieldHeight {
		fmt.Fprintln(leftField, leftFieldRow)
	}

	// 右侧区域。
	rightWidth := max(15, o.Width-crushWidth-leftWidth-2) // 2用于间距。
	const stepDownAt = 0
	rightField := new(strings.Builder)
	for i := range fieldHeight {
		width := rightWidth
		if i >= stepDownAt {
			width = rightWidth - (i - stepDownAt)
		}
		fmt.Fprint(rightField, fg(o.FieldColor, strings.Repeat(diag, width)), "\n")
	}

	// 返回宽版本。
	const hGap = " "
	logo := lipgloss.JoinHorizontal(lipgloss.Top, leftField.String(), hGap, crush, hGap, rightField.String())
	if o.Width > 0 {
		// 将logo截断到指定宽度。
		lines := strings.Split(logo, "\n")
		for i, line := range lines {
			lines[i] = ansi.Truncate(line, o.Width, "")
		}
		logo = strings.Join(lines, "\n")
	}
	return logo
}

// SmallRender 渲染较小版本的Crush标志，适用于
// 较小的窗口或侧边栏使用。
func SmallRender(t *styles.Styles, width int) string {
	title := t.Base.Foreground(t.Secondary).Render("Charm™")
	title = fmt.Sprintf("%s %s", title, styles.ApplyBoldForegroundGrad(t, "Crush", t.Secondary, t.Primary))
	remainingWidth := width - lipgloss.Width(title) - 1 // 1用于"Crush"后面的空格
	if remainingWidth > 0 {
		lines := strings.Repeat("╱", remainingWidth)
		title = fmt.Sprintf("%s %s", title, t.Base.Foreground(t.Primary).Render(lines))
	}
	return title
}

// renderWord 渲染字母形式以组成单词。stretchIndex是要拉伸的字母的索引，
// 如果没有字母应该拉伸则为-1。
func renderWord(spacing int, stretchIndex int, letterforms ...letterform) string {
	if spacing < 0 {
		spacing = 0
	}

	renderedLetterforms := make([]string, len(letterforms))

	// 随机选择一个字母进行拉伸
	for i, letter := range letterforms {
		renderedLetterforms[i] = letter(i == stretchIndex)
	}

	if spacing > 0 {
		// 在字母之间添加空格并渲染。
		renderedLetterforms = slice.Intersperse(renderedLetterforms, strings.Repeat(" ", spacing))
	}
	return strings.TrimSpace(
		lipgloss.JoinHorizontal(lipgloss.Top, renderedLetterforms...),
	)
}

// letterC 以风格化的方式渲染字母C。它接受一个整数，
// 该整数确定要拉伸多少个单元格。如果拉伸小于1，
// 则默认为不拉伸。
func letterC(stretch bool) string {
	// 这是我们正在制作的：
	//
	// ▄▀▀▀▀
	// █
	//	▀▀▀▀

	left := heredoc.Doc(`
		▄
		█
	`)
	right := heredoc.Doc(`
		▀

		▀
	`)
	return joinLetterform(
		left,
		stretchLetterformPart(right, letterformProps{
			stretch:    stretch,
			width:      4,
			minStretch: 7,
			maxStretch: 12,
		}),
	)
}

// letterH 以风格化的方式渲染字母H。它接受一个整数，
// 该整数确定要拉伸多少个单元格。如果拉伸小于1，
// 则默认为不拉伸。
func letterH(stretch bool) string {
	// 这是我们正在制作的：
	//
	// █   █
	// █▀▀▀█
	// ▀   ▀

	side := heredoc.Doc(`
		█
		█
		▀`)
	middle := heredoc.Doc(`

		▀
	`)
	return joinLetterform(
		side,
		stretchLetterformPart(middle, letterformProps{
			stretch:    stretch,
			width:      3,
			minStretch: 8,
			maxStretch: 12,
		}),
		side,
	)
}

// letterR 以风格化的方式渲染字母R。它接受一个整数，
// 该整数确定要拉伸多少个单元格。如果拉伸小于1，
// 则默认为不拉伸。
func letterR(stretch bool) string {
	// 这是我们正在制作的：
	//
	// █▀▀▀▄
	// █▀▀▀▄
	// ▀   ▀

	left := heredoc.Doc(`
		█
		█
		▀
	`)
	center := heredoc.Doc(`
		▀
		▀
	`)
	right := heredoc.Doc(`
		▄
		▄
		▀
	`)
	return joinLetterform(
		left,
		stretchLetterformPart(center, letterformProps{
			stretch:    stretch,
			width:      3,
			minStretch: 7,
			maxStretch: 12,
		}),
		right,
	)
}

// letterSStylized 以风格化的方式渲染字母S，比[letterS]更具风格化。
// 它接受一个整数，该整数确定要拉伸多少个单元格。如果拉伸小于1，
// 则默认为不拉伸。
func letterSStylized(stretch bool) string {
	// 这是我们正在制作的：
	//
	// ▄▀▀▀▀▀
	// ▀▀▀▀▀█
	// ▀▀▀▀▀

	left := heredoc.Doc(`
		▄
		▀
		▀
	`)
	center := heredoc.Doc(`
		▀
		▀
		▀
	`)
	right := heredoc.Doc(`
		▀
		█
	`)
	return joinLetterform(
		left,
		stretchLetterformPart(center, letterformProps{
			stretch:    stretch,
			width:      3,
			minStretch: 7,
			maxStretch: 12,
		}),
		right,
	)
}

// letterU 以风格化的方式渲染字母U。它接受一个整数，
// 该整数确定要拉伸多少个单元格。如果拉伸小于1，
// 则默认为不拉伸。
func letterU(stretch bool) string {
	// 这是我们正在制作的：
	//
	// █   █
	// █   █
	//	▀▀▀

	side := heredoc.Doc(`
		█
		█
	`)
	middle := heredoc.Doc(`


		▀
	`)
	return joinLetterform(
		side,
		stretchLetterformPart(middle, letterformProps{
			stretch:    stretch,
			width:      3,
			minStretch: 7,
			maxStretch: 12,
		}),
		side,
	)
}

func joinLetterform(letters ...string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, letters...)
}

// letterformProps 定义字母形式拉伸属性。
// 用于提高可读性。
type letterformProps struct {
	width      int
	minStretch int
	maxStretch int
	stretch    bool
}

// stretchLetterformPart 是字母拉伸的辅助函数。如果randomize
// 为false，则使用最小值。
func stretchLetterformPart(s string, p letterformProps) string {
	if p.maxStretch < p.minStretch {
		p.minStretch, p.maxStretch = p.maxStretch, p.minStretch
	}
	n := p.width
	if p.stretch {
		n = cachedRandN(p.maxStretch-p.minStretch) + p.minStretch //nolint:gosec
	}
	parts := make([]string, n)
	for i := range parts {
		parts[i] = s
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}
