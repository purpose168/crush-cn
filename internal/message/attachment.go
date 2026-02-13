package message

import (
	"slices"
	"strings"
)

// Attachment 表示消息附件的结构体
// 用于存储附件的文件信息、MIME类型和内容数据
type Attachment struct {
	FilePath string // 文件路径，附件在文件系统中的完整路径
	FileName string // 文件名，附件的原始文件名
	MimeType string // MIME类型，用于标识文件的媒体类型（如 text/plain, image/png）
	Content  []byte // 文件内容，附件的二进制数据
}

// IsText 判断附件是否为文本类型
// 返回值：如果附件的 MIME 类型以 "text/" 开头则返回 true，否则返回 false
func (a Attachment) IsText() bool  { return strings.HasPrefix(a.MimeType, "text/") }

// IsImage 判断附件是否为图片类型
// 返回值：如果附件的 MIME 类型以 "image/" 开头则返回 true，否则返回 false
func (a Attachment) IsImage() bool { return strings.HasPrefix(a.MimeType, "image/") }

// ContainsTextAttachment 检查附件列表中是否包含文本类型的附件
// 参数：
//   - attachments: 附件切片，需要检查的附件列表
// 返回值：
//   - bool: 如果附件列表中至少有一个文本类型的附件则返回 true，否则返回 false
func ContainsTextAttachment(attachments []Attachment) bool {
	return slices.ContainsFunc(attachments, func(a Attachment) bool {
		return a.IsText()
	})
}
