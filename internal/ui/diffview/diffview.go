package diffview

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/aymanbagabas/go-udiff"
	"github.com/charmbracelet/x/ansi"
	"github.com/zeebo/xxh3"
)

const (
	leadingSymbolsSize = 2
	lineNumPadding     = 1
)

type file struct {
	path    string
	content string
}

type layout int

const (
	layoutUnified layout = iota + 1
	layoutSplit
)

// DiffView 表示用于显示两个文件之间差异的视图。
type DiffView struct {
	layout          layout
	before          file
	after           file
	contextLines    int
	lineNumbers     bool
	height          int
	width           int
	xOffset         int
	yOffset         int
	infiniteYScroll bool
	style           Style
	tabWidth        int
	chromaStyle     *chroma.Style

	isComputed bool
	err        error
	unified    udiff.UnifiedDiff
	edits      []udiff.Edit

	splitHunks []splitHunk

	totalLines      int
	codeWidth       int
	fullCodeWidth   int  // 包含前导符号
	extraColOnAfter bool // 在after面板上添加额外列
	beforeNumDigits int
	afterNumDigits  int

	// 缓存词法分析器以避免在每一行上进行昂贵的文件模式匹配
	cachedLexer chroma.Lexer

	// 缓存高亮显示的行以避免重新高亮显示相同内容
	// 键：（内容+背景色）的哈希，值：高亮显示的字符串
	syntaxCache map[string]string
}

// New 使用默认设置创建一个新的DiffView。
func New() *DiffView {
	dv := &DiffView{
		layout:       layoutUnified,
		contextLines: udiff.DefaultContextLines,
		lineNumbers:  true,
		tabWidth:     8,
		syntaxCache:  make(map[string]string),
	}
	dv.style = DefaultDarkStyle()
	return dv
}

// Unified 将DiffView的布局设置为统一视图。
func (dv *DiffView) Unified() *DiffView {
	dv.layout = layoutUnified
	return dv
}

// Split 将DiffView的布局设置为分屏（并排）视图。
func (dv *DiffView) Split() *DiffView {
	dv.layout = layoutSplit
	return dv
}

// Before 为DiffView设置"before"文件。
func (dv *DiffView) Before(path, content string) *DiffView {
	dv.before = file{path: path, content: content}
	// 当内容更改时清除缓存
	dv.clearCaches()
	return dv
}

// After 为DiffView设置"after"文件。
func (dv *DiffView) After(path, content string) *DiffView {
	dv.after = file{path: path, content: content}
	// 当内容更改时清除缓存
	dv.clearCaches()
	return dv
}

// clearCaches 当内容或主要设置更改时清除所有缓存。
func (dv *DiffView) clearCaches() {
	dv.cachedLexer = nil
	dv.clearSyntaxCache()
	dv.isComputed = false
}

// ContextLines 为DiffView设置上下文行数。
func (dv *DiffView) ContextLines(contextLines int) *DiffView {
	dv.contextLines = contextLines
	return dv
}

// Style 为DiffView设置样式。
func (dv *DiffView) Style(style Style) *DiffView {
	dv.style = style
	return dv
}

// LineNumbers 设置是否在DiffView中显示行号。
func (dv *DiffView) LineNumbers(lineNumbers bool) *DiffView {
	dv.lineNumbers = lineNumbers
	return dv
}

// Height 设置DiffView的高度。
func (dv *DiffView) Height(height int) *DiffView {
	dv.height = height
	return dv
}

// Width 设置DiffView的宽度。
func (dv *DiffView) Width(width int) *DiffView {
	dv.width = width
	return dv
}

// XOffset 为DiffView设置水平偏移。
func (dv *DiffView) XOffset(xOffset int) *DiffView {
	dv.xOffset = xOffset
	return dv
}

// YOffset 为DiffView设置垂直偏移。
func (dv *DiffView) YOffset(yOffset int) *DiffView {
	dv.yOffset = yOffset
	return dv
}

// InfiniteYScroll 允许YOffset滚动到最后一行之外。
func (dv *DiffView) InfiniteYScroll(infiniteYScroll bool) *DiffView {
	dv.infiniteYScroll = infiniteYScroll
	return dv
}

// TabWidth 设置制表符宽度。仅对包含制表符的代码（如Go代码）相关。
func (dv *DiffView) TabWidth(tabWidth int) *DiffView {
	dv.tabWidth = tabWidth
	return dv
}

