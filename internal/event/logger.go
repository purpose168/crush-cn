// Package event 提供事件处理相关的功能
package event

import (
	"fmt"
	"log/slog"

	"github.com/posthog/posthog-go"
)

// 确保 logger 类型实现了 posthog.Logger 接口
// 这是一个编译时的类型检查
var _ posthog.Logger = logger{}

// logger 是一个实现了 posthog.Logger 接口的日志记录器
// 它将 PostHog 的日志调用转发到标准库的 slog 包
type logger struct{}

// Debugf 记录调试级别的日志消息
// 参数:
//   - format: 格式化字符串
//   - args: 格式化参数
//
// 该方法将消息格式化后，通过 slog.Debug 输出
func (logger) Debugf(format string, args ...any) {
	slog.Debug(fmt.Sprintf(format, args...))
}

// Logf 记录信息级别的日志消息
// 参数:
//   - format: 格式化字符串
//   - args: 格式化参数
//
// 该方法将消息格式化后，通过 slog.Info 输出
func (logger) Logf(format string, args ...any) {
	slog.Info(fmt.Sprintf(format, args...))
}

// Warnf 记录警告级别的日志消息
// 参数:
//   - format: 格式化字符串
//   - args: 格式化参数
//
// 该方法将消息格式化后，通过 slog.Warn 输出
func (logger) Warnf(format string, args ...any) {
	slog.Warn(fmt.Sprintf(format, args...))
}

// Errorf 记录错误级别的日志消息
// 参数:
//   - format: 格式化字符串
//   - args: 格式化参数
//
// 该方法将消息格式化后，通过 slog.Error 输出
func (logger) Errorf(format string, args ...any) {
	slog.Error(fmt.Sprintf(format, args...))
}
