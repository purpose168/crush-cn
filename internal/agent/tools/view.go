package tools

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"charm.land/fantasy"
	"github.com/purpose168/crush-cn/internal/filepathext"
	"github.com/purpose168/crush-cn/internal/filetracker"
	"github.com/purpose168/crush-cn/internal/lsp"
	"github.com/purpose168/crush-cn/internal/permission"
)

//go:embed view.md
var viewDescription []byte

type ViewParams struct {
	FilePath string `json:"file_path" description:"要读取的文件路径"`
	Offset   int    `json:"offset,omitempty" description:"开始读取的行号（从 0 开始）"`
	Limit    int    `json:"limit,omitempty" description:"要读取的行数（默认为 2000）"`
}

type ViewPermissionsParams struct {
	FilePath string `json:"file_path"`
	Offset   int    `json:"offset"`
	Limit    int    `json:"limit"`
}

type ViewResponseMetadata struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

const (
	ViewToolName     = "view"
	MaxReadSize      = 5 * 1024 * 1024 // 5MB
	DefaultReadLimit = 2000
	MaxLineLength    = 2000
)

func NewViewTool(
	lspManager *lsp.Manager,
	permissions permission.Service,
	filetracker filetracker.Service,
	workingDir string,
	skillsPaths ...string,
) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		ViewToolName,
		string(viewDescription),
		func(ctx context.Context, params ViewParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.FilePath == "" {
				return fantasy.NewTextErrorResponse("需要提供 file_path"), nil
			}

			// 处理相对路径
			filePath := filepathext.SmartJoin(workingDir, params.FilePath)

			// 检查文件是否在工作目录外，如需请求权限
			absWorkingDir, err := filepath.Abs(workingDir)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("解析工作目录错误: %w", err)
			}

			absFilePath, err := filepath.Abs(filePath)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("解析文件路径错误: %w", err)
			}

			relPath, err := filepath.Rel(absWorkingDir, absFilePath)
			isOutsideWorkDir := err != nil || strings.HasPrefix(relPath, "..")
			isSkillFile := isInSkillsPath(absFilePath, skillsPaths)

			sessionID := GetSessionFromContext(ctx)
			if sessionID == "" {
				return fantasy.ToolResponse{}, fmt.Errorf("访问工作目录外的文件需要会话 ID")
			}

			// 为工作目录外的文件请求权限，除非是技能文件
			if isOutsideWorkDir && !isSkillFile {
				granted, err := permissions.Request(ctx,
					permission.CreatePermissionRequest{
						SessionID:   sessionID,
						Path:        absFilePath,
						ToolCallID:  call.ID,
						ToolName:    ViewToolName,
						Action:      "read",
						Description: fmt.Sprintf("读取工作目录外的文件: %s", absFilePath),
						Params:      ViewPermissionsParams(params),
					},
				)
				if err != nil {
					return fantasy.ToolResponse{}, err
				}
				if !granted {
					return fantasy.ToolResponse{}, permission.ErrorPermissionDenied
				}
			}

			// 检查文件是否存在
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				if os.IsNotExist(err) {
					// 尝试提供名称相似的文件建议
					dir := filepath.Dir(filePath)
					base := filepath.Base(filePath)

					dirEntries, dirErr := os.ReadDir(dir)
					if dirErr == nil {
						var suggestions []string
						for _, entry := range dirEntries {
							if strings.Contains(strings.ToLower(entry.Name()), strings.ToLower(base)) ||
								strings.Contains(strings.ToLower(base), strings.ToLower(entry.Name())) {
								suggestions = append(suggestions, filepath.Join(dir, entry.Name()))
								if len(suggestions) >= 3 {
									break
								}
							}
						}

						if len(suggestions) > 0 {
							return fantasy.NewTextErrorResponse(fmt.Sprintf("文件未找到: %s\n\n您是指以下文件之一吗？\n%s",
								filePath, strings.Join(suggestions, "\n"))), nil
						}
					}

					return fantasy.NewTextErrorResponse(fmt.Sprintf("文件未找到: %s", filePath)), nil
				}
				return fantasy.ToolResponse{}, fmt.Errorf("访问文件错误: %w", err)
			}

			// 检查是否为目录
			if fileInfo.IsDir() {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("路径是目录，不是文件: %s", filePath)), nil
			}

			// 根据规范，我们不应该限制技能文件的读取
			if !isSkillFile && fileInfo.Size() > MaxReadSize {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("文件太大 (%d 字节)。最大大小为 %d 字节",
					fileInfo.Size(), MaxReadSize)), nil
			}

			// 如果未提供限制，则设置默认限制（SKILL.md 文件无限制）
			if params.Limit <= 0 {
				if isSkillFile {
					params.Limit = 1000000 // 技能文件实际上无限制
				} else {
					params.Limit = DefaultReadLimit
				}
			}

			isSupportedImage, mimeType := getImageMimeType(filePath)
			if isSupportedImage {
				if !GetSupportsImagesFromContext(ctx) {
					modelName := GetModelNameFromContext(ctx)
					return fantasy.NewTextErrorResponse(fmt.Sprintf("此模型 (%s) 不支持图像数据。", modelName)), nil
				}

				imageData, err := os.ReadFile(filePath)
				if err != nil {
					return fantasy.ToolResponse{}, fmt.Errorf("读取图像文件错误: %w", err)
				}

				encoded := base64.StdEncoding.EncodeToString(imageData)
				return fantasy.NewImageResponse([]byte(encoded), mimeType), nil
			}

			// 读取文件内容
			content, lineCount, err := readTextFile(filePath, params.Offset, params.Limit)
			isValidUt8 := utf8.ValidString(content)
			if !isValidUt8 {
				return fantasy.NewTextErrorResponse("文件内容不是有效的 UTF-8"), nil
			}
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("读取文件错误: %w", err)
			}

			notifyLSPs(ctx, lspManager, filePath)
			output := "<file>\n"
			// 格式化输出，添加行号
			output += addLineNumbers(content, params.Offset+1)

			// 如果内容被截断，添加说明
			if lineCount > params.Offset+len(strings.Split(content, "\n")) {
				output += fmt.Sprintf("\n\n(文件还有更多行。使用 'offset' 参数读取第 %d 行之后的内容)",
					params.Offset+len(strings.Split(content, "\n")))
			}
			output += "\n</file>\n"
			output += getDiagnostics(filePath, lspManager)
			filetracker.RecordRead(ctx, sessionID, filePath)
			return fantasy.WithResponseMetadata(
				fantasy.NewTextResponse(output),
				ViewResponseMetadata{
					FilePath: filePath,
					Content:  content,
				},
			), nil
		})
}

