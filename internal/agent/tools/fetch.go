package tools

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"charm.land/fantasy"
	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/purpose168/crush-cn/internal/permission"
)

const FetchToolName = "fetch"

//go:embed fetch.md
var fetchDescription []byte

// NewFetchTool 创建一个新的抓取工具实例
// permissions: 权限服务
// workingDir: 工作目录
// client: HTTP客户端（如果为nil，将创建一个默认客户端）
func NewFetchTool(permissions permission.Service, workingDir string, client *http.Client) fantasy.AgentTool {
	if client == nil {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.MaxIdleConns = 100
		transport.MaxIdleConnsPerHost = 10
		transport.IdleConnTimeout = 90 * time.Second

		client = &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		}
	}

	return fantasy.NewParallelAgentTool(
		FetchToolName,
		string(fetchDescription),
		func(ctx context.Context, params FetchParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.URL == "" {
				return fantasy.NewTextErrorResponse("URL参数是必需的"), nil
			}

			format := strings.ToLower(params.Format)
			if format != "text" && format != "markdown" && format != "html" {
				return fantasy.NewTextErrorResponse("格式必须是以下之一: text, markdown, html"), nil
			}

			if !strings.HasPrefix(params.URL, "http://") && !strings.HasPrefix(params.URL, "https://") {
				return fantasy.NewTextErrorResponse("URL必须以http://或https://开头"), nil
			}

			sessionID := GetSessionFromContext(ctx)
			if sessionID == "" {
				return fantasy.ToolResponse{}, fmt.Errorf("创建新文件需要会话ID")
			}

			p, err := permissions.Request(ctx,
				permission.CreatePermissionRequest{
					SessionID:   sessionID,
					Path:        workingDir,
					ToolCallID:  call.ID,
					ToolName:    FetchToolName,
					Action:      "fetch",
					Description: fmt.Sprintf("从URL抓取内容: %s", params.URL),
					Params:      FetchPermissionsParams(params),
				},
			)
			if err != nil {
				return fantasy.ToolResponse{}, err
			}
			if !p {
				return fantasy.ToolResponse{}, permission.ErrorPermissionDenied
			}

			// maxFetchTimeoutSeconds 是抓取请求允许的最大超时时间（2分钟）
			const maxFetchTimeoutSeconds = 120

			// 使用上下文处理超时
			requestCtx := ctx
			if params.Timeout > 0 {
				if params.Timeout > maxFetchTimeoutSeconds {
					params.Timeout = maxFetchTimeoutSeconds
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
				return fantasy.ToolResponse{}, fmt.Errorf("抓取URL失败: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("请求失败，状态码: %d", resp.StatusCode)), nil
			}

			// maxFetchResponseSizeBytes 是要读取的响应体的最大大小（5MB）
			const maxFetchResponseSizeBytes = int64(5 * 1024 * 1024)

			maxSize := maxFetchResponseSizeBytes
			body, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
			if err != nil {
				return fantasy.NewTextErrorResponse("读取响应体失败: " + err.Error()), nil
			}

			content := string(body)

			validUTF8 := utf8.ValidString(content)
			if !validUTF8 {
				return fantasy.NewTextErrorResponse("响应内容不是有效的UTF-8"), nil
			}
			contentType := resp.Header.Get("Content-Type")

			switch format {
			case "text":
				if strings.Contains(contentType, "text/html") {
					text, err := extractTextFromHTML(content)
					if err != nil {
						return fantasy.NewTextErrorResponse("从HTML提取文本失败: " + err.Error()), nil
					}
					content = text
				}

			case "markdown":
				if strings.Contains(contentType, "text/html") {
					markdown, err := convertHTMLToMarkdown(content)
					if err != nil {
						return fantasy.NewTextErrorResponse("将HTML转换为Markdown失败: " + err.Error()), nil
					}
					content = markdown
				}

				content = "```\n" + content + "\n```"

			case "html":
				// 只返回HTML文档的body部分
				if strings.Contains(contentType, "text/html") {
					doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
					if err != nil {
						return fantasy.NewTextErrorResponse("解析HTML失败: " + err.Error()), nil
					}
					body, err := doc.Find("body").Html()
					if err != nil {
						return fantasy.NewTextErrorResponse("从HTML提取body失败: " + err.Error()), nil
					}
					if body == "" {
						return fantasy.NewTextErrorResponse("在HTML中未找到body内容"), nil
					}
					content = "<html>\n<body>\n" + body + "\n</body>\n</html>"
				}
			}
			// 如果内容超过最大读取大小，则截断
			if int64(len(content)) > MaxReadSize {
				content = content[:MaxReadSize]
				content += fmt.Sprintf("\n\n[内容已截断为 %d 字节]", MaxReadSize)
			}

			return fantasy.NewTextResponse(content), nil
		})
}

// extractTextFromHTML 从HTML中提取纯文本
// html: HTML内容
// 返回提取的纯文本
func extractTextFromHTML(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}

	text := doc.Find("body").Text()
	text = strings.Join(strings.Fields(text), " ")

	return text, nil
}

// convertHTMLToMarkdown 将HTML转换为Markdown格式
// html: HTML内容
// 返回转换后的Markdown内容
func convertHTMLToMarkdown(html string) (string, error) {
	converter := md.NewConverter("", true, nil)

	markdown, err := converter.ConvertString(html)
	if err != nil {
		return "", err
	}

	return markdown, nil
}
