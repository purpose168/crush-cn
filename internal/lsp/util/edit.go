package util

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/x/powernap/pkg/lsp/protocol"
)

// applyTextEdits 将文本编辑应用到指定的文档URI
// 参数:
//   - uri: 文档URI
//   - edits: 要应用的文本编辑列表
// 返回值: 应用编辑时发生的错误
func applyTextEdits(uri protocol.DocumentURI, edits []protocol.TextEdit) error {
	path, err := uri.Path()
	if err != nil {
		return fmt.Errorf("无效的URI: %w", err)
	}

	// 读取文件内容
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	// 检测换行符风格
	var lineEnding string
	if bytes.Contains(content, []byte("\r\n")) {
		lineEnding = "\r\n"
	} else {
		lineEnding = "\n"
	}

	// 跟踪文件是否以换行符结尾
	endsWithNewline := len(content) > 0 && bytes.HasSuffix(content, []byte(lineEnding))

	// 按换行符分割成行（不包含换行符）
	lines := strings.Split(string(content), lineEnding)

	// 检查重叠的编辑
	for i, edit1 := range edits {
		for j := i + 1; j < len(edits); j++ {
			if rangesOverlap(edit1.Range, edits[j].Range) {
				return fmt.Errorf("检测到编辑%d和%d之间有重叠", i, j)
			}
		}
	}

	// 按逆序排序编辑（从后往前应用）
	sortedEdits := make([]protocol.TextEdit, len(edits))
	copy(sortedEdits, edits)
	sort.Slice(sortedEdits, func(i, j int) bool {
		if sortedEdits[i].Range.Start.Line != sortedEdits[j].Range.Start.Line {
			return sortedEdits[i].Range.Start.Line > sortedEdits[j].Range.Start.Line
		}
		return sortedEdits[i].Range.Start.Character > sortedEdits[j].Range.Start.Character
	})

	// 应用每个编辑
	for _, edit := range sortedEdits {
		newLines, err := applyTextEdit(lines, edit)
		if err != nil {
			return fmt.Errorf("应用编辑失败: %w", err)
		}
		lines = newLines
	}

	// 用正确的换行符连接行
	var newContent strings.Builder
	for i, line := range lines {
		if i > 0 {
			newContent.WriteString(lineEnding)
		}
		newContent.WriteString(line)
	}

	// 仅当原始文件有换行符且我们尚未添加时才添加换行符
	if endsWithNewline && !strings.HasSuffix(newContent.String(), lineEnding) {
		newContent.WriteString(lineEnding)
	}

	if err := os.WriteFile(path, []byte(newContent.String()), 0o644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// applyTextEdit 将单个文本编辑应用到行列表
// 参数:
//   - lines: 当前行列表
//   - edit: 要应用的文本编辑
// 返回值: 修改后的行列表和可能的错误
func applyTextEdit(lines []string, edit protocol.TextEdit) ([]string, error) {
	startLine := int(edit.Range.Start.Line)
	endLine := int(edit.Range.End.Line)
	startChar := int(edit.Range.Start.Character)
	endChar := int(edit.Range.End.Character)

	// 验证位置有效性
	if startLine < 0 || startLine >= len(lines) {
		return nil, fmt.Errorf("无效的起始行: %d", startLine)
	}
	if endLine < 0 || endLine >= len(lines) {
		endLine = len(lines) - 1
	}

	// 创建结果切片，预留初始容量
	result := make([]string, 0, len(lines))

	// 复制编辑之前的行
	result = append(result, lines[:startLine]...)

	// 获取起始行的前缀
	startLineContent := lines[startLine]
	if startChar < 0 || startChar > len(startLineContent) {
		startChar = len(startLineContent)
	}
	prefix := startLineContent[:startChar]

	// 获取结束行的后缀
	endLineContent := lines[endLine]
	if endChar < 0 || endChar > len(endLineContent) {
		endChar = len(endLineContent)
	}
	suffix := endLineContent[endChar:]

	// 处理编辑
	if edit.NewText == "" {
		// 删除操作
		if prefix+suffix != "" {
			result = append(result, prefix+suffix)
		}
	} else {
		// 将新文本分割成行，注意不要添加额外的换行符
		newLines := strings.Split(edit.NewText, "\n")

		if len(newLines) == 1 {
			// 单行变更
			result = append(result, prefix+newLines[0]+suffix)
		} else {
			// 多行变更
			result = append(result, prefix+newLines[0])
			result = append(result, newLines[1:len(newLines)-1]...)
			result = append(result, newLines[len(newLines)-1]+suffix)
		}
	}

	// 添加剩余的行
	if endLine+1 < len(lines) {
		result = append(result, lines[endLine+1:]...)
	}

	return result, nil
}

// applyDocumentChange 应用文档变更（创建/重命名/删除操作）
// 参数:
//   - change: 文档变更对象
// 返回值: 应用变更时发生的错误
func applyDocumentChange(change protocol.DocumentChange) error {
	// 处理创建文件操作
	if change.CreateFile != nil {
		path, err := change.CreateFile.URI.Path()
		if err != nil {
			return fmt.Errorf("无效的URI: %w", err)
		}

		if change.CreateFile.Options != nil {
			if change.CreateFile.Options.Overwrite {
				// 继续覆盖操作
			} else if change.CreateFile.Options.IgnoreIfExists {
				if _, err := os.Stat(path); err == nil {
					return nil  // 文件存在，忽略创建
				}
			}
		}
		if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
			return fmt.Errorf("创建文件失败: %w", err)
		}
	}

	// 处理删除文件操作
	if change.DeleteFile != nil {
		path, err := change.DeleteFile.URI.Path()
		if err != nil {
			return fmt.Errorf("无效的URI: %w", err)
		}

		if change.DeleteFile.Options != nil && change.DeleteFile.Options.Recursive {
			if err := os.RemoveAll(path); err != nil {
				return fmt.Errorf("递归删除目录失败: %w", err)
			}
		} else {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("删除文件失败: %w", err)
			}
		}
	}

	// 处理重命名文件操作
	if change.RenameFile != nil {
		var newPath, oldPath string
		var err error

		oldPath, err = change.RenameFile.OldURI.Path()
		if err != nil {
			return err
		}

		newPath, err = change.RenameFile.NewURI.Path()
		if err != nil {
			return err
		}

		if change.RenameFile.Options != nil {
			if !change.RenameFile.Options.Overwrite {
				if _, err := os.Stat(newPath); err == nil {
					return fmt.Errorf("目标文件已存在且不允许覆盖: %s", newPath)
				}
			}
		}
		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("重命名文件失败: %w", err)
		}
	}

	// 处理文本文档编辑操作
	if change.TextDocumentEdit != nil {
		textEdits := make([]protocol.TextEdit, len(change.TextDocumentEdit.Edits))
		for i, edit := range change.TextDocumentEdit.Edits {
			var err error
			textEdits[i], err = edit.AsTextEdit()
			if err != nil {
				return fmt.Errorf("无效的编辑类型: %w", err)
			}
		}
		return applyTextEdits(change.TextDocumentEdit.TextDocument.URI, textEdits)
	}

	return nil
}