func addLineNumbers(content string, startLine int) string {
	if content == "" {
		return ""
	}

	lines := strings.Split(content, "\n")

	var result []string
	for i, line := range lines {
		line = strings.TrimSuffix(line, "\r")

		lineNum := i + startLine
		numStr := fmt.Sprintf("%d", lineNum)

		if len(numStr) >= 6 {
			result = append(result, fmt.Sprintf("%s|%s", numStr, line))
		} else {
			paddedNum := fmt.Sprintf("%6s", numStr)
			result = append(result, fmt.Sprintf("%s|%s", paddedNum, line))
		}
	}

	return strings.Join(result, "\n")
}

func readTextFile(filePath string, offset, limit int) (string, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	lineCount := 0

	scanner := NewLineScanner(file)
	if offset > 0 {
		for lineCount < offset && scanner.Scan() {
			lineCount++
		}
		if err = scanner.Err(); err != nil {
			return "", 0, err
		}
	}

	if offset == 0 {
		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			return "", 0, err
		}
	}

	// 预分配具有预期容量的切片
	lines := make([]string, 0, limit)
	lineCount = offset

	for scanner.Scan() && len(lines) < limit {
		lineCount++
		lineText := scanner.Text()
		if len(lineText) > MaxLineLength {
			lineText = lineText[:MaxLineLength] + "..."
		}
		lines = append(lines, lineText)
	}

	// 继续扫描以获取总行数
	for scanner.Scan() {
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return "", 0, err
	}

	return strings.Join(lines, "\n"), lineCount, nil
}

func getImageMimeType(filePath string) (bool, string) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".jpg", ".jpeg":
		return true, "image/jpeg"
	case ".png":
		return true, "image/png"
	case ".gif":
		return true, "image/gif"
	case ".webp":
		return true, "image/webp"
	default:
		return false, ""
	}
}

type LineScanner struct {
	scanner *bufio.Scanner
}

func NewLineScanner(r io.Reader) *LineScanner {
	scanner := bufio.NewScanner(r)
	// 增加缓冲区大小以处理大行（例如，压缩的 JSON、HTML）
	// 默认值为 64KB，设置为 1MB
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	return &LineScanner{
		scanner: scanner,
	}
}

func (s *LineScanner) Scan() bool {
	return s.scanner.Scan()
}

func (s *LineScanner) Text() string {
	return s.scanner.Text()
}

func (s *LineScanner) Err() error {
	return s.scanner.Err()
}

// isInSkillsPath 检查 filePath 是否在任何配置的技能目录中
// 对于可以无权限提示读取且无大小限制的文件返回 true
//
// 注意：解析符号链接以防止通过符号链接进行路径遍历攻击
func isInSkillsPath(filePath string, skillsPaths []string) bool {
	if len(skillsPaths) == 0 {
		return false
	}

	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return false
	}

	evalFilePath, err := filepath.EvalSymlinks(absFilePath)
	if err != nil {
		return false
	}

	for _, skillsPath := range skillsPaths {
		absSkillsPath, err := filepath.Abs(skillsPath)
		if err != nil {
			continue
		}

		evalSkillsPath, err := filepath.EvalSymlinks(absSkillsPath)
		if err != nil {
			continue
		}

		relPath, err := filepath.Rel(evalSkillsPath, evalFilePath)
		if err == nil && !strings.HasPrefix(relPath, "..") {
			return true
		}
	}

	return false
}
