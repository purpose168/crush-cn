package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	// githubApiUrl GitHub API URL，用于获取最新发布版本
	githubApiUrl = "https://api.github.com/repos/purpose168/crush-cn/releases/latest"
	// userAgent HTTP 请求的用户代理
	userAgent = "crush/1.0"
)

// Default 是默认的更新客户端
var Default Client = &github{}

// Info 包含可用更新的信息

type Info struct {
	Current string // 当前版本
	Latest  string // 最新版本
	URL     string // 更新链接
}

// goInstallRegexp 匹配类似这样的版本字符串：
// v0.0.0-0.20251231235959-06c807842604
var goInstallRegexp = regexp.MustCompile(`^v?\d+\.\d+\.\d+-\d+\.\d{14}-[0-9a-f]{12}$`)

// IsDevelopment 判断当前版本是否为开发版本
// 如果版本是 "devel"、"unknown"，或包含 "dirty"，或匹配 go install 生成的版本格式，则认为是开发版本
func (i Info) IsDevelopment() bool {
	return i.Current == "devel" || i.Current == "unknown" || strings.Contains(i.Current, "dirty") || goInstallRegexp.MatchString(i.Current)
}

// Available 判断是否有可用更新
//
// 如果当前版本和最新版本都是稳定版本，当版本不同时返回 true
// 如果当前版本是预发布版本而最新版本不是，返回 true
// 如果最新版本是预发布版本而当前版本不是，返回 false
func (i Info) Available() bool {
	// 检查当前版本是否为预发布版本（包含 "-"）
	cpr := strings.Contains(i.Current, "-")
	// 检查最新版本是否为预发布版本（包含 "-"）
	lpr := strings.Contains(i.Latest, "-")

	// 当前版本是预发布版本，最新版本不是预发布版本
	if cpr && !lpr {
		return true
	}

	// 最新版本是预发布版本，当前版本不是预发布版本
	if lpr && !cpr {
		return false
	}

	// 其他情况，只要版本不同就认为有更新
	return i.Current != i.Latest
}

// Check 检查是否有新版本可用
func Check(ctx context.Context, current string, client Client) (Info, error) {
	// 初始化 Info 结构体，默认最新版本为当前版本
	info := Info{
		Current: current,
		Latest:  current,
	}

	// 从客户端获取最新发布版本
	release, err := client.Latest(ctx)
	if err != nil {
		return info, fmt.Errorf("failed to fetch latest release: %w", err)
	}

	// 处理版本号，移除前缀 "v"
	info.Latest = strings.TrimPrefix(release.TagName, "v")
	info.Current = strings.TrimPrefix(info.Current, "v")
	// 设置更新链接
	info.URL = release.HTMLURL
	return info, nil
}

// Release 表示 GitHub 发布版本

type Release struct {
	TagName string `json:"tag_name"` // 版本标签
	HTMLURL string `json:"html_url"` // 发布页面 URL
}

// Client 是一个可以获取最新发布版本的客户端接口

type Client interface {
	// Latest 获取最新发布版本
	Latest(ctx context.Context) (*Release, error)
}

// github 是 Client 接口的 GitHub 实现

type github struct{}

// Latest 实现 Client 接口，从 GitHub API 获取最新发布版本
func (c *github) Latest(ctx context.Context) (*Release, error) {
	// 创建 HTTP 客户端，设置 30 秒超时
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "GET", githubApiUrl, nil)
	if err != nil {
		return nil, err
	}

	// 设置请求头
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解码响应体为 Release 结构体
	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}
