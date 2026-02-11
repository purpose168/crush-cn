package common

import (
	tea "charm.land/bubbletea/v2"
)

// Model 表示 UI 组件的通用接口。
type Model[T any] interface {
	Update(msg tea.Msg) (T, tea.Cmd)
	View() string
}
