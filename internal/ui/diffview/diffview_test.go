package diffview_test

import (
	_ "embed"
	"fmt"
	"strings"
	"testing"

	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/purpose168/crush-cn/internal/ui/diffview"
)

//go:embed testdata/TestDefault.before
// TestDefaultBefore 是默认测试用例的修改前文件内容
var TestDefaultBefore string

//go:embed testdata/TestDefault.after
// TestDefaultAfter 是默认测试用例的修改后文件内容
var TestDefaultAfter string

//go:embed testdata/TestMultipleHunks.before
// TestMultipleHunksBefore 是多个块测试用例的修改前文件内容
var TestMultipleHunksBefore string

//go:embed testdata/TestMultipleHunks.after
// TestMultipleHunksAfter 是多个块测试用例的修改后文件内容
var TestMultipleHunksAfter string

//go:embed testdata/TestNarrow.before
// TestNarrowBefore 是窄宽度测试用例的修改前文件内容
var TestNarrowBefore string

//go:embed testdata/TestNarrow.after
// TestNarrowAfter 是窄宽度测试用例的修改后文件内容
var TestNarrowAfter string

//go:embed testdata/TestTabs.before
// TestTabsBefore 是制表符测试用例的修改前文件内容
var TestTabsBefore string

//go:embed testdata/TestTabs.after
// TestTabsAfter 是制表符测试用例的修改后文件内容
var TestTabsAfter string

//go:embed testdata/TestLineBreakIssue.before
// TestLineBreakIssueBefore 是换行问题测试用例的修改前文件内容
var TestLineBreakIssueBefore string

//go:embed testdata/TestLineBreakIssue.after
// TestLineBreakIssueAfter 是换行问题测试用例的修改后文件内容
var TestLineBreakIssueAfter string

type (
	// TestFunc 定义测试函数类型，用于配置 DiffView
	TestFunc func(dv *diffview.DiffView) *diffview.DiffView
	// TestFuncs 是测试函数的映射集合
	TestFuncs map[string]TestFunc
)

var (
	// UnifiedFunc 设置为统一布局的测试函数
	UnifiedFunc = func(dv *diffview.DiffView) *diffview.DiffView {
		return dv.Unified()
	}
	// SplitFunc 设置为分屏布局的测试函数
	SplitFunc = func(dv *diffview.DiffView) *diffview.DiffView {
		return dv.Split()
	}

	// DefaultFunc 使用默认设置的测试函数
	DefaultFunc = func(dv *diffview.DiffView) *diffview.DiffView {
		return dv.
			Before("main.go", TestDefaultBefore).
			After("main.go", TestDefaultAfter)
	}
	// NoLineNumbersFunc 禁用行号显示的测试函数
	NoLineNumbersFunc = func(dv *diffview.DiffView) *diffview.DiffView {
		return dv.
			Before("main.go", TestDefaultBefore).
			After("main.go", TestDefaultAfter).
			LineNumbers(false)
	}
	// MultipleHunksFunc 使用多个块的测试函数
	MultipleHunksFunc = func(dv *diffview.DiffView) *diffview.DiffView {
		return dv.
			Before("main.go", TestMultipleHunksBefore).
			After("main.go", TestMultipleHunksAfter)
	}
	// CustomContextLinesFunc 使用自定义上下文行数的测试函数
	CustomContextLinesFunc = func(dv *diffview.DiffView) *diffview.DiffView {
		return dv.
			Before("main.go", TestMultipleHunksBefore).
			After("main.go", TestMultipleHunksAfter).
			ContextLines(4)
	}
	// NarrowFunc 使用窄宽度内容的测试函数
	NarrowFunc = func(dv *diffview.DiffView) *diffview.DiffView {
		return dv.
			Before("text.txt", TestNarrowBefore).
			After("text.txt", TestNarrowAfter)
	}
	// SmallWidthFunc 使用小宽度的测试函数
	SmallWidthFunc = func(dv *diffview.DiffView) *diffview.DiffView {
		return dv.
			Before("main.go", TestMultipleHunksBefore).
			After("main.go", TestMultipleHunksAfter).
			Width(40)
	}
	// LargeWidthFunc 使用大宽度的测试函数
	LargeWidthFunc = func(dv *diffview.DiffView) *diffview.DiffView {
		return dv.
			Before("main.go", TestMultipleHunksBefore).
			After("main.go", TestMultipleHunksAfter).
			Width(120)
	}
	// NoSyntaxHighlightFunc 禁用语法高亮的测试函数
	NoSyntaxHighlightFunc = func(dv *diffview.DiffView) *diffview.DiffView {
		return dv.
			Before("main.go", TestMultipleHunksBefore).
			After("main.go", TestMultipleHunksAfter).
			ChromaStyle(nil)
	}

	// LightModeFunc 使用浅色主题的测试函数
	LightModeFunc = func(dv *diffview.DiffView) *diffview.DiffView {
		return dv.
			Style(diffview.DefaultLightStyle()).
			ChromaStyle(styles.Get("catppuccin-latte"))
	}
	// DarkModeFunc 使用深色主题的测试函数
	DarkModeFunc = func(dv *diffview.DiffView) *diffview.DiffView {
		return dv.
			Style(diffview.DefaultDarkStyle()).
			ChromaStyle(styles.Get("catppuccin-macchiato"))
	}

	// LayoutFuncs 布局测试函数集合
	LayoutFuncs = TestFuncs{
		"Unified": UnifiedFunc,
		"Split":   SplitFunc,
	}
	// BehaviorFuncs 行为测试函数集合
	BehaviorFuncs = TestFuncs{
		"Default":            DefaultFunc,
		"NoLineNumbers":      NoLineNumbersFunc,
		"MultipleHunks":      MultipleHunksFunc,
		"CustomContextLines": CustomContextLinesFunc,
		"Narrow":             NarrowFunc,
		"SmallWidth":         SmallWidthFunc,
		"LargeWidth":         LargeWidthFunc,
		"NoSyntaxHighlight":  NoSyntaxHighlightFunc,
	}
	// ThemeFuncs 主题测试函数集合
	ThemeFuncs = TestFuncs{
		"LightMode": LightModeFunc,
		"DarkMode":  DarkModeFunc,
	}
)

