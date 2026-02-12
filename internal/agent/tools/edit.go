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

type EditParams struct {
	FilePath   string `json:"file_path" description:"要修改的文件的绝对路径"`
	OldString  string `json:"old_string" description:"要替换的文本"`
	NewString  string `json:"new_string" description:"替换后的文本"`
	ReplaceAll bool   `json:"replace_all,omitempty" description:"替换所有出现的 old_string（默认 false）"`
}

type EditPermissionsParams struct {
	FilePath   string `json:"file_path"`
	OldContent string `json:"old_content,omitempty"`
	NewContent string `json:"new_content,omitempty"`
}

type EditResponseMetadata struct {
	Additions  int    `json:"additions"`
	Removals   int    `json:"removals"`
	OldContent string `json:"old_content,omitempty"`
	NewContent string `json:"new_content,omitempty"`
}

const EditToolName = "edit"

var (
	oldStringNotFoundErr        = fantasy.NewTextErrorResponse("在文件中未找到 old_string。确保它完全匹配，包括空白字符和换行符。")
	oldStringMultipleMatchesErr = fantasy.NewTextErrorResponse("old_string 在文件中出现多次。请提供更多上下文以确保唯一匹配，或设置 replace_all 为 true")
)

//go:embed edit.md
var editDescription []byte

type editContext struct {
	ctx         context.Context
	permissions permission.Service
	files       history.Service
	filetracker filetracker.Service
	workingDir  string
}

func NewEditTool(
	lspManager *lsp.Manager,
	permissions permission.Service,
	files history.Service,
	filetracker filetracker.Service,
	workingDir string,
) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		EditToolName,
		string(editDescription),
		func(ctx context.Context, params EditParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.FilePath == "" {
				return fantasy.NewTextErrorResponse("需要提供 file_path"), nil
			}

			params.FilePath = filepathext.SmartJoin(workingDir, params.FilePath)

			var response fantasy.ToolResponse
			var err error

			editCtx := editContext{ctx, permissions, files, filetracker, workingDir}

			if params.OldString == "" {
				response, err = createNewFile(editCtx, params.FilePath, params.NewString, call)
			} else if params.NewString == "" {
				response, err = deleteContent(editCtx, params.FilePath, params.OldString, params.ReplaceAll, call)
			} else {
				response, err = replaceContent(editCtx, params.FilePath, params.OldString, params.NewString, params.ReplaceAll, call)
			}

			if err != nil {
				return response, err
			}
			if response.IsError {
				// 如果内容替换过程中出错，提前返回
				// 这可以防止不必要的 LSP 诊断处理
				return response, nil
			}

			notifyLSPs(ctx, lspManager, params.FilePath)

			text := fmt.Sprintf("<result>\n%s\n</result>\n", response.Content)
			text += getDiagnostics(params.FilePath, lspManager)
			response.Content = text
			return response, nil
		})
}

func createNewFile(edit editContext, filePath, content string, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
	fileInfo, err := os.Stat(filePath)
	if err == nil {
		if fileInfo.IsDir() {
			return fantasy.NewTextErrorResponse(fmt.Sprintf("路径是目录，不是文件: %s", filePath)), nil
		}
		return fantasy.NewTextErrorResponse(fmt.Sprintf("文件已存在: %s", filePath)), nil
	} else if !os.IsNotExist(err) {
		return fantasy.ToolResponse{}, fmt.Errorf("访问文件失败: %w", err)
	}

	dir := filepath.Dir(filePath)
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("创建父目录失败: %w", err)
	}

	sessionID := GetSessionFromContext(edit.ctx)
	if sessionID == "" {
		return fantasy.ToolResponse{}, fmt.Errorf("创建新文件需要会话 ID")
	}

	_, additions, removals := diff.GenerateDiff(
		"",
		content,
		strings.TrimPrefix(filePath, edit.workingDir),
	)
	p, err := edit.permissions.Request(edit.ctx,
		permission.CreatePermissionRequest{
			SessionID:   sessionID,
			Path:        fsext.PathOrPrefix(filePath, edit.workingDir),
			ToolCallID:  call.ID,
			ToolName:    EditToolName,
			Action:      "write",
			Description: fmt.Sprintf("创建文件 %s", filePath),
			Params: EditPermissionsParams{
				FilePath:   filePath,
				OldContent: "",
				NewContent: content,
			},
		},
	)
	if err != nil {
		return fantasy.ToolResponse{}, err
	}
	if !p {
		return fantasy.ToolResponse{}, permission.ErrorPermissionDenied
	}

	err = os.WriteFile(filePath, []byte(content), 0o644)
	if err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("写入文件失败: %w", err)
	}

	// 文件不可能在历史记录中，所以我们创建一个新的文件历史
	_, err = edit.files.Create(edit.ctx, sessionID, filePath, "")
	if err != nil {
		// 记录错误但不中断操作
		return fantasy.ToolResponse{}, fmt.Errorf("创建文件历史错误: %w", err)
	}

	// 将新内容添加到文件历史
	_, err = edit.files.CreateVersion(edit.ctx, sessionID, filePath, content)
	if err != nil {
		// 记录错误但不中断操作
		slog.Error("创建文件历史版本错误", "error", err)
	}

	edit.filetracker.RecordRead(edit.ctx, sessionID, filePath)

	return fantasy.WithResponseMetadata(
		fantasy.NewTextResponse("文件已创建: "+filePath),
		EditResponseMetadata{
			OldContent: "",
			NewContent: content,
			Additions:  additions,
			Removals:   removals,
		},
	), nil
}

