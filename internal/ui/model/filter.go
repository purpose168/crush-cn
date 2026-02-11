package model

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// lastMouseEvent 记录上一次鼠标事件的时间
var lastMouseEvent time.Time

// MouseEventFilter 过滤鼠标事件，防止触摸板发送过多请求
func MouseEventFilter(m tea.Model, msg tea.Msg) tea.Msg {
	switch msg.(type) {
	case tea.MouseWheelMsg, tea.MouseMotionMsg:
		now := time.Now()
		// trackpad is sending too many requests 触摸板发送的请求过多
		if now.Sub(lastMouseEvent) < 15*time.Millisecond {
			return nil
		}
		lastMouseEvent = now
	}
	return msg
}
