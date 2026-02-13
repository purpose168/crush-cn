package filepathext

import (
	"path/filepath"
	"runtime"
	"strings"
)

// SmartJoin 智能连接两个路径，如果第二个路径是绝对路径，则直接返回第二个路径。
// 该函数会自动判断第二个路径是否为绝对路径，如果是绝对路径则忽略第一个路径，
// 否则使用 filepath.Join 将两个路径连接起来。
func SmartJoin(one, two string) string {
	if SmartIsAbs(two) {
		return two
	}
	return filepath.Join(one, two)
}

// SmartIsAbs 智能检查路径是否为绝对路径，同时考虑操作系统特定的路径格式和 Unix 风格路径。
// 在 Windows 系统上，除了检查系统原生的绝对路径外，还会检查以 "/" 开头的 Unix 风格路径。
// 在其他操作系统上，直接使用 filepath.IsAbs 进行检查。
func SmartIsAbs(path string) bool {
	switch runtime.GOOS {
	case "windows":
		return filepath.IsAbs(path) || strings.HasPrefix(filepath.ToSlash(path), "/")
	default:
		return filepath.IsAbs(path)
	}
}