func deleteContent(edit editContext, filePath, oldString string, replaceAll bool, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fantasy.NewTextErrorResponse(fmt.Sprintf("文件未找到: %s", filePath)), nil
		}
		return fantasy.ToolResponse{}, fmt.Errorf("访问文件失败: %w", err)
	}

	if fileInfo.IsDir() {
		return fantasy.NewTextErrorResponse(fmt.Sprintf("路径是目录，不是文件: %s", filePath)), nil
	}

	sessionID := GetSessionFromContext(edit.ctx)
	if sessionID == "" {
		return fantasy.ToolResponse{}, fmt.Errorf("删除内容需要会话 ID")
	}

	lastRead := edit.filetracker.LastReadTime(edit.ctx, sessionID, filePath)
	if lastRead.IsZero() {
		return fantasy.NewTextErrorResponse("编辑文件前必须先读取它。请先使用 View 工具"), nil
	}

	modTime := fileInfo.ModTime().Truncate(time.Second)
	if modTime.After(lastRead) {
		return fantasy.NewTextErrorResponse(
			fmt.Sprintf("文件 %s 自上次读取后已被修改（修改时间: %s，上次读取: %s）",
				filePath, modTime.Format(time.RFC3339), lastRead.Format(time.RFC3339),
			)), nil
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("读取文件失败: %w", err)
	}

	oldContent, isCrlf := fsext.ToUnixLineEndings(string(content))

	var newContent string

	if replaceAll {
		newContent = strings.ReplaceAll(oldContent, oldString, "")
		if newContent == oldContent {
			return oldStringNotFoundErr, nil
		}
	} else {
		index := strings.Index(oldContent, oldString)
		if index == -1 {
			return oldStringNotFoundErr, nil
		}

		lastIndex := strings.LastIndex(oldContent, oldString)
		if index != lastIndex {
			return fantasy.NewTextErrorResponse("old_string 在文件中出现多次。请提供更多上下文以确保唯一匹配，或设置 replace_all 为 true"), nil
		}

		newContent = oldContent[:index] + oldContent[index+len(oldString):]
	}

	_, additions, removals := diff.GenerateDiff(
		oldContent,
		newContent,
		strings.TrimPrefix(filePath, edit.workingDir),
	)

	p, err := edit.permissions.Request(edit.ctx,
		permission.CreatePermissionRequest{
			SessionID:   sessionID,
			Path:        fsext.PathOrPrefix(filePath, edit.workingDir),
			ToolCallID:  call.ID,
			ToolName:    EditToolName,
			Action:      "write",
			Description: fmt.Sprintf("从文件 %s 中删除内容", filePath),
			Params: EditPermissionsParams{
				FilePath:   filePath,
				OldContent: oldContent,
				NewContent: newContent,
			},
		},
	)
	if err != nil {
		return fantasy.ToolResponse{}, err
	}
	if !p {
		return fantasy.ToolResponse{}, permission.ErrorPermissionDenied
	}

	if isCrlf {
		newContent, _ = fsext.ToWindowsLineEndings(newContent)
	}

	err = os.WriteFile(filePath, []byte(newContent), 0o644)
	if err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("写入文件失败: %w", err)
	}

	// 检查文件是否存在于历史记录中
	file, err := edit.files.GetByPathAndSession(edit.ctx, filePath, sessionID)
	if err != nil {
		_, err = edit.files.Create(edit.ctx, sessionID, filePath, oldContent)
		if err != nil {
			// 记录错误但不中断操作
			return fantasy.ToolResponse{}, fmt.Errorf("创建文件历史错误: %w", err)
		}
	}
	if file.Content != oldContent {
		// 用户手动更改了内容；存储中间版本
		_, err = edit.files.CreateVersion(edit.ctx, sessionID, filePath, oldContent)
		if err != nil {
			slog.Error("创建文件历史版本错误", "error", err)
		}
	}
	// 存储新版本
	_, err = edit.files.CreateVersion(edit.ctx, sessionID, filePath, newContent)
	if err != nil {
		slog.Error("创建文件历史版本错误", "error", err)
	}

	edit.filetracker.RecordRead(edit.ctx, sessionID, filePath)

	return fantasy.WithResponseMetadata(
		fantasy.NewTextResponse("已从文件中删除内容: "+filePath),
		EditResponseMetadata{
			OldContent: oldContent,
			NewContent: newContent,
			Additions:  additions,
			Removals:   removals,
		},
	), nil
}

