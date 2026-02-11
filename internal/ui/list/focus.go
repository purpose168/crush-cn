package list

// FocusedRenderCallback 是一个辅助函数，它返回一个渲染回调，
// 该回调在渲染期间将项目标记为已聚焦。
func FocusedRenderCallback(list *List) RenderCallback {
	return func(idx, selectedIdx int, item Item) Item {
		if focusable, ok := item.(Focusable); ok {
			focusable.SetFocused(list.Focused() && idx == selectedIdx)
			return focusable.(Item)
		}
		return item
	}
}
