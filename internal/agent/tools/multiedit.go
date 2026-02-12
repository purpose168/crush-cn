package tools

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/fantasy"
	"github.com/purpose168/crush-cn/internal/diff"
	"github.com/purpose168/crush-cn/internal/filepathext"
	"github.com/purpose168/crush-cn/internal/filetracker"
	"github.com/purpose168/crush-cn/internal/fsext"
	"github.com/purpose168/crush-cn/internal/history"
	"github.com/purpose168/crush-cn/internal/lsp"
	"github.com/purpose168/crush-cn/internal/permission"
)

type MultiEditOperation struct {
	OldString  string `json:"old_string" description:"要替换的文本"`
	NewString  string `json:"new_string" description:"替换为的文本"`
	ReplaceAll bool   `json:"replace_all,omitempty" description:"替换所有出现的old_string（默认false）"`
}

type MultiEditParams struct {
	FilePath string               `json:"file_path" description:"要修改的文件的绝对路径"`
	Edits    []MultiEditOperation `json:"edits" description:"要在文件上顺序执行的编辑操作数组"`
}

type MultiEditPermissionsParams struct {
	FilePath   string `json:"file_path"`
	OldContent string `json:"old_content,omitempty"`
	NewContent string `json:"new_content,omitempty"`
}

type FailedEdit struct {
	Index int                `json:"index"`
	Error string             `json:"error"`
	Edit  MultiEditOperation `json:"edit"`
}

type MultiEditResponseMetadata struct {
	Additions    int          `json:"additions"`
	Removals     int          `json:"removals"`
	OldContent   string       `json:"old_content,omitempty"`
	NewContent   string       `json:"new_content,omitempty"`
	EditsApplied int          `json:"edits_applied"`
	EditsFailed  []FailedEdit `json:"edits_failed,omitempty"`
}

const MultiEditToolName = "multiedit"

//go:embed multiedit.md
var multieditDescription []byte

// NewMultiEditTool 创建一个新的多重编辑工具实例
// lspManager: LSP客户端管理器
// permissions: 权限服务
// files: 文件历史服务
// filetracker: 文件跟踪服务
// workingDir: 工作目录
func NewMultiEditTool(
	lspManager *lsp.Manager,
	permissions permission.Service,
	files history.Service,
	filetracker filetracker.Service,
	workingDir string,
) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		MultiEditToolName,
		string(multieditDescription),
		func(ctx context.Context, params MultiEditParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.FilePath == "" {
				return fantasy.NewTextErrorResponse("file_path是必需的"), nil
			}

			if len(params.Edits) == 0 {
				return fantasy.NewTextErrorResponse("至少需要一个编辑操作"), nil
			}

			params.FilePath = filepathext.SmartJoin(workingDir, params.FilePath)

			// 在应用任何编辑之前验证所有编辑
			if err := validateEdits(params.Edits); err != nil {
				return fantasy.NewTextErrorResponse(err.Error()), nil
			}

			var response fantasy.ToolResponse
			var err error

			editCtx := editContext{ctx, permissions, files, filetracker, workingDir}
			// 处理文件创建情况（第一个编辑的old_string为空）
			if len(params.Edits) > 0 && params.Edits[0].OldString == "" {
				response, err = processMultiEditWithCreation(editCtx, params, call)
			} else {
				response, err = processMultiEditExistingFile(editCtx, params, call)
			}

			if err != nil {
				return response, err
			}

			if response.IsError {
				return response, nil
			}

			// 通知LSP客户端有关更改
			notifyLSPs(ctx, lspManager, params.FilePath)

			// 等待LSP诊断并将其添加到响应中
			text := fmt.Sprintf("<result>\n%s\n</result>\n", response.Content)
			text += getDiagnostics(params.FilePath, lspManager)
			response.Content = text
			return response, nil
		})
}

