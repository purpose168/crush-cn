package diffview

import (
	"slices"

	"github.com/aymanbagabas/go-udiff"
	"github.com/charmbracelet/x/exp/slice"
)

type splitHunk struct {
	fromLine int
	toLine   int
	lines    []*splitLine
}

type splitLine struct {
	before *udiff.Line
	after  *udiff.Line
}

func hunkToSplit(h *udiff.Hunk) (sh splitHunk) {
	lines := slices.Clone(h.Lines)
	sh = splitHunk{
		fromLine: h.FromLine,
		toLine:   h.ToLine,
		lines:    make([]*splitLine, 0, len(lines)),
	}

	for {
		var ul udiff.Line
		var ok bool
		ul, lines, ok = slice.Shift(lines)
		if !ok {
			break
		}

		var sl splitLine

		switch ul.Kind {
		// 对于相等的行，原样添加
		case udiff.Equal:
			sl.before = &ul
			sl.after = &ul

		// 对于插入的行，设置after并将before保持为nil
		case udiff.Insert:
			sl.before = nil
			sl.after = &ul

		// 对于删除的行，设置before并遍历后续行搜索等效的after行。
		case udiff.Delete:
			sl.before = &ul

		inner:
			for i, l := range lines {
				switch l.Kind {
				case udiff.Insert:
					var ll udiff.Line
					ll, lines, _ = slice.DeleteAt(lines, i)
					sl.after = &ll
					break inner
				case udiff.Equal:
					break inner
				}
			}
		}

		sh.lines = append(sh.lines, &sl)
	}

	return sh
}
