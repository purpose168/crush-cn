// Package stringext 提供字符串处理相关的扩展功能
package stringext

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Capitalize 将给定文本的首字母大写
// 参数：
//   - text: 需要处理的文本字符串
//
// 返回值：
//   - 返回首字母大写后的文本
//
// 该函数使用 golang.org/x/text/cases 包提供的 Title 功能，
// 并采用英语语言规则和紧凑模式进行文本转换
func Capitalize(text string) string {
	return cases.Title(language.English, cases.Compact).String(text)
}

// NormalizeSpace 规范化给定内容字符串中的空白字符
// 参数：
//   - content: 需要规范化的内容字符串
//
// 返回值：
//   - 返回规范化后的字符串
//
// 该函数执行以下规范化操作：
//  1. 将 Windows 风格的换行符（\r\n）替换为 Unix 风格的换行符（\n）
//  2. 将制表符（\t）转换为四个空格
//  3. 去除字符串首尾的空白字符
func NormalizeSpace(content string) string {
	// 将 Windows 换行符替换为 Unix 换行符
	content = strings.ReplaceAll(content, "\r\n", "\n")
	// 将制表符转换为四个空格
	content = strings.ReplaceAll(content, "\t", "    ")
	// 去除首尾空白字符
	content = strings.TrimSpace(content)
	return content
}
