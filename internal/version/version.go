package version

import "runtime/debug"

// Build-time parameters set via -ldflags

// Version 存储应用程序的版本号
// 默认值为 "devel"，会在构建时通过 -ldflags 覆盖
var Version = "devel"

// 初始化函数，用于从构建信息中获取版本号
// 当用户使用 `go install github.com/charmbracelet/crush@latest` 安装时
// 没有 -ldflags 参数，此时上面的版本号未设置
// 作为 workaround，我们使用 `go install` 时会设置的嵌入式构建版本
// （此版本号仅在 `go install` 时设置，`go build` 时不会设置）
func init() {
	// 读取构建信息
	info, ok := debug.ReadBuildInfo()
	if !ok {
		// 如果无法读取构建信息，直接返回
		return
	}

	// 获取主模块的版本号
	mainVersion := info.Main.Version

	// 如果版本号不为空且不是开发版本，则使用该版本号
	if mainVersion != "" && mainVersion != "(devel)" {
		Version = mainVersion
	}
}