// ChromaStyle 设置语法高亮的chroma样式。
// 如果为nil，则不应用语法高亮。
func (dv *DiffView) ChromaStyle(style *chroma.Style) *DiffView {
	dv.chromaStyle = style
	// 当样式更改时清除语法缓存，因为高亮显示将不同
	dv.clearSyntaxCache()
	return dv
}

// clearSyntaxCache 清除语法高亮缓存。
func (dv *DiffView) clearSyntaxCache() {
	if dv.syntaxCache != nil {
		// 清除映射但保持其分配
		for k := range dv.syntaxCache {
			delete(dv.syntaxCache, k)
		}
	}
}

// String 返回DiffView的字符串表示。
func (dv *DiffView) String() string {
	dv.normalizeLineEndings()
	dv.replaceTabs()
	if err := dv.computeDiff(); err != nil {
		return err.Error()
	}
	dv.convertDiffToSplit()
	dv.adjustStyles()
	dv.detectNumDigits()
	dv.detectTotalLines()
	dv.preventInfiniteYScroll()

	if dv.width <= 0 {
		dv.detectCodeWidth()
	} else {
		dv.resizeCodeWidth()
	}

	style := lipgloss.NewStyle()
	if dv.width > 0 {
		style = style.MaxWidth(dv.width)
	}
	if dv.height > 0 {
		style = style.MaxHeight(dv.height)
	}

	switch dv.layout {
	case layoutUnified:
		return style.Render(strings.TrimSuffix(dv.renderUnified(), "\n"))
	case layoutSplit:
		return style.Render(strings.TrimSuffix(dv.renderSplit(), "\n"))
	default:
		panic("unknown diffview layout")
	}
}

// normalizeLineEndings 确保文件内容使用Unix风格的行结束符。
func (dv *DiffView) normalizeLineEndings() {
	dv.before.content = strings.ReplaceAll(dv.before.content, "\r\n", "\n")
	dv.after.content = strings.ReplaceAll(dv.after.content, "\r\n", "\n")
}

// replaceTabs 根据指定的制表符宽度，将before和after文件内容中的制表符替换为空格。
func (dv *DiffView) replaceTabs() {
	spaces := strings.Repeat(" ", dv.tabWidth)
	dv.before.content = strings.ReplaceAll(dv.before.content, "\t", spaces)
	dv.after.content = strings.ReplaceAll(dv.after.content, "\t", spaces)
}

// computeDiff 计算"before"和"after"文件之间的差异。
func (dv *DiffView) computeDiff() error {
	if dv.isComputed {
		return dv.err
	}
	dv.isComputed = true
	dv.edits = udiff.Strings(
		dv.before.content,
		dv.after.content,
	)
	dv.unified, dv.err = udiff.ToUnifiedDiff(
		dv.before.path,
		dv.after.path,
		dv.before.content,
		dv.edits,
		dv.contextLines,
	)
	return dv.err
}

// convertDiffToSplit 如果布局设置为分屏，则将统一差异转换为分屏差异。
func (dv *DiffView) convertDiffToSplit() {
	if dv.layout != layoutSplit {
		return
	}

	dv.splitHunks = make([]splitHunk, len(dv.unified.Hunks))
	for i, h := range dv.unified.Hunks {
		dv.splitHunks[i] = hunkToSplit(h)
	}
}

// adjustStyles 调整添加填充和对齐到样式。
func (dv *DiffView) adjustStyles() {
	setPadding := func(s lipgloss.Style) lipgloss.Style {
		return s.Padding(0, lineNumPadding).Align(lipgloss.Right)
	}
	dv.style.MissingLine.LineNumber = setPadding(dv.style.MissingLine.LineNumber)
	dv.style.DividerLine.LineNumber = setPadding(dv.style.DividerLine.LineNumber)
	dv.style.EqualLine.LineNumber = setPadding(dv.style.EqualLine.LineNumber)
	dv.style.InsertLine.LineNumber = setPadding(dv.style.InsertLine.LineNumber)
	dv.style.DeleteLine.LineNumber = setPadding(dv.style.DeleteLine.LineNumber)
}

// detectNumDigits 计算before和after行号所需的最大位数。
func (dv *DiffView) detectNumDigits() {
	dv.beforeNumDigits = 0
	dv.afterNumDigits = 0

	for _, h := range dv.unified.Hunks {
		dv.beforeNumDigits = max(dv.beforeNumDigits, len(strconv.Itoa(h.FromLine+len(h.Lines))))
		dv.afterNumDigits = max(dv.afterNumDigits, len(strconv.Itoa(h.ToLine+len(h.Lines))))
	}
}

