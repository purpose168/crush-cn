package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"charm.land/catwalk/pkg/catwalk"
	xetag "github.com/charmbracelet/x/etag"
	"github.com/purpose168/crush-cn/internal/agent/hyper"
)

// hyperClient 定义 Hyper 客户端接口，用于获取提供商信息
type hyperClient interface {
	Get(context.Context, string) (catwalk.Provider, error)
}

// 确保 hyperSync 实现了 syncer 接口
var _ syncer[catwalk.Provider] = (*hyperSync)(nil)

// hyperSync 负责 Hyper 提供商信息的同步和缓存管理
type hyperSync struct {
	once       sync.Once         // 确保初始化只执行一次
	result     catwalk.Provider  // 存储获取到的提供商信息
	cache      cache[catwalk.Provider] // 缓存管理器
	client     hyperClient       // Hyper 客户端
	autoupdate bool              // 是否启用自动更新
	init       atomic.Bool       // 标记是否已初始化
}

// Init 初始化 hyperSync，设置客户端、缓存路径和自动更新选项
func (s *hyperSync) Init(client hyperClient, path string, autoupdate bool) {
	s.client = client
	s.cache = newCache[catwalk.Provider](path)
	s.autoupdate = autoupdate
	s.init.Store(true)
}

// Get 获取 Hyper 提供商信息，支持缓存和自动更新
// 如果启用自动更新，会尝试从远程获取最新数据；否则使用内置提供商
func (s *hyperSync) Get(ctx context.Context) (catwalk.Provider, error) {
	if !s.init.Load() {
		panic("在初始化之前调用了 Get 方法")
	}

	var throwErr error
	s.once.Do(func() {
		if !s.autoupdate {
			slog.Info("使用内置的 Hyper 提供商")
			s.result = hyper.Embedded()
			return
		}

		cached, etag, cachedErr := s.cache.Get()
		if cached.ID == "" || cachedErr != nil {
			// 如果缓存文件为空，则默认使用内置提供商
			cached = hyper.Embedded()
		}

		slog.Info("正在获取 Hyper 提供商")
		result, err := s.client.Get(ctx, etag)
		if errors.Is(err, context.DeadlineExceeded) {
			slog.Warn("Hyper 提供商未及时更新")
			s.result = cached
			return
		}
		if errors.Is(err, catwalk.ErrNotModified) {
			slog.Info("Hyper 提供商未修改")
			s.result = cached
			return
		}
		if len(result.Models) == 0 {
			slog.Warn("Hyper 未返回任何模型")
			s.result = cached
			return
		}

		s.result = result
		throwErr = s.cache.Store(result)
	})
	return s.result, throwErr
}

// 确保 realHyperClient 实现了 hyperClient 接口
var _ hyperClient = realHyperClient{}

// realHyperClient 是 Hyper API 的真实客户端实现
type realHyperClient struct {
	baseURL string // Hyper API 的基础 URL
}

// Get 实现 hyperClient 接口
// 从 Hyper API 获取提供商信息，支持 ETag 缓存验证
func (r realHyperClient) Get(ctx context.Context, etag string) (catwalk.Provider, error) {
	var result catwalk.Provider
	// 创建带有上下文的 HTTP GET 请求
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		r.baseURL+"/api/v1/provider",
		nil,
	)
	if err != nil {
		return result, fmt.Errorf("无法创建请求: %w", err)
	}
	// 添加 ETag 请求头用于缓存验证
	xetag.Request(req, etag)

	// 创建 HTTP 客户端，设置 30 秒超时
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return result, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	// 检查响应状态码
	if resp.StatusCode == http.StatusNotModified {
		return result, catwalk.ErrNotModified
	}

	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("意外的状态码: %d", resp.StatusCode)
	}

	// 解码 JSON 响应体
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return result, fmt.Errorf("解码响应失败: %w", err)
	}

	return result, nil
}