// validateEdits 验证编辑操作的有效性
// edits: 编辑操作数组
// 返回验证错误
func validateEdits(edits []MultiEditOperation) error {
	for i, edit := range edits {
		// 只有第一个编辑可以有空的old_string（用于文件创建）
		if i > 0 && edit.OldString == "" {
			return fmt.Errorf("编辑 %d: 只有第一个编辑可以有空的old_string（用于文件创建）", i+1)
		}
	}
	return nil
}

// processMultiEditWithCreation 处理创建文件的多重编辑操作
// edit: 编辑上下文
// params: 多重编辑参数
// call: 工具调用信息
// 返回工具响应
func processMultiEditWithCreation(edit editContext, params MultiEditParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
	// 第一个编辑创建文件
	firstEdit := params.Edits[0]
	if firstEdit.OldString != "" {
		return fantasy.NewTextErrorResponse("文件创建时第一个编辑必须有空的old_string"), nil
	}

	// 检查文件是否已存在
	if _, err := os.Stat(params.FilePath); err == nil {
		return fantasy.NewTextErrorResponse(fmt.Sprintf("文件已存在: %s", params.FilePath)), nil
	} else if !os.IsNotExist(err) {
		return fantasy.ToolResponse{}, fmt.Errorf("访问文件失败: %w", err)
	}

	// 创建父目录
	dir := filepath.Dir(params.FilePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("创建父目录失败: %w", err)
	}

	// 从第一个编辑的内容开始
	currentContent := firstEdit.NewString

	// 应用剩余的编辑操作到内容，跟踪失败的编辑
	var failedEdits []FailedEdit
	for i := 1; i < len(params.Edits); i++ {
		edit := params.Edits[i]
		newContent, err := applyEditToContent(currentContent, edit)
		if err != nil {
			failedEdits = append(failedEdits, FailedEdit{
				Index: i + 1,
				Error: err.Error(),
				Edit:  edit,
			})
			continue
		}
		currentContent = newContent
	}

	// 获取会话ID
	sessionID := GetSessionFromContext(edit.ctx)
	if sessionID == "" {
		return fantasy.ToolResponse{}, fmt.Errorf("创建新文件需要会话ID")
	}

	// 检查权限
	_, additions, removals := diff.GenerateDiff("", currentContent, strings.TrimPrefix(params.FilePath, edit.workingDir))

	editsApplied := len(params.Edits) - len(failedEdits)
	var description string
	if len(failedEdits) > 0 {
		description = fmt.Sprintf("创建文件 %s 并应用 %d/%d 个编辑（%d 个失败）", params.FilePath, editsApplied, len(params.Edits), len(failedEdits))
	} else {
		description = fmt.Sprintf("创建文件 %s 并应用 %d 个编辑", params.FilePath, editsApplied)
	}
	p, err := edit.permissions.Request(edit.ctx, permission.CreatePermissionRequest{
		SessionID:   sessionID,
		Path:        fsext.PathOrPrefix(params.FilePath, edit.workingDir),
		ToolCallID:  call.ID,
		ToolName:    MultiEditToolName,
		Action:      "write",
		Description: description,
		Params: MultiEditPermissionsParams{
			FilePath:   params.FilePath,
			OldContent: "",
			NewContent: currentContent,
		},
	})
	if err != nil {
		return fantasy.ToolResponse{}, err
	}
	if !p {
		return fantasy.ToolResponse{}, permission.ErrorPermissionDenied
	}

	// 写入文件
	err = os.WriteFile(params.FilePath, []byte(currentContent), 0o644)
	if err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("写入文件失败: %w", err)
	}

	// 更新文件历史
	_, err = edit.files.Create(edit.ctx, sessionID, params.FilePath, "")
	if err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("创建文件历史失败: %w", err)
	}

	_, err = edit.files.CreateVersion(edit.ctx, sessionID, params.FilePath, currentContent)
	if err != nil {
		slog.Error("创建文件历史版本失败", "error", err)
	}

	edit.filetracker.RecordRead(edit.ctx, sessionID, params.FilePath)

	var message string
	if len(failedEdits) > 0 {
		message = fmt.Sprintf("文件已创建，应用了 %d/%d 个编辑: %s （%d 个编辑失败）", editsApplied, len(params.Edits), params.FilePath, len(failedEdits))
	} else {
		message = fmt.Sprintf("文件已创建，应用了 %d 个编辑: %s", len(params.Edits), params.FilePath)
	}

	return fantasy.WithResponseMetadata(
		fantasy.NewTextResponse(message),
		MultiEditResponseMetadata{
			OldContent:   "",
			NewContent:   currentContent,
			Additions:    additions,
			Removals:     removals,
			EditsApplied: editsApplied,
			EditsFailed:  failedEdits,
		},
	), nil
}