// TestDiffView 测试差异视图的各种组合配置
func TestDiffView(t *testing.T) {
	for layoutName, layoutFunc := range LayoutFuncs {
		t.Run(layoutName, func(t *testing.T) {
			for behaviorName, behaviorFunc := range BehaviorFuncs {
				t.Run(behaviorName, func(t *testing.T) {
					for themeName, themeFunc := range ThemeFuncs {
						t.Run(themeName, func(t *testing.T) {
							t.Parallel()

							dv := diffview.New()
							dv = layoutFunc(dv)
							dv = themeFunc(dv)
							dv = behaviorFunc(dv)

							output := dv.String()
							golden.RequireEqual(t, []byte(output))

							switch behaviorName {
							case "SmallWidth":
								assertLineWidth(t, 40, output)
							case "LargeWidth":
								assertLineWidth(t, 120, output)
							}
						})
					}
				})
			}
		})
	}
}

// TestDiffViewTabs 测试制表符处理
func TestDiffViewTabs(t *testing.T) {
	t.Parallel()

	for layoutName, layoutFunc := range LayoutFuncs {
		t.Run(layoutName, func(t *testing.T) {
			t.Parallel()

			dv := diffview.New().
				Before("main.go", TestTabsBefore).
				After("main.go", TestTabsAfter).
				Style(diffview.DefaultLightStyle()).
				ChromaStyle(styles.Get("catppuccin-latte"))
			dv = layoutFunc(dv)

			output := dv.String()
			golden.RequireEqual(t, []byte(output))
		})
	}
}

// TestDiffViewLineBreakIssue 测试换行问题处理
func TestDiffViewLineBreakIssue(t *testing.T) {
	t.Parallel()

	for layoutName, layoutFunc := range LayoutFuncs {
		t.Run(layoutName, func(t *testing.T) {
			t.Parallel()

			dv := diffview.New().
				Before("index.js", TestLineBreakIssueBefore).
				After("index.js", TestLineBreakIssueAfter).
				Style(diffview.DefaultLightStyle()).
				ChromaStyle(styles.Get("catppuccin-latte"))
			dv = layoutFunc(dv)

			output := dv.String()
			golden.RequireEqual(t, []byte(output))
		})
	}
}

// TestDiffViewWidth 测试不同宽度的差异视图
func TestDiffViewWidth(t *testing.T) {
	for layoutName, layoutFunc := range LayoutFuncs {
		t.Run(layoutName, func(t *testing.T) {
			for width := 1; width <= 110; width++ {
				if layoutName == "Unified" && width > 60 {
					continue
				}

				t.Run(fmt.Sprintf("WidthOf%03d", width), func(t *testing.T) {
					t.Parallel()

					dv := diffview.New().
						Before("main.go", TestMultipleHunksBefore).
						After("main.go", TestMultipleHunksAfter).
						Width(width).
						Style(diffview.DefaultLightStyle()).
						ChromaStyle(styles.Get("catppuccin-latte"))
					dv = layoutFunc(dv)

					output := dv.String()
					golden.RequireEqual(t, []byte(output))

					assertLineWidth(t, width, output)
				})
			}
		})
	}
}