func (dv *DiffView) detectTotalLines() {
	dv.totalLines = 0

	switch dv.layout {
	case layoutUnified:
		for _, h := range dv.unified.Hunks {
			dv.totalLines += 1 + len(h.Lines)
		}
	case layoutSplit:
		for _, h := range dv.splitHunks {
			dv.totalLines += 1 + len(h.lines)
		}
	}
}

func (dv *DiffView) preventInfiniteYScroll() {
	if dv.infiniteYScroll {
		return
	}

	// 限制yOffset以防止滚动到最后一行之外
	if dv.height > 0 {
		maxYOffset := max(0, dv.totalLines-dv.height)
		dv.yOffset = min(dv.yOffset, maxYOffset)
	} else {
		// 如果没有高度限制，确保yOffset不超过总行数
		dv.yOffset = min(dv.yOffset, max(0, dv.totalLines-1))
	}
	dv.yOffset = max(0, dv.yOffset) // 确保yOffset不为负数
}

// detectCodeWidth 计算差异视图中代码行的最大宽度。
func (dv *DiffView) detectCodeWidth() {
	switch dv.layout {
	case layoutUnified:
		dv.detectUnifiedCodeWidth()
	case layoutSplit:
		dv.detectSplitCodeWidth()
	}
	dv.fullCodeWidth = dv.codeWidth + leadingSymbolsSize
}

// detectUnifiedCodeWidth 计算统一差异中代码行的最大宽度。
func (dv *DiffView) detectUnifiedCodeWidth() {
	dv.codeWidth = 0

	for _, h := range dv.unified.Hunks {
		shownLines := ansi.StringWidth(dv.hunkLineFor(h))

		for _, l := range h.Lines {
			lineWidth := ansi.StringWidth(strings.TrimSuffix(l.Content, "\n")) + 1
			dv.codeWidth = max(dv.codeWidth, lineWidth, shownLines)
		}
	}
}

// detectSplitCodeWidth 计算分屏差异中代码行的最大宽度。
func (dv *DiffView) detectSplitCodeWidth() {
	dv.codeWidth = 0

	for i, h := range dv.splitHunks {
		shownLines := ansi.StringWidth(dv.hunkLineFor(dv.unified.Hunks[i]))

		for _, l := range h.lines {
			if l.before != nil {
				codeWidth := ansi.StringWidth(strings.TrimSuffix(l.before.Content, "\n")) + 1
				dv.codeWidth = max(dv.codeWidth, codeWidth, shownLines)
			}
			if l.after != nil {
				codeWidth := ansi.StringWidth(strings.TrimSuffix(l.after.Content, "\n")) + 1
				dv.codeWidth = max(dv.codeWidth, codeWidth, shownLines)
			}
		}
	}
}

// resizeCodeWidth 调整代码宽度以适应指定的宽度。
func (dv *DiffView) resizeCodeWidth() {
	fullNumWidth := dv.beforeNumDigits + dv.afterNumDigits
	fullNumWidth += lineNumPadding * 4 // left and right padding for both line numbers

	switch dv.layout {
	case layoutUnified:
		dv.codeWidth = dv.width - fullNumWidth - leadingSymbolsSize
	case layoutSplit:
		remainingWidth := dv.width - fullNumWidth - leadingSymbolsSize*2
		dv.codeWidth = remainingWidth / 2
		dv.extraColOnAfter = isOdd(remainingWidth)
	}

	dv.fullCodeWidth = dv.codeWidth + leadingSymbolsSize
}

