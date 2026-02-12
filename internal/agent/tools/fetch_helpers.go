package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"golang.org/x/net/html"
)

// BrowserUserAgent 是一个逼真的浏览器用户代理，用于更好的兼容性
const BrowserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

var multipleNewlinesRe = regexp.MustCompile(`\n{3,}`)

// FetchURLAndConvert 抓取URL并将HTML内容转换为markdown格式
// ctx: 上下文对象
// client: HTTP客户端
// url: 要抓取的URL
// 返回处理后的内容
func FetchURLAndConvert(ctx context.Context, client *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	// 使用逼真的浏览器头信息以获得更好的兼容性
	req.Header.Set("User-Agent", BrowserUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("抓取URL失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
	}

	maxSize := int64(5 * 1024 * 1024) // 5MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
	if err != nil {
		return "", fmt.Errorf("读取响应体失败: %w", err)
	}

	content := string(body)

	if !utf8.ValidString(content) {
		return "", errors.New("响应内容不是有效的UTF-8")
	}

	contentType := resp.Header.Get("Content-Type")

	// 将HTML转换为markdown以获得更好的AI处理效果
	if strings.Contains(contentType, "text/html") {
		// 在转换前移除噪声元素
		cleanedHTML := removeNoisyElements(content)
		markdown, err := ConvertHTMLToMarkdown(cleanedHTML)
		if err != nil {
			return "", fmt.Errorf("将HTML转换为markdown失败: %w", err)
		}
		content = cleanupMarkdown(markdown)
	} else if strings.Contains(contentType, "application/json") || strings.Contains(contentType, "text/json") {
		// 格式化JSON以提高可读性
		formatted, err := FormatJSON(content)
		if err == nil {
			content = formatted
		}
		// 如果格式化失败，保留原始内容
	}

	return content, nil
}

// removeNoisyElements 从HTML中移除script、style、nav、header、footer等噪声元素，以改善内容提取
// htmlContent: HTML内容
// 返回清理后的HTML内容
func removeNoisyElements(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		// 如果解析失败，返回原始内容
		return htmlContent
	}

	// 要完全移除的元素
	noisyTags := map[string]bool{
		"script":   true,
		"style":    true,
		"nav":      true,
		"header":   true,
		"footer":   true,
		"aside":    true,
		"noscript": true,
		"iframe":   true,
		"svg":      true,
	}

	var removeNodes func(*html.Node)
	removeNodes = func(n *html.Node) {
		var toRemove []*html.Node

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && noisyTags[c.Data] {
				toRemove = append(toRemove, c)
			} else {
				removeNodes(c)
			}
		}

		for _, node := range toRemove {
			n.RemoveChild(node)
		}
	}

	removeNodes(doc)

	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return htmlContent
	}

	return buf.String()
}

// cleanupMarkdown 从markdown中移除过多的空白字符和空行
// content: markdown内容
// 返回清理后的markdown内容
func cleanupMarkdown(content string) string {
	// 将多个空行折叠为最多两个
	content = multipleNewlinesRe.ReplaceAllString(content, "\n\n")

	// 移除每行末尾的空白字符
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	content = strings.Join(lines, "\n")

	// 修剪开头和结尾的空白字符
	content = strings.TrimSpace(content)

	return content
}

// ConvertHTMLToMarkdown 将HTML内容转换为markdown格式
// htmlContent: HTML内容
// 返回转换后的markdown内容
func ConvertHTMLToMarkdown(htmlContent string) (string, error) {
	converter := md.NewConverter("", true, nil)

	markdown, err := converter.ConvertString(htmlContent)
	if err != nil {
		return "", err
	}

	return markdown, nil
}

// FormatJSON 用适当的缩进格式化JSON内容
// content: JSON内容
// 返回格式化后的JSON内容
func FormatJSON(content string) (string, error) {
	var data any
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return "", err
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
