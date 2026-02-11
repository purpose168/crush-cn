//go:build (linux || darwin || windows) && !arm && !386 && !ios && !android

// 该文件在以下平台上构建：linux、darwin或windows系统，且不是arm、386、ios或android系统

package model

import "github.com/aymanbagabas/go-nativeclipboard"

// readClipboard 读取剪贴板内容
// 参数：f - 剪贴板格式
// 返回：剪贴板内容的字节数组和可能的错误
// 支持的格式：clipboardFormatText（文本格式）和clipboardFormatImage（图片格式）
// 如果格式未知，返回errClipboardUnknownFormat错误
func readClipboard(f clipboardFormat) ([]byte, error) {
	switch f {
	case clipboardFormatText:
		return nativeclipboard.Text.Read()
	case clipboardFormatImage:
		return nativeclipboard.Image.Read()
	}
	return nil, errClipboardUnknownFormat
}
