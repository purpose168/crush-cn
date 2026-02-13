package log

import (
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/purpose168/crush-cn/internal/event"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	initOnce    sync.Once      // 确保初始化只执行一次
	initialized atomic.Bool    // 标记日志系统是否已初始化
)

// Setup 初始化日志系统
// 参数:
//   - logFile: 日志文件路径
//   - debug: 是否启用调试模式（调试模式会输出更详细的日志）
func Setup(logFile string, debug bool) {
	initOnce.Do(func() {
		// 创建日志轮转器，用于管理日志文件的大小和备份数量
		logRotator := &lumberjack.Logger{
			Filename:   logFile,  // 日志文件名
			MaxSize:    10,       // 单个日志文件最大大小（MB）
			MaxBackups: 0,        // 保留的旧日志文件最大数量，0表示不限制
			MaxAge:     30,       // 保留旧日志文件的最大天数
			Compress:   false,    // 是否压缩旧日志文件
		}

		// 根据调试模式设置日志级别
		level := slog.LevelInfo
		if debug {
			level = slog.LevelDebug
		}

		// 创建JSON格式的日志处理器
		logger := slog.NewJSONHandler(logRotator, &slog.HandlerOptions{
			Level:     level,      // 日志级别
			AddSource: true,       // 添加源代码位置信息
		})

		// 设置为默认日志记录器
		slog.SetDefault(slog.New(logger))
		initialized.Store(true)
	})
}

// Initialized 检查日志系统是否已初始化
// 返回值: 如果日志系统已初始化返回true，否则返回false
func Initialized() bool {
	return initialized.Load()
}

// RecoverPanic 恢复panic并记录错误信息
// 该函数应在defer语句中调用，用于捕获并处理panic
// 参数:
//   - name: panic发生位置的标识名称
//   - cleanup: 清理函数，在panic发生后执行（可选）
func RecoverPanic(name string, cleanup func()) {
	if r := recover(); r != nil {
		// 记录panic错误事件
		event.Error(r, "panic", true, "name", name)

		// 创建带有时间戳的panic日志文件
		timestamp := time.Now().Format("20060102-150405")
		filename := fmt.Sprintf("crush-panic-%s-%s.log", name, timestamp)

		file, err := os.Create(filename)
		if err == nil {
			defer file.Close()

			// 写入panic信息和堆栈跟踪
			fmt.Fprintf(file, "Panic in %s: %v\n\n", name, r)
			fmt.Fprintf(file, "Time: %s\n\n", time.Now().Format(time.RFC3339))
			fmt.Fprintf(file, "Stack Trace:\n%s\n", debug.Stack())

			// 如果提供了清理函数，则执行清理操作
			if cleanup != nil {
				cleanup()
			}
		}
	}
}