// renderUnified 将统一差异视图渲染为字符串。
func (dv *DiffView) renderUnified() string {
	var b strings.Builder

	fullContentStyle := lipgloss.NewStyle().MaxWidth(dv.fullCodeWidth)
	printedLines := -dv.yOffset
	shouldWrite := func() bool { return printedLines >= 0 }

	getContent := func(in string, ls LineStyle) (content string, leadingEllipsis bool) {
		content = strings.TrimSuffix(in, "\n")
		content = dv.hightlightCode(content, ls.Code.GetBackground())
		content = ansi.GraphemeWidth.Cut(content, dv.xOffset, len(content))
		content = ansi.Truncate(content, dv.codeWidth, "…")
		leadingEllipsis = dv.xOffset > 0 && strings.TrimSpace(content) != ""
		return content, leadingEllipsis
	}

outer:
	for i, h := range dv.unified.Hunks {
		if shouldWrite() {
			ls := dv.style.DividerLine
			if dv.lineNumbers {
				b.WriteString(ls.LineNumber.Render(pad("…", dv.beforeNumDigits)))
				b.WriteString(ls.LineNumber.Render(pad("…", dv.afterNumDigits)))
			}
			content := ansi.Truncate(dv.hunkLineFor(h), dv.fullCodeWidth, "…")
			b.WriteString(ls.Code.Width(dv.fullCodeWidth).Render(content))
			b.WriteString("\n")
		}
		printedLines++

		beforeLine := h.FromLine
		afterLine := h.ToLine

		for j, l := range h.Lines {
			// 如果我们没有足够的空间来打印差异的其余部分，则打印省略号
			hasReachedHeight := dv.height > 0 && printedLines+1 == dv.height
			isLastHunk := i+1 == len(dv.unified.Hunks)
			isLastLine := j+1 == len(h.Lines)
			if hasReachedHeight && (!isLastHunk || !isLastLine) {
				if shouldWrite() {
					ls := dv.lineStyleForType(l.Kind)
					if dv.lineNumbers {
						b.WriteString(ls.LineNumber.Render(pad("…", dv.beforeNumDigits)))
						b.WriteString(ls.LineNumber.Render(pad("…", dv.afterNumDigits)))
					}
					b.WriteString(fullContentStyle.Render(
						ls.Code.Width(dv.fullCodeWidth).Render("  …"),
					))
					b.WriteRune('\n')
				}
				break outer
			}

			switch l.Kind {
			case udiff.Equal:
				if shouldWrite() {
					ls := dv.style.EqualLine
					content, leadingEllipsis := getContent(l.Content, ls)
					if dv.lineNumbers {
						b.WriteString(ls.LineNumber.Render(pad(beforeLine, dv.beforeNumDigits)))
						b.WriteString(ls.LineNumber.Render(pad(afterLine, dv.afterNumDigits)))
					}
					b.WriteString(fullContentStyle.Render(
						ls.Code.Width(dv.fullCodeWidth).Render(ternary(leadingEllipsis, " …", "  ") + content),
					))
				}
				beforeLine++
				afterLine++
			case udiff.Insert:
				if shouldWrite() {
					ls := dv.style.InsertLine
					content, leadingEllipsis := getContent(l.Content, ls)
					if dv.lineNumbers {
						b.WriteString(ls.LineNumber.Render(pad(" ", dv.beforeNumDigits)))
						b.WriteString(ls.LineNumber.Render(pad(afterLine, dv.afterNumDigits)))
					}
					b.WriteString(fullContentStyle.Render(
						ls.Symbol.Render(ternary(leadingEllipsis, "+…", "+ ")) +
							ls.Code.Width(dv.codeWidth).Render(content),
					))
				}
				afterLine++
			case udiff.Delete:
				if shouldWrite() {
					ls := dv.style.DeleteLine
					content, leadingEllipsis := getContent(l.Content, ls)
					if dv.lineNumbers {
						b.WriteString(ls.LineNumber.Render(pad(beforeLine, dv.beforeNumDigits)))
						b.WriteString(ls.LineNumber.Render(pad(" ", dv.afterNumDigits)))
					}
					b.WriteString(fullContentStyle.Render(
						ls.Symbol.Render(ternary(leadingEllipsis, "-…", "- ")) +
							ls.Code.Width(dv.codeWidth).Render(content),
					))
				}
				beforeLine++
			}
			if shouldWrite() {
				b.WriteRune('\n')
			}

			printedLines++
		}
	}

	return b.String()
}