func replaceContent(edit editContext, filePath, oldString, newString string, replaceAll bool, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fantasy.NewTextErrorResponse(fmt.Sprintf("文件未找到: %s", filePath)), nil
		}
		return fantasy.ToolResponse{}, fmt.Errorf("访问文件失败: %w", err)
	}

	if fileInfo.IsDir() {
		return fantasy.NewTextErrorResponse(fmt.Sprintf("路径是目录，不是文件: %s", filePath)), nil
	}

	sessionID := GetSessionFromContext(edit.ctx)
	if sessionID == "" {
		return fantasy.ToolResponse{}, fmt.Errorf("编辑文件需要会话 ID")
	}

	lastRead := edit.filetracker.LastReadTime(edit.ctx, sessionID, filePath)
	if lastRead.IsZero() {
		return fantasy.NewTextErrorResponse("编辑文件前必须先读取它。请先使用 View 工具"), nil
	}

	modTime := fileInfo.ModTime().Truncate(time.Second)
	if modTime.After(lastRead) {
		return fantasy.NewTextErrorResponse(
			fmt.Sprintf("文件 %s 自上次读取后已被修改（修改时间: %s，上次读取: %s）",
				filePath, modTime.Format(time.RFC3339), lastRead.Format(time.RFC3339),
			)), nil
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("读取文件失败: %w", err)
	}

	oldContent, isCrlf := fsext.ToUnixLineEndings(string(content))

	var newContent string

	if replaceAll {
		newContent = strings.ReplaceAll(oldContent, oldString, newString)
	} else {
		index := strings.Index(oldContent, oldString)
		if index == -1 {
			return oldStringNotFoundErr, nil
		}

		lastIndex := strings.LastIndex(oldContent, oldString)
		if index != lastIndex {
			return oldStringMultipleMatchesErr, nil
		}

		newContent = oldContent[:index] + newString + oldContent[index+len(oldString):]
	}

	if oldContent == newContent {
		return fantasy.NewTextErrorResponse("新内容与旧内容相同。未进行任何更改。"), nil
	}
	_, additions, removals := diff.GenerateDiff(
		oldContent,
		newContent,
		strings.TrimPrefix(filePath, edit.workingDir),
	)

	p, err := edit.permissions.Request(edit.ctx,
		permission.CreatePermissionRequest{
			SessionID:   sessionID,
			Path:        fsext.PathOrPrefix(filePath, edit.workingDir),
			ToolCallID:  call.ID,
			ToolName:    EditToolName,
			Action:      "write",
			Description: fmt.Sprintf("替换文件 %s 中的内容", filePath),
			Params: EditPermissionsParams{
				FilePath:   filePath,
				OldContent: oldContent,
				NewContent: newContent,
			},
		},
	)
	if err != nil {
		return fantasy.ToolResponse{}, err
	}
	if !p {
		return fantasy.ToolResponse{}, permission.ErrorPermissionDenied
	}

	if isCrlf {
		newContent, _ = fsext.ToWindowsLineEndings(newContent)
	}

	err = os.WriteFile(filePath, []byte(newContent), 0o644)
	if err != nil {
		return fantasy.ToolResponse{}, fmt.Errorf("写入文件失败: %w", err)
	}

	// 检查文件是否存在于历史记录中
	file, err := edit.files.GetByPathAndSession(edit.ctx, filePath, sessionID)
	if err != nil {
		_, err = edit.files.Create(edit.ctx, sessionID, filePath, oldContent)
		if err != nil {
			// 记录错误但不中断操作
			return fantasy.ToolResponse{}, fmt.Errorf("创建文件历史错误: %w", err)
		}
	}
	if file.Content != oldContent {
		// 用户手动更改了内容；存储中间版本
		_, err = edit.files.CreateVersion(edit.ctx, sessionID, filePath, oldContent)
		if err != nil {
			slog.Debug("创建文件历史版本错误", "error", err)
		}
	}
	// 存储新版本
	_, err = edit.files.CreateVersion(edit.ctx, sessionID, filePath, newContent)
	if err != nil {
		slog.Error("创建文件历史版本错误", "error", err)
	}

	edit.filetracker.RecordRead(edit.ctx, sessionID, filePath)

	return fantasy.WithResponseMetadata(
		fantasy.NewTextResponse("已替换文件中的内容: "+filePath),
		EditResponseMetadata{
			OldContent: oldContent,
			NewContent: newContent,
			Additions:  additions,
			Removals:   removals,
		}), nil
}