// processMultiEditExistingFile 处理对现有文件的多重编辑操作
// edit: 编辑上下文
// params: 多重编辑参数
// call: 工具调用信息
// 返回工具响应
func processMultiEditExistingFile(edit editContext, params MultiEditParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
	// 验证文件存在且可读
	fileInfo, err := os.Stat(params.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fantasy.NewTextErrorResponse(fmt.Sprintf("文件未找到: %s", params.FilePath)), nil
		}
		return fantasy.ToolResponse{}, fmt.Errorf("访问文件失败: %w", err)
	}

	if fileInfo.IsDir() {
		return fantasy.NewTextErrorResponse(fmt.Sprintf("路径是目录，不是文件: %s", params.FilePath)), nil
	}

	sessionID := GetSessionFromContext(edit.ctx)
	if sessionID == "" {
		return fantasy.ToolResponse{}, fmt.Errorf("编辑文件需要会话ID")
	}

	// 检查文件在编辑前是否已被读取
	lastRead := edit.filetracker.LastReadTime(edit.ctx, sessionID, params.FilePath)
	if lastRead.IsZero() {
		return fantasy.NewTextErrorResponse("编辑文件前必须先读取它。请先使用View工具"), nil
	}

	// 检查文件自上次读取后是否被修改
	modTime := fileInfo.ModTime().Truncate(time.Second)
	if modTime.After(lastRead) {
		return fantasy.NewTextErrorResponse(
			fmt.Sprintf("文件 %s 自上次读取后已被修改（修改时间: %s, 上次读取: %s）",
				params.FilePath, modTime.Format(time.RFC3339), lastRead.Format(time.RFC3339),
			)), nil
	}

	// 读取当前文件内容
	content, err := os.ReadFile(params.FilePath)
	if err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("读取文件失败: %w", err)
	}

	oldContent, isCrlf := fsext.ToUnixLineEndings(string(content))
	currentContent := oldContent

	// 顺序应用所有编辑，跟踪失败的编辑
	var failedEdits []FailedEdit
	for i, edit := range params.Edits {
		newContent, err := applyEditToContent(currentContent, edit)
		if err != nil {
			failedEdits = append(failedEdits, FailedEdit{
				Index: i + 1,
				Error: err.Error(),
				Edit:  edit,
			})
			continue
		}
		currentContent = newContent
	}

	// 检查内容是否实际发生了变化
	if oldContent == currentContent {
		// 如果有失败的编辑，报告它们
		if len(failedEdits) > 0 {
			return fantasy.WithResponseMetadata(
				fantasy.NewTextErrorResponse(fmt.Sprintf("未做任何更改 - 所有 %d 个编辑都失败了", len(failedEdits))),
				MultiEditResponseMetadata{
					EditsApplied: 0,
					EditsFailed:  failedEdits,
				},
			), nil
		}
		return fantasy.NewTextErrorResponse("未做任何更改 - 所有编辑都导致内容相同"), nil
	}

	// 生成差异并检查权限
	_, additions, removals := diff.GenerateDiff(oldContent, currentContent, strings.TrimPrefix(params.FilePath, edit.workingDir))

	editsApplied := len(params.Edits) - len(failedEdits)
	var description string
	if len(failedEdits) > 0 {
		description = fmt.Sprintf("对文件 %s 应用 %d/%d 个编辑（%d 个失败）", editsApplied, len(params.Edits), params.FilePath, len(failedEdits))
	} else {
		description = fmt.Sprintf("对文件 %s 应用 %d 个编辑", editsApplied, params.FilePath)
	}
	p, err := edit.permissions.Request(edit.ctx, permission.CreatePermissionRequest{
		SessionID:   sessionID,
		Path:        fsext.PathOrPrefix(params.FilePath, edit.workingDir),
		ToolCallID:  call.ID,
		ToolName:    MultiEditToolName,
		Action:      "write",
		Description: description,
		Params: MultiEditPermissionsParams{
			FilePath:   params.FilePath,
			OldContent: oldContent,
			NewContent: currentContent,
		},
	})
	if err != nil {
		return fantasy.ToolResponse{}, err
	}
	if !p {
		return fantasy.ToolResponse{}, permission.ErrorPermissionDenied
	}

	if isCrlf {
		currentContent, _ = fsext.ToWindowsLineEndings(currentContent)
	}

	// 写入更新的内容
	err = os.WriteFile(params.FilePath, []byte(currentContent), 0o644)
	if err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("写入文件失败: %w", err)
	}

	// 更新文件历史
	file, err := edit.files.GetByPathAndSession(edit.ctx, params.FilePath, sessionID)
	if err != nil {
		_, err = edit.files.Create(edit.ctx, sessionID, params.FilePath, oldContent)
		if err != nil {
			return fantasy.ToolResponse{}, fmt.Errorf("创建文件历史失败: %w", err)
		}
	}
	if file.Content != oldContent {
		// 用户手动更改了内容，存储中间版本
		_, err = edit.files.CreateVersion(edit.ctx, sessionID, params.FilePath, oldContent)
		if err != nil {
			slog.Error("创建文件历史版本失败", "error", err)
		}
	}

	// 存储新版本
	_, err = edit.files.CreateVersion(edit.ctx, sessionID, params.FilePath, currentContent)
	if err != nil {
		slog.Error("创建文件历史版本失败", "error", err)
	}

	edit.filetracker.RecordRead(edit.ctx, sessionID, params.FilePath)

	var message string
	if len(failedEdits) > 0 {
		message = fmt.Sprintf("已对文件应用 %d/%d 个编辑: %s （%d 个编辑失败）", editsApplied, len(params.Edits), params.FilePath, len(failedEdits))
	} else {
		message = fmt.Sprintf("已对文件应用 %d 个编辑: %s", len(params.Edits), params.FilePath)
	}

	return fantasy.WithResponseMetadata(
		fantasy.NewTextResponse(message),
		MultiEditResponseMetadata{
			OldContent:   oldContent,
			NewContent:   currentContent,
			Additions:    additions,
			Removals:     removals,
			EditsApplied: editsApplied,
			EditsFailed:  failedEdits,
		},
	), nil
}

