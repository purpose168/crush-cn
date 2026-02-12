package agent

import "errors"

var (
	// ErrRequestCancelled 请求被用户取消
	ErrRequestCancelled = errors.New("请求被用户取消")
	// ErrSessionBusy 会话当前正在处理另一个请求
	ErrSessionBusy      = errors.New("会话当前正在处理另一个请求")
	// ErrEmptyPrompt 提示词为空
	ErrEmptyPrompt      = errors.New("提示词为空")
	// ErrSessionMissing 会话ID缺失
	ErrSessionMissing   = errors.New("会话ID缺失")
)
