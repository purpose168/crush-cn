package config

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	"github.com/stretchr/testify/require"
)

// mockHyperClient 是 Hyper 客户端的模拟实现,用于测试
type mockHyperClient struct {
	provider  catwalk.Provider // 提供者配置
	err       error            // 返回的错误
	callCount int              // 调用次数计数器
}

// Get 模拟从远程获取提供者配置的方法
// 参数:
//   - ctx: 上下文对象,用于控制请求生命周期
//   - etag: 实体标签,用于缓存验证
// 返回:
//   - catwalk.Provider: 提供者配置
//   - error: 错误信息
func (m *mockHyperClient) Get(ctx context.Context, etag string) (catwalk.Provider, error) {
	m.callCount++ // 增加调用计数
	return m.provider, m.err
}

// TestHyperSync_Init 测试 hyperSync 的初始化功能
// 验证初始化后是否正确设置了客户端、缓存路径和初始化标志
func TestHyperSync_Init(t *testing.T) {
	t.Parallel() // 并行运行测试

	syncer := &hyperSync{}
	client := &mockHyperClient{}
	path := "/tmp/hyper.json"

	syncer.Init(client, path, true)

	require.True(t, syncer.init.Load())      // 验证初始化标志已设置
	require.Equal(t, client, syncer.client) // 验证客户端已正确设置
	require.Equal(t, path, syncer.cache.path) // 验证缓存路径已正确设置
}

// TestHyperSync_GetPanicIfNotInit 测试在未初始化的情况下调用 Get 方法时的行为
// 验证如果未调用 Init 方法,直接调用 Get 会触发 panic
func TestHyperSync_GetPanicIfNotInit(t *testing.T) {
	t.Parallel() // 并行运行测试

	syncer := &hyperSync{}
	require.Panics(t, func() { // 验证调用 Get 会触发 panic
		_, _ = syncer.Get(t.Context())
	})
}

// TestHyperSync_GetFreshProvider 测试获取新的提供者配置
// 验证能够成功从远程获取提供者配置并写入缓存
func TestHyperSync_GetFreshProvider(t *testing.T) {
	t.Parallel() // 并行运行测试

	syncer := &hyperSync{}
	client := &mockHyperClient{
		provider: catwalk.Provider{
			Name: "Hyper",
			ID:   "hyper",
			Models: []catwalk.Model{
				{ID: "model-1", Name: "Model 1"},
			},
		},
	}
	path := t.TempDir() + "/hyper.json"

	syncer.Init(client, path, true)

	provider, err := syncer.Get(t.Context())
	require.NoError(t, err)                  // 验证没有错误
	require.Equal(t, "Hyper", provider.Name) // 验证提供者名称正确
	require.Equal(t, 1, client.callCount)    // 验证客户端被调用一次

	// 验证缓存文件已写入
	fileInfo, err := os.Stat(path)
	require.NoError(t, err)           // 验证文件信息获取成功
	require.False(t, fileInfo.IsDir()) // 验证是文件而非目录
}

// TestHyperSync_GetNotModifiedUsesCached 测试当服务器返回未修改状态时使用缓存
// 验证当远程服务器返回 ErrNotModified 错误时,会使用本地缓存的提供者配置
func TestHyperSync_GetNotModifiedUsesCached(t *testing.T) {
	t.Parallel() // 并行运行测试

	tmpDir := t.TempDir()
	path := tmpDir + "/hyper.json"

	// 创建缓存文件
	cachedProvider := catwalk.Provider{
		Name: "Cached Hyper",
		ID:   "hyper",
	}
	data, err := json.Marshal(cachedProvider)
	require.NoError(t, err) // 验证序列化成功
	require.NoError(t, os.WriteFile(path, data, 0o644)) // 验证文件写入成功

	syncer := &hyperSync{}
	client := &mockHyperClient{
		err: catwalk.ErrNotModified, // 模拟服务器返回未修改状态
	}

	syncer.Init(client, path, true)

	provider, err := syncer.Get(t.Context())
	require.NoError(t, err)                    // 验证没有错误
	require.Equal(t, "Cached Hyper", provider.Name) // 验证使用的是缓存的提供者
	require.Equal(t, 1, client.callCount)      // 验证客户端被调用一次
}