// applyEditToContent 将编辑操作应用到内容
// content: 原始内容
// edit: 编辑操作
// 返回更新后的内容
func applyEditToContent(content string, edit MultiEditOperation) (string, error) {
	if edit.OldString == "" && edit.NewString == "" {
		return content, nil
	}

	if edit.OldString == "" {
		return "", fmt.Errorf("内容替换时old_string不能为空")
	}

	var newContent string
	var replacementCount int

	if edit.ReplaceAll {
		newContent = strings.ReplaceAll(content, edit.OldString, edit.NewString)
		replacementCount = strings.Count(content, edit.OldString)
		if replacementCount == 0 {
			return "", fmt.Errorf("在内容中未找到old_string。请确保它完全匹配，包括空格和换行符")
		}
	} else {
		index := strings.Index(content, edit.OldString)
		if index == -1 {
			return "", fmt.Errorf("在内容中未找到old_string。请确保它完全匹配，包括空格和换行符")
		}

		lastIndex := strings.LastIndex(content, edit.OldString)
		if index != lastIndex {
			return "", fmt.Errorf("old_string在内容中出现多次。请提供更多上下文以确保唯一匹配，或将replace_all设置为true")
		}

		newContent = content[:index] + edit.NewString + content[index+len(edit.OldString):]
		replacementCount = 1
	}

	return newContent, nil
}