// renderSplit 将分屏（并排）差异视图渲染为字符串。
func (dv *DiffView) renderSplit() string {
	var b strings.Builder

	beforeFullContentStyle := lipgloss.NewStyle().MaxWidth(dv.fullCodeWidth)
	afterFullContentStyle := lipgloss.NewStyle().MaxWidth(dv.fullCodeWidth + btoi(dv.extraColOnAfter))
	printedLines := -dv.yOffset
	shouldWrite := func() bool { return printedLines >= 0 }

	getContent := func(in string, ls LineStyle) (content string, leadingEllipsis bool) {
		content = strings.TrimSuffix(in, "\n")
		content = dv.hightlightCode(content, ls.Code.GetBackground())
		content = ansi.GraphemeWidth.Cut(content, dv.xOffset, len(content))
		content = ansi.Truncate(content, dv.codeWidth, "…")
		leadingEllipsis = dv.xOffset > 0 && strings.TrimSpace(content) != ""
		return content, leadingEllipsis
	}

outer:
	for i, h := range dv.splitHunks {
		if shouldWrite() {
			ls := dv.style.DividerLine
			if dv.lineNumbers {
				b.WriteString(ls.LineNumber.Render(pad("…", dv.beforeNumDigits)))
			}
			content := ansi.Truncate(dv.hunkLineFor(dv.unified.Hunks[i]), dv.fullCodeWidth, "…")
			b.WriteString(ls.Code.Width(dv.fullCodeWidth).Render(content))
			if dv.lineNumbers {
				b.WriteString(ls.LineNumber.Render(pad("…", dv.afterNumDigits)))
			}
			b.WriteString(ls.Code.Width(dv.fullCodeWidth + btoi(dv.extraColOnAfter)).Render(" "))
			b.WriteRune('\n')
		}
		printedLines++

		beforeLine := h.fromLine
		afterLine := h.toLine

		for j, l := range h.lines {
			// 如果我们没有足够的空间来打印差异的其余部分，则打印省略号
			hasReachedHeight := dv.height > 0 && printedLines+1 == dv.height
			isLastHunk := i+1 == len(dv.unified.Hunks)
			isLastLine := j+1 == len(h.lines)
			if hasReachedHeight && (!isLastHunk || !isLastLine) {
				if shouldWrite() {
					ls := dv.style.MissingLine
					if l.before != nil {
						ls = dv.lineStyleForType(l.before.Kind)
					}
					if dv.lineNumbers {
						b.WriteString(ls.LineNumber.Render(pad("…", dv.beforeNumDigits)))
					}
					b.WriteString(beforeFullContentStyle.Render(
						ls.Code.Width(dv.fullCodeWidth).Render("  …"),
					))
					ls = dv.style.MissingLine
					if l.after != nil {
						ls = dv.lineStyleForType(l.after.Kind)
					}
					if dv.lineNumbers {
						b.WriteString(ls.LineNumber.Render(pad("…", dv.afterNumDigits)))
					}
					b.WriteString(afterFullContentStyle.Render(
						ls.Code.Width(dv.fullCodeWidth).Render("  …"),
					))
					b.WriteRune('\n')
				}
				break outer
			}

			switch {
			case l.before == nil:
				if shouldWrite() {
					ls := dv.style.MissingLine
					if dv.lineNumbers {
						b.WriteString(ls.LineNumber.Render(pad(" ", dv.beforeNumDigits)))
					}
					b.WriteString(beforeFullContentStyle.Render(
						ls.Code.Width(dv.fullCodeWidth).Render("  "),
					))
				}
			case l.before.Kind == udiff.Equal:
				if shouldWrite() {
					ls := dv.style.EqualLine
					content, leadingEllipsis := getContent(l.before.Content, ls)
					if dv.lineNumbers {
						b.WriteString(ls.LineNumber.Render(pad(beforeLine, dv.beforeNumDigits)))
					}
					b.WriteString(beforeFullContentStyle.Render(
						ls.Code.Width(dv.fullCodeWidth).Render(ternary(leadingEllipsis, " …", "  ") + content),
					))
				}
				beforeLine++
			case l.before.Kind == udiff.Delete:
				if shouldWrite() {
					ls := dv.style.DeleteLine
					content, leadingEllipsis := getContent(l.before.Content, ls)
					if dv.lineNumbers {
						b.WriteString(ls.LineNumber.Render(pad(beforeLine, dv.beforeNumDigits)))
					}
					b.WriteString(beforeFullContentStyle.Render(
						ls.Symbol.Render(ternary(leadingEllipsis, "-…", "- ")) +
							ls.Code.Width(dv.codeWidth).Render(content),
					))
				}
				beforeLine++
			}

			switch {
			case l.after == nil:
				if shouldWrite() {
					ls := dv.style.MissingLine
					if dv.lineNumbers {
						b.WriteString(ls.LineNumber.Render(pad(" ", dv.afterNumDigits)))
					}
					b.WriteString(afterFullContentStyle.Render(
						ls.Code.Width(dv.fullCodeWidth + btoi(dv.extraColOnAfter)).Render("  "),
					))
				}
			case l.after.Kind == udiff.Equal:
				if shouldWrite() {
					ls := dv.style.EqualLine
					content, leadingEllipsis := getContent(l.after.Content, ls)
					if dv.lineNumbers {
						b.WriteString(ls.LineNumber.Render(pad(afterLine, dv.afterNumDigits)))
					}
					b.WriteString(afterFullContentStyle.Render(
						ls.Code.Width(dv.fullCodeWidth + btoi(dv.extraColOnAfter)).Render(ternary(leadingEllipsis, " …", "  ") + content),
					))
				}
				afterLine++
			case l.after.Kind == udiff.Insert:
				if shouldWrite() {
					ls := dv.style.InsertLine
					content, leadingEllipsis := getContent(l.after.Content, ls)
					if dv.lineNumbers {
						b.WriteString(ls.LineNumber.Render(pad(afterLine, dv.afterNumDigits)))
					}
					b.WriteString(afterFullContentStyle.Render(
						ls.Symbol.Render(ternary(leadingEllipsis, "+…", "+ ")) +
							ls.Code.Width(dv.codeWidth+btoi(dv.extraColOnAfter)).Render(content),
					))
				}
				afterLine++
			}

			if shouldWrite() {
				b.WriteRune('\n')
			}

			printedLines++
		}
	}

	return b.String()
}

