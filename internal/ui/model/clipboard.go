package model

import "errors"

// clipboardFormat 表示剪贴板格式类型
type clipboardFormat int

const (
	// clipboardFormatText 文本格式
	clipboardFormatText clipboardFormat = iota
	// clipboardFormatImage 图片格式
	clipboardFormatImage
)

var (
	// errClipboardPlatformUnsupported 该平台不支持剪贴板操作错误
	errClipboardPlatformUnsupported = errors.New("clipboard operations are not supported on this platform")
	// errClipboardUnknownFormat 未知剪贴板格式错误
	errClipboardUnknownFormat = errors.New("unknown clipboard format")
)
