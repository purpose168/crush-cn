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

// mockCatwalkClient 是 catwalk 客户端的模拟实现，用于测试
// 包含提供者列表、错误信息和调用计数
type mockCatwalkClient struct {
	providers []catwalk.Provider // 提供者列表
	err       error              // 错误信息
	callCount int                // 调用次数计数器
}

// GetProviders 获取提供者列表的模拟方法
// 增加调用计数并返回预设的提供者列表和错误
func (m *mockCatwalkClient) GetProviders(ctx context.Context, etag string) ([]catwalk.Provider, error) {
	m.callCount++ // 增加调用计数
	return m.providers, m.err
}

// TestCatwalkSync_Init 测试 catwalkSync 的初始化功能
// 验证初始化后各个字段是否正确设置
func TestCatwalkSync_Init(t *testing.T) {
	t.Parallel() // 并行运行测试

	syncer := &catwalkSync{} // 创建同步器实例
	client := &mockCatwalkClient{} // 创建模拟客户端
	path := "/tmp/test.json" // 设置缓存文件路径

	syncer.Init(client, path, true) // 初始化同步器，启用自动更新

	require.True(t, syncer.init.Load()) // 验证初始化标志已设置
	require.Equal(t, client, syncer.client) // 验证客户端已正确设置
	require.Equal(t, path, syncer.cache.path) // 验证缓存路径已正确设置
	require.True(t, syncer.autoupdate) // 验证自动更新标志已启用
}

// TestCatwalkSync_GetPanicIfNotInit 测试未初始化时调用 Get 方法是否会触发 panic
// 验证在未初始化状态下调用 Get 方法会导致程序崩溃
func TestCatwalkSync_GetPanicIfNotInit(t *testing.T) {
	t.Parallel() // 并行运行测试

	syncer := &catwalkSync{} // 创建未初始化的同步器实例
	require.Panics(t, func() { // 验证调用会触发 panic
		_, _ = syncer.Get(t.Context()) // 尝试获取提供者
	})
}

// TestCatwalkSync_GetWithAutoUpdateDisabled 测试禁用自动更新时的 Get 方法行为
// 验证当自动更新被禁用时，系统会使用内置提供者而不调用客户端
func TestCatwalkSync_GetWithAutoUpdateDisabled(t *testing.T) {
	t.Parallel() // 并行运行测试

	syncer := &catwalkSync{} // 创建同步器实例
	client := &mockCatwalkClient{
		providers: []catwalk.Provider{{Name: "should-not-be-used"}}, // 设置不应该被使用的提供者
	}
	path := t.TempDir() + "/providers.json" // 设置缓存文件路径

	syncer.Init(client, path, false) // 初始化同步器，禁用自动更新

	providers, err := syncer.Get(t.Context()) // 获取提供者列表
	require.NoError(t, err) // 验证没有错误发生
	require.NotEmpty(t, providers) // 验证提供者列表不为空
	require.Equal(t, 0, client.callCount, "Client should not be called when autoupdate is disabled") // 验证客户端未被调用

	// Should return embedded providers.
	// 应该返回内置的提供者
	for _, p := range providers {
		require.NotEqual(t, "should-not-be-used", p.Name) // 验证返回的不是客户端中的提供者
	}
}

// TestCatwalkSync_GetFreshProviders 测试获取最新的提供者列表
// 验证系统能够从客户端获取最新的提供者并写入缓存
func TestCatwalkSync_GetFreshProviders(t *testing.T) {
	t.Parallel() // 并行运行测试

	syncer := &catwalkSync{} // 创建同步器实例
	client := &mockCatwalkClient{
		providers: []catwalk.Provider{
			{Name: "Fresh Provider", ID: "fresh"}, // 设置最新的提供者
		},
	}
	path := t.TempDir() + "/providers.json" // 设置缓存文件路径

	syncer.Init(client, path, true) // 初始化同步器，启用自动更新

	providers, err := syncer.Get(t.Context()) // 获取提供者列表
	require.NoError(t, err) // 验证没有错误发生
	require.Len(t, providers, 1) // 验证返回了一个提供者
	require.Equal(t, "Fresh Provider", providers[0].Name) // 验证提供者名称正确
	require.Equal(t, 1, client.callCount) // 验证客户端被调用了一次

	// Verify cache was written.
	// 验证缓存文件已被写入
	fileInfo, err := os.Stat(path) // 获取文件信息
	require.NoError(t, err) // 验证没有错误发生
	require.False(t, fileInfo.IsDir()) // 验证是文件而不是目录
}

// TestCatwalkSync_GetNotModifiedUsesCached 测试当内容未修改时使用缓存
// 验证当客户端返回 ErrNotModified 错误时，系统会使用缓存的提供者列表
func TestCatwalkSync_GetNotModifiedUsesCached(t *testing.T) {
	t.Parallel() // 并行运行测试

	tmpDir := t.TempDir() // 创建临时目录
	path := tmpDir + "/providers.json" // 设置缓存文件路径

	// Create cache file.
	// 创建缓存文件
	cachedProviders := []catwalk.Provider{
		{Name: "Cached Provider", ID: "cached"}, // 设置缓存的提供者
	}
	data, err := json.Marshal(cachedProviders) // 序列化为 JSON
	require.NoError(t, err) // 验证序列化成功
	require.NoError(t, os.WriteFile(path, data, 0o644)) // 写入缓存文件

	syncer := &catwalkSync{} // 创建同步器实例
	client := &mockCatwalkClient{
		err: catwalk.ErrNotModified, // 设置未修改错误
	}

	syncer.Init(client, path, true) // 初始化同步器，启用自动更新

	providers, err := syncer.Get(t.Context()) // 获取提供者列表
	require.NoError(t, err) // 验证没有错误发生
	require.Len(t, providers, 1) // 验证返回了一个提供者
	require.Equal(t, "Cached Provider", providers[0].Name) // 验证使用的是缓存的提供者
	require.Equal(t, 1, client.callCount) // 验证客户端被调用了一次
}

