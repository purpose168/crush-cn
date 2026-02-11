// Package util 提供 UI 消息处理的工具函数。
package util

import (
	"context"
	"errors"
	"log/slog"
	"os/exec"
	"time"

	tea "charm.land/bubbletea/v2"
	"mvdan.cc/sh/v3/shell"
)

// Cursor 接口定义了获取光标信息的方法
type Cursor interface {
	Cursor() *tea.Cursor
}

// CmdHandler 创建一个返回指定消息的命令
// 参数: msg - 要返回的消息
// 返回值: 一个执行时返回该消息的命令
func CmdHandler(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return msg
	}
}

// ReportError 报告错误并创建错误消息命令
// 参数: err - 要报告的错误
// 返回值: 一个包含错误信息的命令
func ReportError(err error) tea.Cmd {
	slog.Error("报告错误", "error", err)
	return CmdHandler(NewErrorMsg(err))
}

// InfoType 定义信息消息的类型
type InfoType int

const (
	InfoTypeInfo    InfoType = iota // 普通信息
	InfoTypeSuccess                 // 成功信息
	InfoTypeWarn                    // 警告信息
	InfoTypeError                   // 错误信息
	InfoTypeUpdate                  // 更新信息
)

// NewInfoMsg 创建新的普通信息消息
// 参数: info - 信息内容
// 返回值: 包含指定信息的 InfoMsg
func NewInfoMsg(info string) InfoMsg {
	return InfoMsg{
		Type: InfoTypeInfo,
		Msg:  info,
	}
}

// NewWarnMsg 创建新的警告信息消息
// 参数: warn - 警告内容
// 返回值: 包含指定警告的 InfoMsg
func NewWarnMsg(warn string) InfoMsg {
	return InfoMsg{
		Type: InfoTypeWarn,
		Msg:  warn,
	}
}

// NewErrorMsg 创建新的错误信息消息
// 参数: err - 错误对象
// 返回值: 包含错误信息的 InfoMsg
func NewErrorMsg(err error) InfoMsg {
	return InfoMsg{
		Type: InfoTypeError,
		Msg:  err.Error(),
	}
}

// ReportInfo 报告信息并创建信息消息命令
// 参数: info - 信息内容
// 返回值: 一个包含信息的命令
func ReportInfo(info string) tea.Cmd {
	return CmdHandler(NewInfoMsg(info))
}

// ReportWarn 报告警告并创建警告消息命令
// 参数: warn - 警告内容
// 返回值: 一个包含警告的命令
func ReportWarn(warn string) tea.Cmd {
	return CmdHandler(NewWarnMsg(warn))
}

// InfoMsg 定义信息消息结构
type (
	InfoMsg struct {
		Type InfoType      // 消息类型
		Msg  string        // 消息内容
		TTL  time.Duration // 消息存活时间
	}
	ClearStatusMsg struct{} // 清除状态消息
)

// IsEmpty 检查 [InfoMsg] 是否为空
func (m InfoMsg) IsEmpty() bool {
	var zero InfoMsg
	return m == zero
}

// ExecShell 解析 shell 命令字符串并使用 exec.Command 执行它
// 使用 shell.Fields 正确处理 shell 语法，如引号和参数，同时为终端编辑器保留 TTY 处理
// 参数:
//
//	ctx - 上下文，用于控制命令执行
//	cmdStr - 要执行的 shell 命令字符串
//	callback - 命令执行的回调函数
//
// 返回值: 执行命令的 tea.Cmd
func ExecShell(ctx context.Context, cmdStr string, callback tea.ExecCallback) tea.Cmd {
	fields, err := shell.Fields(cmdStr, nil)
	if err != nil {
		return ReportError(err)
	}
	if len(fields) == 0 {
		return ReportError(errors.New("空命令"))
	}

	cmd := exec.CommandContext(ctx, fields[0], fields[1:]...)
	return tea.ExecProcess(cmd, callback)
}
