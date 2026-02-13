package message

import (
	"fmt"
	"strings"
	"testing"
)

// makeTestAttachments 创建指定数量的测试附件
// 参数:
//   - n: 要创建的附件数量
//   - contentSize: 每个附件的内容大小（字节数）
// 返回:
//   - []Attachment: 生成的附件切片
func makeTestAttachments(n int, contentSize int) []Attachment {
	attachments := make([]Attachment, n)
	// 创建指定大小的重复内容用于测试
	content := []byte(strings.Repeat("x", contentSize))
	for i := range n {
		attachments[i] = Attachment{
			FilePath: fmt.Sprintf("/path/to/file%d.txt", i),
			MimeType: "text/plain",
			Content:  content,
		}
	}
	return attachments
}

// BenchmarkPromptWithTextAttachments 基准测试：测试带文本附件的提示词处理性能
// 该基准测试评估不同文件数量和内容大小对 PromptWithTextAttachments 函数性能的影响
func BenchmarkPromptWithTextAttachments(b *testing.B) {
	// 定义测试用例：包含不同的文件数量和内容大小组合
	cases := []struct {
		name        string // 测试用例名称
		numFiles    int    // 文件数量
		contentSize int    // 内容大小（字节）
	}{
		{"1file_100bytes", 1, 100},         // 1个文件，100字节
		{"5files_1KB", 5, 1024},            // 5个文件，每个1KB
		{"10files_10KB", 10, 10 * 1024},    // 10个文件，每个10KB
		{"20files_50KB", 20, 50 * 1024},    // 20个文件，每个50KB
	}

	for _, tc := range cases {
		// 根据测试用例参数创建测试附件
		attachments := makeTestAttachments(tc.numFiles, tc.contentSize)
		// 提示词：处理这些文件
		prompt := "处理这些文件"

		b.Run(tc.name, func(b *testing.B) {
			// 启用内存分配统计
			b.ReportAllocs()
			// 执行基准测试循环
			for range b.N {
				_ = PromptWithTextAttachments(prompt, attachments)
			}
		})
	}
}
