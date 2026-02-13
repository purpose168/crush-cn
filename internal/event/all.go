// Package event 提供应用程序事件跟踪和记录功能
// 该文件定义了应用程序生命周期和会话管理相关的事件记录函数
package event

import (
	"time"
)

// appStartTime 记录应用程序启动时间
var appStartTime time.Time

// AppInitialized 记录应用程序初始化完成事件
// 在应用程序启动完成时调用，用于标记应用开始运行的时间点
func AppInitialized() {
	appStartTime = time.Now()
	send("应用已初始化")
}

// AppExited 记录应用程序退出事件
// 在应用程序退出时调用，计算并记录应用运行时长
func AppExited() {
	duration := time.Since(appStartTime).Truncate(time.Second)
	send(
		"应用已退出",
		"应用运行时长（可读格式）", duration.String(),
		"应用运行时长（秒）", int64(duration.Seconds()),
	)
	Flush()
}

// SessionCreated 记录会话创建事件
// 当用户创建新的会话时调用
func SessionCreated() {
	send("会话已创建")
}

// SessionDeleted 记录会话删除事件
// 当用户删除会话时调用
func SessionDeleted() {
	send("会话已删除")
}

// SessionSwitched 记录会话切换事件
// 当用户在不同会话之间切换时调用
func SessionSwitched() {
	send("会话已切换")
}

// FilePickerOpened 记录文件选择器打开事件
// 当文件选择对话框打开时调用
func FilePickerOpened() {
	send("文件选择器已打开")
}

// PromptSent 记录提示消息发送事件
// 当用户向AI发送提示消息时调用
// props: 附加的事件属性，以键值对形式传入
func PromptSent(props ...any) {
	send(
		"提示已发送",
		props...,
	)
}

// PromptResponded 记录提示响应事件
// 当AI对用户提示做出响应时调用
// props: 附加的事件属性，以键值对形式传入
func PromptResponded(props ...any) {
	send(
		"提示已响应",
		props...,
	)
}

// TokensUsed 记录令牌使用事件
// 用于跟踪AI模型调用时的令牌消耗情况
// props: 附加的事件属性，以键值对形式传入
func TokensUsed(props ...any) {
	send(
		"令牌已使用",
		props...,
	)
}
