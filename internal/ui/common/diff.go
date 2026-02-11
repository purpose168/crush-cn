package common

import (
	"github.com/alecthomas/chroma/v2"
	"github.com/charmbracelet/crush/internal/ui/diffview"
	"github.com/charmbracelet/crush/internal/ui/styles"
)

// DiffFormatter 返回一个使用给定样式的差异格式化器，可用于格式化差异输出。
func DiffFormatter(s *styles.Styles) *diffview.DiffView {
	formatDiff := diffview.New()
	style := chroma.MustNewStyle("crush", s.ChromaTheme())
	diff := formatDiff.ChromaStyle(style).Style(s.Diff).TabWidth(4)
	return diff
}
