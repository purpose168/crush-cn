//go:build !(darwin || linux || windows) || arm || 386 || ios || android

// 该文件在以下平台上构建：非darwin、linux或windows系统，或者arm、386、ios或android系统

package model

// readClipboard 读取剪贴板内容
// 参数：clipboardFormat - 剪贴板格式
// 返回：剪贴板内容的字节数组和可能的错误
// 该平台不支持剪贴板操作，返回errClipboardPlatformUnsupported错误
func readClipboard(clipboardFormat) ([]byte, error) {
	return nil, errClipboardPlatformUnsupported
}
