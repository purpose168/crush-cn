package tools

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"charm.land/fantasy"
)

type SourcegraphParams struct {
	Query         string `json:"query" description:"Sourcegraph搜索查询"`
	Count         int    `json:"count,omitempty" description:"可选的返回结果数量（默认：10，最大：20）"`
	ContextWindow int    `json:"context_window,omitempty" description:"返回匹配周围的上下文（默认：10行）"`
	Timeout       int    `json:"timeout,omitempty" description:"可选的超时时间（秒），最大120"`
}

type SourcegraphResponseMetadata struct {
	NumberOfMatches int  `json:"number_of_matches"`
	Truncated       bool `json:"truncated"`
}

const SourcegraphToolName = "sourcegraph"

//go:embed sourcegraph.md
var sourcegraphDescription []byte

// NewSourcegraphTool 创建一个新的Sourcegraph搜索工具实例
// client: HTTP客户端（如果为nil，将创建一个默认客户端）
func NewSourcegraphTool(client *http.Client) fantasy.AgentTool {
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
		SourcegraphToolName,
		string(sourcegraphDescription),
		func(ctx context.Context, params SourcegraphParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.Query == "" {
				return fantasy.NewTextErrorResponse("Query参数是必需的"), nil
			}

			if params.Count <= 0 {
				params.Count = 10
			} else if params.Count > 20 {
				params.Count = 20 // 限制为20个结果
			}

			if params.ContextWindow <= 0 {
				params.ContextWindow = 10 // 默认上下文窗口
			}

			// 使用上下文处理超时
			requestCtx := ctx
			if params.Timeout > 0 {
				maxTimeout := 120 // 2分钟
				if params.Timeout > maxTimeout {
					params.Timeout = maxTimeout
				}
				var cancel context.CancelFunc
				requestCtx, cancel = context.WithTimeout(ctx, time.Duration(params.Timeout)*time.Second)
				defer cancel()
			}

			type graphqlRequest struct {
				Query     string `json:"query"`
				Variables struct {
					Query string `json:"query"`
				} `json:"variables"`
			}

			request := graphqlRequest{
				Query: "query Search($query: String!) { search(query: $query, version: V2, patternType: keyword ) { results { matchCount, limitHit, resultCount, approximateResultCount, missing { name }, timedout { name }, indexUnavailable, results { __typename, ... on FileMatch { repository { name }, file { path, url, content }, lineMatches { preview, lineNumber, offsetAndLengths } } } } } }",
			}
			request.Variables.Query = params.Query

			graphqlQueryBytes, err := json.Marshal(request)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("序列化GraphQL请求失败: %w", err)
			}
			graphqlQuery := string(graphqlQueryBytes)

			req, err := http.NewRequestWithContext(
				requestCtx,
				"POST",
				"https://sourcegraph.com/.api/graphql",
				bytes.NewBuffer([]byte(graphqlQuery)),
			)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("创建请求失败: %w", err)
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("User-Agent", "crush/1.0")

			resp, err := client.Do(req)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("获取URL失败: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				if len(body) > 0 {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))), nil
				}

				return fantasy.NewTextErrorResponse(fmt.Sprintf("请求失败，状态码: %d", resp.StatusCode)), nil
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("读取响应体失败: %w", err)
			}

			var result map[string]any
			if err = json.Unmarshal(body, &result); err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("解析响应失败: %w", err)
			}

			formattedResults, err := formatSourcegraphResults(result, params.ContextWindow)
			if err != nil {
				return fantasy.NewTextErrorResponse("格式化结果失败: " + err.Error()), nil
			}

			return fantasy.NewTextResponse(formattedResults), nil
		})
}

// formatSourcegraphResults 格式化Sourcegraph搜索结果为人类可读的字符串
// result: 搜索结果数据
// contextWindow: 匹配周围的上下文行数
// 返回格式化的搜索结果字符串
func formatSourcegraphResults(result map[string]any, contextWindow int) (string, error) {
	var buffer strings.Builder

	if errors, ok := result["errors"].([]any); ok && len(errors) > 0 {
		buffer.WriteString("## Sourcegraph API 错误\n\n")
		for _, err := range errors {
			if errMap, ok := err.(map[string]any); ok {
				if message, ok := errMap["message"].(string); ok {
					buffer.WriteString(fmt.Sprintf("- %s\n", message))
				}
			}
		}
		return buffer.String(), nil
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("无效的响应格式: 缺少data字段")
	}

	search, ok := data["search"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("无效的响应格式: 缺少search字段")
	}

	searchResults, ok := search["results"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("无效的响应格式: 缺少results字段")
	}

	matchCount, _ := searchResults["matchCount"].(float64)
	resultCount, _ := searchResults["resultCount"].(float64)
	limitHit, _ := searchResults["limitHit"].(bool)

	buffer.WriteString("# Sourcegraph 搜索结果\n\n")
	buffer.WriteString(fmt.Sprintf("在 %d 个结果中找到 %d 个匹配\n", int(resultCount), int(matchCount)))

	if limitHit {
		buffer.WriteString("(已达到结果限制，请尝试更具体的查询)\n")
	}

	buffer.WriteString("\n")

	results, ok := searchResults["results"].([]any)
	if !ok || len(results) == 0 {
		buffer.WriteString("未找到结果。请尝试不同的查询。\n")
		return buffer.String(), nil
	}

	maxResults := 10
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	for i, res := range results {
		fileMatch, ok := res.(map[string]any)
		if !ok {
			continue
		}

		typeName, _ := fileMatch["__typename"].(string)
		if typeName != "FileMatch" {
			continue
		}

		repo, _ := fileMatch["repository"].(map[string]any)
		file, _ := fileMatch["file"].(map[string]any)
		lineMatches, _ := fileMatch["lineMatches"].([]any)

		if repo == nil || file == nil {
			continue
		}

		repoName, _ := repo["name"].(string)
		filePath, _ := file["path"].(string)
		fileURL, _ := file["url"].(string)
		fileContent, _ := file["content"].(string)

		buffer.WriteString(fmt.Sprintf("## 结果 %d: %s/%s\n\n", i+1, repoName, filePath))

		if fileURL != "" {
			buffer.WriteString(fmt.Sprintf("URL: %s\n\n", fileURL))
		}

		if len(lineMatches) > 0 {
			for _, lm := range lineMatches {
				lineMatch, ok := lm.(map[string]any)
				if !ok {
					continue
				}

				lineNumber, _ := lineMatch["lineNumber"].(float64)
				preview, _ := lineMatch["preview"].(string)

				if fileContent != "" {
					lines := strings.Split(fileContent, "\n")

					buffer.WriteString("```\n")

					startLine := max(1, int(lineNumber)-contextWindow)

					for j := startLine - 1; j < int(lineNumber)-1 && j < len(lines); j++ {
						if j >= 0 {
							buffer.WriteString(fmt.Sprintf("%d| %s\n", j+1, lines[j]))
						}
					}

					buffer.WriteString(fmt.Sprintf("%d|  %s\n", int(lineNumber), preview))

					endLine := int(lineNumber) + contextWindow

					for j := int(lineNumber); j < endLine && j < len(lines); j++ {
						if j < len(lines) {
							buffer.WriteString(fmt.Sprintf("%d| %s\n", j+1, lines[j]))
						}
					}

					buffer.WriteString("```\n\n")
				} else {
					buffer.WriteString("```\n")
					buffer.WriteString(fmt.Sprintf("%d| %s\n", int(lineNumber), preview))
					buffer.WriteString("```\n\n")
				}
			}
		}
	}

	return buffer.String(), nil
}
