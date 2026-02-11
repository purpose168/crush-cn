package list

import (
	"github.com/sahilm/fuzzy"
)

// FilterableItem 是可以通过查询进行过滤的项目。
type FilterableItem interface {
	Item
	// Filter 返回用于过滤的值。
	Filter() string
}

// MatchSettable 是一个接口，用于可以设置其匹配索引
// 和匹配分数的项目。
type MatchSettable interface {
	SetMatch(fuzzy.Match)
}

// FilterableList 是一个列表，它接受可以通过可设置查询进行过滤的可过滤项目。
type FilterableList struct {
	*List
	items []FilterableItem
	query string
}

// NewFilterableList 创建一个新的可过滤列表。
func NewFilterableList(items ...FilterableItem) *FilterableList {
	f := &FilterableList{
		List:  NewList(),
		items: items,
	}
	f.RegisterRenderCallback(FocusedRenderCallback(f.List))
	f.SetItems(items...)
	return f
}

// SetItems 设置列表项目并更新过滤后的项目。
func (f *FilterableList) SetItems(items ...FilterableItem) {
	f.items = items
	fitems := make([]Item, len(items))
	for i, item := range items {
		fitems[i] = item
	}
	f.List.SetItems(fitems...)
}

// AppendItems 将项目追加到列表并更新过滤后的项目。
func (f *FilterableList) AppendItems(items ...FilterableItem) {
	f.items = append(f.items, items...)
	itms := make([]Item, len(f.items))
	for i, item := range f.items {
		itms[i] = item
	}
	f.List.SetItems(itms...)
}

// PrependItems 将项目前置到列表并更新过滤后的项目。
func (f *FilterableList) PrependItems(items ...FilterableItem) {
	f.items = append(items, f.items...)
	itms := make([]Item, len(f.items))
	for i, item := range f.items {
		itms[i] = item
	}
	f.List.SetItems(itms...)
}

// SetFilter 设置过滤查询并更新列表项目。
func (f *FilterableList) SetFilter(q string) {
	f.query = q
	f.List.SetItems(f.FilteredItems()...)
	f.ScrollToTop()
}

// FilterableItemsSource 是一个类型，它实现了[fuzzy.Source]用于过滤
// [FilterableItem]。
type FilterableItemsSource []FilterableItem

// Len 返回源的长度。
func (f FilterableItemsSource) Len() int {
	return len(f)
}

// String 返回索引i处项目的字符串表示。
func (f FilterableItemsSource) String(i int) string {
	return f[i].Filter()
}

// FilteredItems 返回过滤后的可见项目。
func (f *FilterableList) FilteredItems() []Item {
	if f.query == "" {
		items := make([]Item, len(f.items))
		for i, item := range f.items {
			if ms, ok := item.(MatchSettable); ok {
				ms.SetMatch(fuzzy.Match{})
				item = ms.(FilterableItem)
			}
			items[i] = item
		}
		return items
	}

	items := FilterableItemsSource(f.items)
	matches := fuzzy.FindFrom(f.query, items)
	matchedItems := []Item{}
	resultSize := len(matches)
	for i := range resultSize {
		match := matches[i]
		item := items[match.Index]
		if ms, ok := item.(MatchSettable); ok {
			ms.SetMatch(match)
			item = ms.(FilterableItem)
		}
		matchedItems = append(matchedItems, item)
	}

	return matchedItems
}

// Render 渲染可过滤列表。
func (f *FilterableList) Render() string {
	f.List.SetItems(f.FilteredItems()...)
	return f.List.Render()
}
