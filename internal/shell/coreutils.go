package shell

import (
	"os"
	"runtime"
	"strconv"
)

// useGoCoreUtils 标志变量,用于控制是否使用Go语言实现的核心工具函数
// 当设置为true时,使用Go原生实现;当设置为false时,使用系统命令
var useGoCoreUtils bool

func init() {
	// 如果环境变量 CRUSH_CORE_UTILS 被设置为 true 或 false,则遵循该设置
	// 默认情况下,仅在 Windows 系统上启用Go核心工具实现
	if v, err := strconv.ParseBool(os.Getenv("CRUSH_CORE_UTILS")); err == nil {
		useGoCoreUtils = v
	} else {
		useGoCoreUtils = runtime.GOOS == "windows"
	}
}
