package tools

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

// SearchResult 表示来自DuckDuckGo的单个搜索结果

type SearchResult struct {
	Title    string // 搜索结果标题
	Link     string // 搜索结果链接
	Snippet  string // 搜索结果摘要
	Position int    // 搜索结果位置
}

// userAgents 是一个用户代理字符串列表，用于随机化HTTP请求头
var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:133.0) Gecko/20100101 Firefox/133.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:132.0) Gecko/20100101 Firefox/132.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:133.0) Gecko/20100101 Firefox/133.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.1 Safari/605.1.15",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0",
}

// acceptLanguages 是一个接受语言字符串列表，用于随机化HTTP请求头
var acceptLanguages = []string{
	"en-US,en;q=0.9",
	"en-US,en;q=0.9,es;q=0.8",
	"en-GB,en;q=0.9,en-US;q=0.8",
	"en-US,en;q=0.5",
	"en-CA,en;q=0.9,en-US;q=0.8",
}

// searchDuckDuckGo 使用DuckDuckGo进行网络搜索
// ctx: 上下文对象
// client: HTTP客户端
// query: 搜索查询
// maxResults: 最大结果数
// 返回搜索结果列表
func searchDuckDuckGo(ctx context.Context, client *http.Client, query string, maxResults int) ([]SearchResult, error) {
	if maxResults <= 0 {
		maxResults = 10
	}

	searchURL := "https://lite.duckduckgo.com/lite/?q=" + url.QueryEscape(query)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	setRandomizedHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("执行搜索失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("搜索失败，状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	return parseLiteSearchResults(string(body), maxResults)
}

// setRandomizedHeaders 为HTTP请求设置随机化的请求头
// req: HTTP请求对象
func setRandomizedHeaders(req *http.Request) {
	req.Header.Set("User-Agent", userAgents[rand.IntN(len(userAgents))])
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", acceptLanguages[rand.IntN(len(acceptLanguages))])
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Cache-Control", "max-age=0")
	if rand.IntN(2) == 0 {
		req.Header.Set("DNT", "1")
	}
}

// parseLiteSearchResults 解析DuckDuckGo Lite搜索结果
// htmlContent: HTML内容
// maxResults: 最大结果数
// 返回搜索结果列表
func parseLiteSearchResults(htmlContent string, maxResults int) ([]SearchResult, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("解析HTML失败: %w", err)
	}

	var results []SearchResult
	var currentResult *SearchResult

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "a" && hasClass(n, "result-link") {
				if currentResult != nil && currentResult.Link != "" {
					currentResult.Position = len(results) + 1
					results = append(results, *currentResult)
					if len(results) >= maxResults {
						return
					}
				}
				currentResult = &SearchResult{Title: getTextContent(n)}
				for _, attr := range n.Attr {
					if attr.Key == "href" {
						currentResult.Link = cleanDuckDuckGoURL(attr.Val)
						break
					}
				}
			}
			if n.Data == "td" && hasClass(n, "result-snippet") && currentResult != nil {
				currentResult.Snippet = getTextContent(n)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if len(results) >= maxResults {
				return
			}
			traverse(c)
		}
	}

	traverse(doc)

	if currentResult != nil && currentResult.Link != "" && len(results) < maxResults {
		currentResult.Position = len(results) + 1
		results = append(results, *currentResult)
	}

	return results, nil
}

// hasClass 检查HTML节点是否包含指定的class
// n: HTML节点
// class: 要检查的class名称
// 返回节点是否包含指定的class
func hasClass(n *html.Node, class string) bool {
	for _, attr := range n.Attr {
		if attr.Key == "class" {
			if slices.Contains(strings.Fields(attr.Val), class) {
				return true
			}
		}
	}
	return false
}

// getTextContent 获取HTML节点的文本内容
// n: HTML节点
// 返回节点的文本内容
func getTextContent(n *html.Node) string {
	var text strings.Builder
	var traverse func(*html.Node)
	traverse = func(node *html.Node) {
		if node.Type == html.TextNode {
			text.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(n)
	return strings.TrimSpace(text.String())
}

// cleanDuckDuckGoURL 清理DuckDuckGo URL，去除重定向参数
// rawURL: 原始URL
// 返回清理后的URL
func cleanDuckDuckGoURL(rawURL string) string {
	if strings.HasPrefix(rawURL, "//duckduckgo.com/l/?uddg=") {
		if idx := strings.Index(rawURL, "uddg="); idx != -1 {
			encoded := rawURL[idx+5:]
			if ampIdx := strings.Index(encoded, "&"); ampIdx != -1 {
				encoded = encoded[:ampIdx]
			}
			if decoded, err := url.QueryUnescape(encoded); err == nil {
				return decoded
			}
		}
	}
	return rawURL
}

// formatSearchResults 格式化搜索结果为人类可读的字符串
// results: 搜索结果列表
// 返回格式化的搜索结果字符串
func formatSearchResults(results []SearchResult) string {
	if len(results) == 0 {
		return "未找到结果。请尝试重新表述您的搜索。"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("找到 %d 个搜索结果:\n\n", len(results)))
	for _, result := range results {
		sb.WriteString(fmt.Sprintf("%d. %s\n", result.Position, result.Title))
		sb.WriteString(fmt.Sprintf("   URL: %s\n", result.Link))
		sb.WriteString(fmt.Sprintf("   摘要: %s\n\n", result.Snippet))
	}
	return sb.String()
}

var (
	lastSearchMu   sync.Mutex    // 保护lastSearchTime的互斥锁
	lastSearchTime time.Time     // 上次搜索的时间
)

// maybeDelaySearch 如果上次搜索是最近的，则添加随机延迟
func maybeDelaySearch() {
	lastSearchMu.Lock()
	defer lastSearchMu.Unlock()

	minGap := time.Duration(500+rand.IntN(1500)) * time.Millisecond
	elapsed := time.Since(lastSearchTime)
	if elapsed < minGap {
		time.Sleep(minGap - elapsed)
	}
	lastSearchTime = time.Now()
}