// TestDiffViewHeight 测试不同高度的差异视图
func TestDiffViewHeight(t *testing.T) {
	for layoutName, layoutFunc := range LayoutFuncs {
		t.Run(layoutName, func(t *testing.T) {
			for height := 1; height <= 20; height++ {
				t.Run(fmt.Sprintf("HeightOf%03d", height), func(t *testing.T) {
					t.Parallel()

					dv := diffview.New().
						Before("main.go", TestMultipleHunksBefore).
						After("main.go", TestMultipleHunksAfter).
						Height(height).
						Style(diffview.DefaultLightStyle()).
						ChromaStyle(styles.Get("catppuccin-latte"))
					dv = layoutFunc(dv)

					output := dv.String()
					golden.RequireEqual(t, []byte(output))
				})
			}
		})
	}
}

// TestDiffViewXOffset 测试水平偏移
func TestDiffViewXOffset(t *testing.T) {
	for layoutName, layoutFunc := range LayoutFuncs {
		t.Run(layoutName, func(t *testing.T) {
			for xOffset := range 21 {
				t.Run(fmt.Sprintf("XOffsetOf%02d", xOffset), func(t *testing.T) {
					t.Parallel()

					dv := diffview.New().
						Before("main.go", TestDefaultBefore).
						After("main.go", TestDefaultAfter).
						Style(diffview.DefaultLightStyle()).
						ChromaStyle(styles.Get("catppuccin-latte")).
						Width(60).
						XOffset(xOffset)
					dv = layoutFunc(dv)

					output := dv.String()
					golden.RequireEqual(t, []byte(output))

					assertLineWidth(t, 60, output)
				})
			}
		})
	}
}

// TestDiffViewYOffset 测试垂直偏移
func TestDiffViewYOffset(t *testing.T) {
	for layoutName, layoutFunc := range LayoutFuncs {
		t.Run(layoutName, func(t *testing.T) {
			for yOffset := range 17 {
				t.Run(fmt.Sprintf("YOffsetOf%02d", yOffset), func(t *testing.T) {
					t.Parallel()

					dv := diffview.New().
						Before("main.go", TestMultipleHunksBefore).
						After("main.go", TestMultipleHunksAfter).
						Style(diffview.DefaultLightStyle()).
						ChromaStyle(styles.Get("catppuccin-latte")).
						Height(5).
						YOffset(yOffset)
					dv = layoutFunc(dv)

					output := dv.String()
					golden.RequireEqual(t, []byte(output))
				})
			}
		})
	}
}

// TestDiffViewYOffsetInfinite 测试无限垂直滚动
func TestDiffViewYOffsetInfinite(t *testing.T) {
	for layoutName, layoutFunc := range LayoutFuncs {
		t.Run(layoutName, func(t *testing.T) {
			for yOffset := range 17 {
				t.Run(fmt.Sprintf("YOffsetOf%02d", yOffset), func(t *testing.T) {
					t.Parallel()

					dv := diffview.New().
						Before("main.go", TestMultipleHunksBefore).
						After("main.go", TestMultipleHunksAfter).
						Style(diffview.DefaultLightStyle()).
						ChromaStyle(styles.Get("catppuccin-latte")).
						Height(5).
						YOffset(yOffset).
						InfiniteYScroll(true)
					dv = layoutFunc(dv)

					output := dv.String()
					golden.RequireEqual(t, []byte(output))
				})
			}
		})
	}
}

// assertLineWidth 断言输出行的宽度符合预期
func assertLineWidth(t *testing.T, expected int, output string) {
	var lineWidth int
	for line := range strings.SplitSeq(output, "\n") {
		lineWidth = max(lineWidth, ansi.StringWidth(line))
	}
	if lineWidth != expected {
		t.Errorf("期望输出宽度为 %d，实际为 %d", expected, lineWidth)
	}
}

// assertHeight 断言输出高度符合预期
func assertHeight(t *testing.T, expected int, output string) {
	output = strings.TrimSuffix(output, "\n")
	lines := strings.Count(output, "\n") + 1
	if lines != expected {
		t.Errorf("期望输出高度为 %d，实际为 %d", expected, lines)
	}
}
