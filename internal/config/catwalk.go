package config

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"

	"charm.land/catwalk/pkg/catwalk"
	"charm.land/catwalk/pkg/embedded"
)

// catwalkClient 定义了 Catwalk 客户端接口，用于获取提供商信息
type catwalkClient interface {
	GetProviders(context.Context, string) ([]catwalk.Provider, error)
}

// 确保 catwalkSync 实现了 syncer 接口
var _ syncer[[]catwalk.Provider] = (*catwalkSync)(nil)

// catwalkSync 管理 Catwalk 提供商信息的同步，支持缓存和自动更新
type catwalkSync struct {
	once       sync.Once         // 确保初始化只执行一次
	result     []catwalk.Provider // 存储提供商结果
	cache      cache[[]catwalk.Provider] // 缓存接口
	client     catwalkClient     // Catwalk 客户端
	autoupdate bool              // 是否启用自动更新
	init       atomic.Bool       // 初始化状态标志
}

// Init 初始化 catwalkSync 实例
// client: Catwalk 客户端实例
// path: 缓存文件路径
// autoupdate: 是否启用自动更新
func (s *catwalkSync) Init(client catwalkClient, path string, autoupdate bool) {
	s.client = client
	s.cache = newCache[[]catwalk.Provider](path)
	s.autoupdate = autoupdate
	s.init.Store(true)
}

// Get 获取 Catwalk 提供商信息
// ctx: 上下文对象，用于控制请求超时和取消
// 返回: 提供商列表和可能的错误
func (s *catwalkSync) Get(ctx context.Context) ([]catwalk.Provider, error) {
	if !s.init.Load() {
		panic("在 Init 之前调用了 Get 方法")
	}

	var throwErr error
	s.once.Do(func() {
		if !s.autoupdate {
			slog.Info("使用嵌入的 Catwalk 提供商")
			s.result = embedded.GetAll()
			return
		}

		cached, etag, cachedErr := s.cache.Get()
		if len(cached) == 0 || cachedErr != nil {
			// 如果缓存文件为空，默认使用嵌入的提供商
			cached = embedded.GetAll()
		}

		slog.Info("从 Catwalk 获取提供商")
		result, err := s.client.GetProviders(ctx, etag)
		if errors.Is(err, context.DeadlineExceeded) {
			slog.Warn("Catwalk 提供商未及时更新")
			s.result = cached
			return
		}
		if errors.Is(err, catwalk.ErrNotModified) {
			slog.Info("Catwalk 提供商未修改")
			s.result = cached
			return
		}
		if err != nil {
			// 出错时回退到缓存（如果缓存为空则使用嵌入的提供商）
			s.result = cached
			return
		}
		if len(result) == 0 {
			s.result = cached
			throwErr = errors.New("从 catwalk 获取的提供商列表为空")
			return
		}

		s.result = result
		throwErr = s.cache.Store(result)
	})
	return s.result, throwErr
}