// ApplyWorkspaceEdit 将给定的工作区编辑应用到文件系统
// 参数:
//   - edit: 工作区编辑对象
// 返回值: 应用编辑时发生的错误
func ApplyWorkspaceEdit(edit protocol.WorkspaceEdit) error {
	// 处理Changes字段（按URI映射的文本编辑）
	for uri, textEdits := range edit.Changes {
		if err := applyTextEdits(uri, textEdits); err != nil {
			return fmt.Errorf("应用文本编辑失败: %w", err)
		}
	}

	// 处理DocumentChanges字段（文档变更列表）
	for _, change := range edit.DocumentChanges {
		if err := applyDocumentChange(change); err != nil {
			return fmt.Errorf("应用文档变更失败: %w", err)
		}
	}

	return nil
}

// rangesOverlap 检查两个范围是否重叠
// 参数:
//   - r1: 第一个范围
//   - r2: 第二个范围
// 返回值: 如果范围重叠返回true，否则返回false
func rangesOverlap(r1, r2 protocol.Range) bool {
	if r1.Start.Line > r2.End.Line || r2.Start.Line > r1.End.Line {
		return false
	}
	if r1.Start.Line == r2.End.Line && r1.Start.Character > r2.End.Character {
		return false
	}
	if r2.Start.Line == r1.End.Line && r2.Start.Character > r1.End.Character {
		return false
	}
	return true
}
