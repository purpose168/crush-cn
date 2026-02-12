package tools

import (
	"cmp"
	"context"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/fantasy"
	"github.com/purpose168/crush-cn/internal/filepathext"
	"github.com/purpose168/crush-cn/internal/permission"
)

type DownloadParams struct {
	URL      string `json:"url" description:"要下载的URL地址"`
	FilePath string `json:"file_path" description:"下载内容应保存的本地文件路径"`
	Timeout  int    `json:"timeout,omitempty" description:"可选的超时时间（秒），最大600秒"`
}

type DownloadPermissionsParams struct {
	URL      string `json:"url"`
	FilePath string `json:"file_path"`
	Timeout  int    `json:"timeout,omitempty"`
}

const DownloadToolName = "download"

//go:embed download.md
var downloadDescription []byte

// NewDownloadTool 创建一个新的下载工具实例
// permissions: 权限服务
// workingDir: 工作目录
// client: HTTP客户端（如果为nil，将创建一个默认客户端）
func NewDownloadTool(permissions permission.Service, workingDir string, client *http.Client) fantasy.AgentTool {
	if client == nil {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.MaxIdleConns = 100
		transport.MaxIdleConnsPerHost = 10
		transport.IdleConnTimeout = 90 * time.Second

		client = &http.Client{
			Timeout:   5 * time.Minute, // 下载默认超时时间为5分钟
			Transport: transport,
		}
	}
	return fantasy.NewParallelAgentTool(
		DownloadToolName,
		string(downloadDescription),
		func(ctx context.Context, params DownloadParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.URL == "" {
				return fantasy.NewTextErrorResponse("URL参数是必需的"), nil
			}

			if params.FilePath == "" {
				return fantasy.NewTextErrorResponse("file_path参数是必需的"), nil
			}

			if !strings.HasPrefix(params.URL, "http://") && !strings.HasPrefix(params.URL, "https://") {
				return fantasy.NewTextErrorResponse("URL必须以http://或https://开头"), nil
			}

			filePath := filepathext.SmartJoin(workingDir, params.FilePath)
			relPath, _ := filepath.Rel(workingDir, filePath)
			relPath = filepath.ToSlash(cmp.Or(relPath, filePath))

			sessionID := GetSessionFromContext(ctx)
			if sessionID == "" {
				return fantasy.ToolResponse{}, fmt.Errorf("下载文件需要会话ID")
			}

			p, err := permissions.Request(ctx,
				permission.CreatePermissionRequest{
					SessionID:   sessionID,
					Path:        filePath,
					ToolName:    DownloadToolName,
					Action:      "download",
					Description: fmt.Sprintf("从URL下载文件: %s 到 %s", params.URL, filePath),
					Params:      DownloadPermissionsParams(params),
				},
			)
			if err != nil {
				return fantasy.ToolResponse{}, err
			}
			if !p {
				return fantasy.ToolResponse{}, permission.ErrorPermissionDenied
			}

			// 使用上下文处理超时
			requestCtx := ctx
			if params.Timeout > 0 {
				maxTimeout := 600 // 10分钟
				if params.Timeout > maxTimeout {
					params.Timeout = maxTimeout
				}
				var cancel context.CancelFunc
				requestCtx, cancel = context.WithTimeout(ctx, time.Duration(params.Timeout)*time.Second)
				defer cancel()
			}

			req, err := http.NewRequestWithContext(requestCtx, "GET", params.URL, nil)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("创建请求失败: %w", err)
			}

			req.Header.Set("User-Agent", "crush/1.0")

			resp, err := client.Do(req)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("从URL下载失败: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("请求失败，状态码: %d", resp.StatusCode)), nil
			}

			// 如果父目录不存在，则创建
			if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("创建父目录失败: %w", err)
			}

			// 创建输出文件
			outFile, err := os.Create(filePath)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("创建输出文件失败: %w", err)
			}
			defer outFile.Close()

			// 复制数据，不设置显式大小限制
			// 整体下载仍然受到HTTP客户端超时和上游服务器限制的约束
			bytesWritten, err := io.Copy(outFile, resp.Body)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("写入文件失败: %w", err)
			}

			contentType := resp.Header.Get("Content-Type")
			responseMsg := fmt.Sprintf("成功下载 %d 字节到 %s", bytesWritten, relPath)
			if contentType != "" {
				responseMsg += fmt.Sprintf(" (Content-Type: %s)", contentType)
			}

			return fantasy.NewTextResponse(responseMsg), nil
		})
}
