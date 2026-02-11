package dialog

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/charmbracelet/crush/internal/ui/list"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/sahilm/fuzzy"
)

// ModelsList 是一个专门用于模型项目和组的列表。
type ModelsList struct {
	*list.List
	groups []ModelGroup
	query  string
	t      *styles.Styles
}

// NewModelsList 创建一个适合模型项目和组的新列表。
func NewModelsList(sty *styles.Styles, groups ...ModelGroup) *ModelsList {
	f := &ModelsList{
		List:   list.NewList(),
		groups: groups,
		t:      sty,
	}
	f.RegisterRenderCallback(list.FocusedRenderCallback(f.List))
	return f
}

// Len 返回所有组中的模型项目数量。
func (f *ModelsList) Len() int {
	n := 0
	for _, g := range f.groups {
		n += len(g.Items)
	}
	return n
}

// SetGroups 设置模型组并更新列表项目。
func (f *ModelsList) SetGroups(groups ...ModelGroup) {
	f.groups = groups
	items := []list.Item{}
	for _, g := range f.groups {
		items = append(items, &g)
		for _, item := range g.Items {
			items = append(items, item)
		}
		// 在每个提供者部分后添加一个空格分隔符
		items = append(items, list.NewSpacerItem(1))
	}
	f.SetItems(items...)
}

// SetFilter 设置过滤查询并更新列表项目。
func (f *ModelsList) SetFilter(q string) {
	f.query = q
	f.SetItems(f.VisibleItems()...)
}

// SetSelected 设置选中的项目索引。它重写基类方法以跳过非模型项目。
func (f *ModelsList) SetSelected(index int) {
	if index < 0 || index >= f.Len() {
		f.List.SetSelected(index)
		return
	}

	f.List.SetSelected(index)
	for {
		selectedItem := f.SelectedItem()
		if _, ok := selectedItem.(*ModelItem); ok {
			return
		}
		f.List.SetSelected(index + 1)
		index++
		if index >= f.Len() {
			return
		}
	}
}

// SetSelectedItem 通过项目 ID 设置列表中的选中项目。
func (f *ModelsList) SetSelectedItem(itemID string) {
	if itemID == "" {
		f.SetSelected(0)
		return
	}

	count := 0
	for _, g := range f.groups {
		for _, item := range g.Items {
			if item.ID() == itemID {
				f.SetSelected(count)
				return
			}
			count++
		}
	}
}

// SelectNext 选择下一个模型项目，跳过任何不可聚焦的项目，如组标题和分隔符。
func (f *ModelsList) SelectNext() (v bool) {
	v = f.List.SelectNext()
	for v {
		selectedItem := f.SelectedItem()
		if _, ok := selectedItem.(*ModelItem); ok {
			return v
		}
		v = f.List.SelectNext()
	}
	return v
}

// SelectPrev 选择上一个模型项目，跳过任何不可聚焦的项目，如组标题和分隔符。
func (f *ModelsList) SelectPrev() (v bool) {
	v = f.List.SelectPrev()
	for v {
		selectedItem := f.SelectedItem()
		if _, ok := selectedItem.(*ModelItem); ok {
			return v
		}
		v = f.List.SelectPrev()
	}
	return v
}

// SelectFirst 选择列表中的第一个模型项目。
func (f *ModelsList) SelectFirst() (v bool) {
	v = f.List.SelectFirst()
	for v {
		selectedItem := f.SelectedItem()
		_, ok := selectedItem.(*ModelItem)
		if ok {
			return v
		}
		v = f.List.SelectNext()
	}
	return v
}

// SelectLast 选择列表中的最后一个模型项目。
func (f *ModelsList) SelectLast() (v bool) {
	v = f.List.SelectLast()
	for v {
		selectedItem := f.SelectedItem()
		if _, ok := selectedItem.(*ModelItem); ok {
			return v
		}
		v = f.List.SelectPrev()
	}
	return v
}

// IsSelectedFirst 检查选中项目是否是第一个模型项目。
func (f *ModelsList) IsSelectedFirst() bool {
	originalIndex := f.Selected()
	f.SelectFirst()
	isFirst := f.Selected() == originalIndex
	f.List.SetSelected(originalIndex)
	return isFirst
}

// IsSelectedLast 检查选中项目是否是最后一个模型项目。
func (f *ModelsList) IsSelectedLast() bool {
	originalIndex := f.Selected()
	f.SelectLast()
	isLast := f.Selected() == originalIndex
	f.List.SetSelected(originalIndex)
	return isLast
}

// VisibleItems 返回过滤后的可见项目。
func (f *ModelsList) VisibleItems() []list.Item {
	query := strings.ToLower(strings.ReplaceAll(f.query, " ", ""))

	if query == "" {
		// 无过滤，返回所有项目及组标题
		items := []list.Item{}
		for _, g := range f.groups {
			items = append(items, &g)
			for _, item := range g.Items {
				item.SetMatch(fuzzy.Match{})
				items = append(items, item)
			}
			// 在每个提供者部分后添加一个空格分隔符
			items = append(items, list.NewSpacerItem(1))
		}
		return items
	}

	filterableItems := make([]list.FilterableItem, 0, f.Len())
	for _, g := range f.groups {
		for _, item := range g.Items {
			filterableItems = append(filterableItems, item)
		}
	}

	items := []list.Item{}
	visitedGroups := map[int]bool{}

	// 使用匹配的项目重建组
	// 查找此项目属于哪个组
	for gi, g := range f.groups {
		addedCount := 0
		name := strings.ToLower(g.Title) + " "

		names := make([]string, len(filterableItems))
		for i, item := range filterableItems {
			ms := item.(*ModelItem)
			names[i] = fmt.Sprintf("%s%s", name, ms.Filter())
		}

		matches := fuzzy.Find(query, names)
		sort.SliceStable(matches, func(i, j int) bool {
			return matches[i].Score > matches[j].Score
		})

		for _, match := range matches {
			item := filterableItems[match.Index].(*ModelItem)
			idxs := []int{}
			for _, idx := range match.MatchedIndexes {
				// 调整移除提供者名称高亮
				if idx < len(name) {
					continue
				}
				idxs = append(idxs, idx-len(name))
			}

			match.MatchedIndexes = idxs
			if slices.Contains(g.Items, item) {
				if !visitedGroups[gi] {
					// 添加部分标题
					items = append(items, &g)
					visitedGroups[gi] = true
				}
				// 添加匹配的项目
				item.SetMatch(match)
				items = append(items, item)
				addedCount++
			}
		}
		if addedCount > 0 {
			// 在每个提供者部分后添加一个空格分隔符
			items = append(items, list.NewSpacerItem(1))
		}
	}

	return items
}

// Render 渲染可过滤列表。
func (f *ModelsList) Render() string {
	f.SetItems(f.VisibleItems()...)
	return f.List.Render()
}

type modelGroups []ModelGroup

func (m modelGroups) Len() int {
	n := 0
	for _, g := range m {
		n += len(g.Items)
	}
	return n
}

func (m modelGroups) String(i int) string {
	count := 0
	for _, g := range m {
		if i < count+len(g.Items) {
			return g.Items[i-count].Filter()
		}
		count += len(g.Items)
	}
	return ""
}
