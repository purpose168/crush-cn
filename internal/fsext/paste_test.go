package fsext

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParsePastedFiles 测试粘贴文件路径解析功能
func TestParsePastedFiles(t *testing.T) {
	// 测试 Windows 终端格式
	t.Run("Windows终端", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected []string
		}{
			{
				name:     "单个路径",
				input:    `"C:\path\my-screenshot-one.png"`,
				expected: []string{`C:\path\my-screenshot-one.png`},
			},
			{
				name:     "多个路径无空格",
				input:    `"C:\path\my-screenshot-one.png" "C:\path\my-screenshot-two.png" "C:\path\my-screenshot-three.png"`,
				expected: []string{`C:\path\my-screenshot-one.png`, `C:\path\my-screenshot-two.png`, `C:\path\my-screenshot-three.png`},
			},
			{
				name:     "单个路径含空格",
				input:    `"C:\path\my screenshot one.png"`,
				expected: []string{`C:\path\my screenshot one.png`},
			},
			{
				name:     "多个路径含空格",
				input:    `"C:\path\my screenshot one.png" "C:\path\my screenshot two.png" "C:\path\my screenshot three.png"`,
				expected: []string{`C:\path\my screenshot one.png`, `C:\path\my screenshot two.png`, `C:\path\my screenshot three.png`},
			},
			{
				name:     "空字符串",
				input:    "",
				expected: nil,
			},
			{
				name:     "未闭合引号",
				input:    `"C:\path\file.png`,
				expected: nil,
			},
			{
				name:     "引号外有文本",
				input:    `"C:\path\file.png" some random text "C:\path\file2.png"`,
				expected: nil,
			},
			{
				name:     "路径间多个空格",
				input:    `"C:\path\file1.png"    "C:\path\file2.png"`,
				expected: []string{`C:\path\file1.png`, `C:\path\file2.png`},
			},
			{
				name:     "仅空白字符",
				input:    "   ",
				expected: nil,
			},
			{
				name:     "连续引用部分",
				input:    `"C:\path1""C:\path2"`,
				expected: []string{`C:\path1`, `C:\path2`},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := windowsTerminalParsePastedFiles(tt.input)
				require.Equal(t, tt.expected, result)
			})
		}
	})

	// 测试 Unix 系统格式
	t.Run("Unix系统", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected []string
		}{
			{
				name:     "单个路径",
				input:    `/path/my-screenshot.png`,
				expected: []string{"/path/my-screenshot.png"},
			},
			{
				name:     "多个路径无空格",
				input:    `/path/screenshot-one.png /path/screenshot-two.png /path/screenshot-three.png`,
				expected: []string{"/path/screenshot-one.png", "/path/screenshot-two.png", "/path/screenshot-three.png"},
			},
			{
				name:     "单个路径含空格",
				input:    `/path/my\ screenshot\ one.png`,
				expected: []string{"/path/my screenshot one.png"},
			},
			{
				name:     "多个路径含空格",
				input:    `/path/my\ screenshot\ one.png /path/my\ screenshot\ two.png /path/my\ screenshot\ three.png`,
				expected: []string{"/path/my screenshot one.png", "/path/my screenshot two.png", "/path/my screenshot three.png"},
			},
			{
				name:     "空字符串",
				input:    "",
				expected: nil,
			},
			{
				name:     "双反斜杠转义",
				input:    `/path/my\\file.png`,
				expected: []string{"/path/my\\file.png"},
			},
			{
				name:     "尾部反斜杠",
				input:    `/path/file\`,
				expected: []string{`/path/file\`},
			},
			{
				name:     "多个连续转义空格",
				input:    `/path/file\ \ with\ \ many\ \ spaces.png`,
				expected: []string{"/path/file  with  many  spaces.png"},
			},
			{
				name:     "多个未转义空格",
				input:    `/path/file1.png   /path/file2.png`,
				expected: []string{"/path/file1.png", "/path/file2.png"},
			},
			{
				name:     "仅空白字符",
				input:    "   ",
				expected: nil,
			},
			{
				name:     "制表符",
				input:    "/path/file1.png\t/path/file2.png",
				expected: []string{"/path/file1.png\t/path/file2.png"},
			},
			{
				name:     "输入中含换行符",
				input:    "/path/file1.png\n/path/file2.png",
				expected: []string{"/path/file1.png\n/path/file2.png"},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := unixParsePastedFiles(tt.input)
				require.Equal(t, tt.expected, result)
			})
		}
	})
}