// TestHyperSync_GetClientError 测试客户端发生错误时的回退机制
// 验证当客户端返回错误时,系统会回退使用内置的提供者配置
func TestHyperSync_GetClientError(t *testing.T) {
	t.Parallel() // 并行运行测试

	tmpDir := t.TempDir()
	path := tmpDir + "/hyper.json"

	syncer := &hyperSync{}
	client := &mockHyperClient{
		err: errors.New("network error"), // 模拟网络错误
	}

	syncer.Init(client, path, true)

	provider, err := syncer.Get(t.Context())
	require.NoError(t, err) // 应该回退使用内置配置,不返回错误
	require.Equal(t, "Charm Hyper", provider.Name) // 验证使用的是内置的 Charm Hyper 提供者
	require.Equal(t, catwalk.InferenceProvider("hyper"), provider.ID) // 验证提供者 ID 正确
}

// TestHyperSync_GetEmptyCache 测试缓存为空时获取新的提供者配置
// 验证当缓存文件不存在时,能够成功从远程获取提供者配置
func TestHyperSync_GetEmptyCache(t *testing.T) {
	t.Parallel() // 并行运行测试

	tmpDir := t.TempDir()
	path := tmpDir + "/hyper.json"

	syncer := &hyperSync{}
	client := &mockHyperClient{
		provider: catwalk.Provider{
			Name: "Fresh Hyper",
			ID:   "hyper",
			Models: []catwalk.Model{
				{ID: "model-1", Name: "Model 1"},
			},
		},
	}

	syncer.Init(client, path, true)

	provider, err := syncer.Get(t.Context())
	require.NoError(t, err) // 验证没有错误
	require.Equal(t, "Fresh Hyper", provider.Name) // 验证获取到新的提供者配置
}

// TestHyperSync_GetCalledMultipleTimesUsesOnce 测试多次调用 Get 方法时客户端只被调用一次
// 验证由于使用了 sync.Once,多次调用 Get 方法时客户端只会被调用一次
func TestHyperSync_GetCalledMultipleTimesUsesOnce(t *testing.T) {
	t.Parallel() // 并行运行测试

	syncer := &hyperSync{}
	client := &mockHyperClient{
		provider: catwalk.Provider{
			Name: "Hyper",
			ID:   "hyper",
			Models: []catwalk.Model{
				{ID: "model-1", Name: "Model 1"},
			},
		},
	}
	path := t.TempDir() + "/hyper.json"

	syncer.Init(client, path, true)

	// 多次调用 Get 方法
	provider1, err1 := syncer.Get(t.Context())
	require.NoError(t, err1) // 验证第一次调用没有错误
	require.Equal(t, "Hyper", provider1.Name) // 验证第一次获取的提供者名称正确

	provider2, err2 := syncer.Get(t.Context())
	require.NoError(t, err2) // 验证第二次调用没有错误
	require.Equal(t, "Hyper", provider2.Name) // 验证第二次获取的提供者名称正确

	// 由于使用了 sync.Once,客户端应该只被调用一次
	require.Equal(t, 1, client.callCount) // 验证客户端只被调用一次
}

// TestHyperSync_GetCacheStoreError 测试缓存存储错误时的处理
// 验证当无法创建缓存目录时,系统会返回错误但仍能提供提供者配置
func TestHyperSync_GetCacheStoreError(t *testing.T) {
	t.Parallel() // 并行运行测试

	// 在需要目录的位置创建一个文件,导致 mkdir 失败
	tmpDir := t.TempDir()
	blockingFile := tmpDir + "/blocking"
	require.NoError(t, os.WriteFile(blockingFile, []byte("block"), 0o644)) // 创建阻塞文件

	// 尝试在阻塞文件下的子目录中创建缓存
	path := blockingFile + "/subdir/hyper.json"

	syncer := &hyperSync{}
	client := &mockHyperClient{
		provider: catwalk.Provider{
			Name: "Hyper",
			ID:   "hyper",
			Models: []catwalk.Model{
				{ID: "model-1", Name: "Model 1"},
			},
		},
	}

	syncer.Init(client, path, true)

	provider, err := syncer.Get(t.Context())
	require.Error(t, err) // 验证返回错误
	require.Contains(t, err.Error(), "failed to create directory for provider cache") // 验证错误信息包含创建目录失败的内容
	require.Equal(t, "Hyper", provider.Name) // 提供者仍然会被返回
}