// hunkLineFor 格式化统一差异视图中块的标题行。
func (dv *DiffView) hunkLineFor(h *udiff.Hunk) string {
	beforeShownLines, afterShownLines := dv.hunkShownLines(h)

	return fmt.Sprintf(
		"  @@ -%d,%d +%d,%d @@ ",
		h.FromLine,
		beforeShownLines,
		h.ToLine,
		afterShownLines,
	)
}

// hunkShownLines 计算before和after版本中显示的行数。
func (dv *DiffView) hunkShownLines(h *udiff.Hunk) (before, after int) {
	for _, l := range h.Lines {
		switch l.Kind {
		case udiff.Equal:
			before++
			after++
		case udiff.Insert:
			after++
		case udiff.Delete:
			before++
		}
	}
	return before, after
}

func (dv *DiffView) lineStyleForType(t udiff.OpKind) LineStyle {
	switch t {
	case udiff.Equal:
		return dv.style.EqualLine
	case udiff.Insert:
		return dv.style.InsertLine
	case udiff.Delete:
		return dv.style.DeleteLine
	default:
		return dv.style.MissingLine
	}
}

func (dv *DiffView) hightlightCode(source string, bgColor color.Color) string {
	if dv.chromaStyle == nil {
		return source
	}

	// 从内容和背景色创建缓存键
	cacheKey := dv.createSyntaxCacheKey(source, bgColor)

	// 检查我们是否已经有这个高亮显示的内容
	if cached, exists := dv.syntaxCache[cacheKey]; exists {
		return cached
	}

	l := dv.getChromaLexer()
	f := dv.getChromaFormatter(bgColor)

	it, err := l.Tokenise(nil, source)
	if err != nil {
		return source
	}

	var b strings.Builder
	if err := f.Format(&b, dv.chromaStyle, it); err != nil {
		return source
	}

	result := b.String()

	// 缓存结果以供将来使用
        dv.syntaxCache[cacheKey] = result

        return result
}

// createSyntaxCacheKey 从源内容和背景色创建缓存键。
// 我们使用简单的哈希来保持合理的内存使用。
func (dv *DiffView) createSyntaxCacheKey(source string, bgColor color.Color) string {
        // 将颜色转换为字符串表示
        r, g, b, a := bgColor.RGBA()
        colorStr := fmt.Sprintf("%d,%d,%d,%d", r, g, b, a)

        // 创建内容+颜色的哈希作为缓存键
        h := xxh3.New()
        h.Write([]byte(source))
        h.Write([]byte(colorStr))
        return fmt.Sprintf("%x", h.Sum(nil))
}

func (dv *DiffView) getChromaLexer() chroma.Lexer {
	if dv.cachedLexer != nil {
		return dv.cachedLexer
	}

	l := lexers.Match(dv.before.path)
	if l == nil {
		l = lexers.Analyse(dv.before.content)
	}
	if l == nil {
		l = lexers.Fallback
	}
	dv.cachedLexer = chroma.Coalesce(l)
	return dv.cachedLexer
}

func (dv *DiffView) getChromaFormatter(bgColor color.Color) chroma.Formatter {
	return chromaFormatter{
		bgColor: bgColor,
	}
}
