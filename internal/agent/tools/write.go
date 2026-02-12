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

//go:embed write.md
var writeDescription []byte

type WriteParams struct {
	FilePath string `json:"file_path" description:"要写入的文件路径"`
	Content  string `json:"content" description:"要写入文件的内容"`
}

type WritePermissionsParams struct {
	FilePath   string `json:"file_path"`
	OldContent string `json:"old_content,omitempty"`
	NewContent string `json:"new_content,omitempty"`
}

type WriteResponseMetadata struct {
	Diff      string `json:"diff"`
	Additions int    `json:"additions"`
	Removals  int    `json:"removals"`
}

const WriteToolName = "write"

func NewWriteTool(
	lspManager *lsp.Manager,
	permissions permission.Service,
	files history.Service,
	filetracker filetracker.Service,
	workingDir string,
) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		WriteToolName,
		string(writeDescription),
		func(ctx context.Context, params WriteParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.FilePath == "" {
				return fantasy.NewTextErrorResponse("需要提供 file_path"), nil
			}

			if params.Content == "" {
				return fantasy.NewTextErrorResponse("需要提供 content"), nil
			}

			sessionID := GetSessionFromContext(ctx)
			if sessionID == "" {
				return fantasy.ToolResponse{}, fmt.Errorf("需要 session_id")
			}

			filePath := filepathext.SmartJoin(workingDir, params.FilePath)

			fileInfo, err := os.Stat(filePath)
			if err == nil {
				if fileInfo.IsDir() {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("路径是目录，不是文件: %s", filePath)), nil
				}

				modTime := fileInfo.ModTime().Truncate(time.Second)
				lastRead := filetracker.LastReadTime(ctx, sessionID, filePath)
				if modTime.After(lastRead) {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("文件 %s 自上次读取后已被修改。\n上次修改: %s\n上次读取: %s\n\n请在修改前再次读取文件。",
						filePath, modTime.Format(time.RFC3339), lastRead.Format(time.RFC3339))), nil
				}

				oldContent, readErr := os.ReadFile(filePath)
				if readErr == nil && string(oldContent) == params.Content {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("文件 %s 已包含完全相同的内容。未进行任何更改。", filePath)), nil
				}
			} else if !os.IsNotExist(err) {
				return fantasy.ToolResponse{}, fmt.Errorf("检查文件错误: %w", err)
			}

			dir := filepath.Dir(filePath)
			if err = os.MkdirAll(dir, 0o755); err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("创建目录错误: %w", err)
			}

			oldContent := ""
			if fileInfo != nil && !fileInfo.IsDir() {
				oldBytes, readErr := os.ReadFile(filePath)
				if readErr == nil {
					oldContent = string(oldBytes)
				}
			}

			diff, additions, removals := diff.GenerateDiff(
				oldContent,
				params.Content,
				strings.TrimPrefix(filePath, workingDir),
			)

			p, err := permissions.Request(ctx,
				permission.CreatePermissionRequest{
					SessionID:   sessionID,
					Path:        fsext.PathOrPrefix(filePath, workingDir),
					ToolCallID:  call.ID,
					ToolName:    WriteToolName,
					Action:      "write",
					Description: fmt.Sprintf("创建文件 %s", filePath),
					Params: WritePermissionsParams{
						FilePath:   filePath,
						OldContent: oldContent,
						NewContent: params.Content,
					},
				},
			)
			if err != nil {
				return fantasy.ToolResponse{}, err
			}
			if !p {
				return fantasy.ToolResponse{}, permission.ErrorPermissionDenied
			}

			err = os.WriteFile(filePath, []byte(params.Content), 0o644)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("写入文件错误: %w", err)
			}

			// 检查文件是否存在于历史记录中
			file, err := files.GetByPathAndSession(ctx, filePath, sessionID)
			if err != nil {
				_, err = files.Create(ctx, sessionID, filePath, oldContent)
				if err != nil {
					// 记录错误但不中断操作
					return fantasy.ToolResponse{}, fmt.Errorf("创建文件历史错误: %w", err)
				}
			}
			if file.Content != oldContent {
				// 用户手动更改了内容；存储中间版本
				_, err = files.CreateVersion(ctx, sessionID, filePath, oldContent)
				if err != nil {
					slog.Error("创建文件历史版本错误", "error", err)
				}
			}
			// 存储新版本
			_, err = files.CreateVersion(ctx, sessionID, filePath, params.Content)
			if err != nil {
				slog.Error("创建文件历史版本错误", "error", err)
			}

			filetracker.RecordRead(ctx, sessionID, filePath)

			notifyLSPs(ctx, lspManager, params.FilePath)

			result := fmt.Sprintf("文件写入成功: %s", filePath)
			result = fmt.Sprintf("<result>\n%s\n</result>", result)
			result += getDiagnostics(filePath, lspManager)
			return fantasy.WithResponseMetadata(fantasy.NewTextResponse(result),
				WriteResponseMetadata{
					Diff:      diff,
					Additions: additions,
					Removals:  removals,
				},
			), nil
		})
}