// TestCatwalkSync_GetEmptyResultFallbackToCached 测试当客户端返回空结果时回退到缓存
// 验证当客户端返回空的提供者列表时，系统会使用缓存的提供者列表
func TestCatwalkSync_GetEmptyResultFallbackToCached(t *testing.T) {
	t.Parallel() // 并行运行测试

	tmpDir := t.TempDir() // 创建临时目录
	path := tmpDir + "/providers.json" // 设置缓存文件路径

	// Create cache file.
	// 创建缓存文件
	cachedProviders := []catwalk.Provider{
		{Name: "Cached Provider", ID: "cached"}, // 设置缓存的提供者
	}
	data, err := json.Marshal(cachedProviders) // 序列化为 JSON
	require.NoError(t, err) // 验证序列化成功
	require.NoError(t, os.WriteFile(path, data, 0o644)) // 写入缓存文件

	syncer := &catwalkSync{} // 创建同步器实例
	client := &mockCatwalkClient{
		providers: []catwalk.Provider{}, // Empty result. // 空结果
	}

	syncer.Init(client, path, true) // 初始化同步器，启用自动更新

	providers, err := syncer.Get(t.Context()) // 获取提供者列表
	require.Error(t, err) // 验证有错误发生
	require.Contains(t, err.Error(), "empty providers list from catwalk") // 验证错误消息包含特定内容
	require.Len(t, providers, 1) // 验证返回了一个提供者
	require.Equal(t, "Cached Provider", providers[0].Name) // 验证使用的是缓存的提供者
}

// TestCatwalkSync_GetEmptyCacheDefaultsToEmbedded 测试当缓存为空时使用内置提供者
// 验证当缓存文件为空且客户端返回错误时，系统会使用内置的提供者列表
func TestCatwalkSync_GetEmptyCacheDefaultsToEmbedded(t *testing.T) {
	t.Parallel() // 并行运行测试

	tmpDir := t.TempDir() // 创建临时目录
	path := tmpDir + "/providers.json" // 设置缓存文件路径

	// Create empty cache file.
	// 创建空的缓存文件
	emptyProviders := []catwalk.Provider{} // 空的提供者列表
	data, err := json.Marshal(emptyProviders) // 序列化为 JSON
	require.NoError(t, err) // 验证序列化成功
	require.NoError(t, os.WriteFile(path, data, 0o644)) // 写入缓存文件

	syncer := &catwalkSync{} // 创建同步器实例
	client := &mockCatwalkClient{
		err: errors.New("network error"), // 设置网络错误
	}

	syncer.Init(client, path, true) // 初始化同步器，启用自动更新

	providers, err := syncer.Get(t.Context()) // 获取提供者列表
	require.NoError(t, err) // 验证没有错误发生
	require.NotEmpty(t, providers, "Should fall back to embedded providers") // 验证提供者列表不为空，回退到内置提供者

	// Verify it's embedded providers by checking we have multiple common ones.
	// 通过检查是否有多个常见提供者来验证是内置提供者
	require.Greater(t, len(providers), 5) // 验证提供者数量大于 5
}

// TestCatwalkSync_GetClientError 测试客户端错误时的处理
// 验证当客户端返回错误时，系统会回退到内置提供者列表
func TestCatwalkSync_GetClientError(t *testing.T) {
	t.Parallel() // 并行运行测试

	tmpDir := t.TempDir() // 创建临时目录
	path := tmpDir + "/providers.json" // 设置缓存文件路径

	syncer := &catwalkSync{} // 创建同步器实例
	client := &mockCatwalkClient{
		err: errors.New("network error"), // 设置网络错误
	}

	syncer.Init(client, path, true) // 初始化同步器，启用自动更新

	providers, err := syncer.Get(t.Context()) // 获取提供者列表
	require.NoError(t, err) // Should fall back to embedded. // 验证没有错误发生，回退到内置提供者
	require.NotEmpty(t, providers) // 验证提供者列表不为空
}

// TestCatwalkSync_GetCalledMultipleTimesUsesOnce 测试多次调用 Get 方法的优化
// 验证由于 sync.Once 的存在，多次调用 Get 方法只会调用客户端一次
func TestCatwalkSync_GetCalledMultipleTimesUsesOnce(t *testing.T) {
	t.Parallel() // 并行运行测试

	syncer := &catwalkSync{} // 创建同步器实例
	client := &mockCatwalkClient{
		providers: []catwalk.Provider{
			{Name: "Provider", ID: "test"}, // 设置测试提供者
		},
	}
	path := t.TempDir() + "/providers.json" // 设置缓存文件路径

	syncer.Init(client, path, true) // 初始化同步器，启用自动更新

	// Call Get multiple times.
	// 多次调用 Get 方法
	providers1, err1 := syncer.Get(t.Context()) // 第一次调用
	require.NoError(t, err1) // 验证没有错误发生
	require.Len(t, providers1, 1) // 验证返回了一个提供者

	providers2, err2 := syncer.Get(t.Context()) // 第二次调用
	require.NoError(t, err2) // 验证没有错误发生
	require.Len(t, providers2, 1) // 验证返回了一个提供者

	// Client should only be called once due to sync.Once.
	// 由于 sync.Once 的存在，客户端应该只被调用一次
	require.Equal(t, 1, client.callCount) // 验证客户端只被调用了一次
}
