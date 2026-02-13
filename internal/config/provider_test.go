package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	"github.com/stretchr/testify/require"
)

// 重置提供者状态，清除全局变量
func resetProviderState() {
	providerOnce = sync.Once{}
	providerList = nil
	providerErr = nil
	catwalkSyncer = &catwalkSync{}
	hyperSyncer = &hyperSync{}
}

// 测试提供者集成功能，禁用自动更新
func TestProviders_Integration_AutoUpdateDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	// 使用测试专用实例以避免全局状态干扰
	testCatwalkSyncer := &catwalkSync{}
	testHyperSyncer := &hyperSync{}

	originalCatwalSyncer := catwalkSyncer
	originalHyperSyncer := hyperSyncer
	defer func() {
		catwalkSyncer = originalCatwalSyncer
		hyperSyncer = originalHyperSyncer
	}()

	catwalkSyncer = testCatwalkSyncer
	hyperSyncer = testHyperSyncer

	resetProviderState()
	defer resetProviderState()

	cfg := &Config{
		Options: &Options{
			DisableProviderAutoUpdate: true,
		},
	}

	providers, err := Providers(cfg)
	require.NoError(t, err)
	require.NotNil(t, providers)
	require.Greater(t, len(providers), 5, "期望内置提供者")
}

// 测试提供者集成功能，使用模拟客户端
func TestProviders_Integration_WithMockClients(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	// 为此测试创建新的同步器
	testCatwalkSyncer := &catwalkSync{}
	testHyperSyncer := &hyperSync{}

	// 使用模拟客户端初始化
	mockCatwalkClient := &mockCatwalkClient{
		providers: []catwalk.Provider{
			{Name: "Provider1", ID: "p1"},
			{Name: "Provider2", ID: "p2"},
		},
	}
	mockHyperClient := &mockHyperClient{
		provider: catwalk.Provider{
			Name: "Hyper",
			ID:   "hyper",
			Models: []catwalk.Model{
				{ID: "hyper-1", Name: "Hyper Model"},
			},
		},
	}

	catwalkPath := tmpDir + "/crush/providers.json"
	hyperPath := tmpDir + "/crush/hyper.json"

	testCatwalkSyncer.Init(mockCatwalkClient, catwalkPath, true)
	testHyperSyncer.Init(mockHyperClient, hyperPath, true)

	// 从每个同步器获取提供者
	catwalkProviders, err := testCatwalkSyncer.Get(t.Context())
	require.NoError(t, err)
	require.Len(t, catwalkProviders, 2)

	hyperProvider, err := testHyperSyncer.Get(t.Context())
	require.NoError(t, err)
	require.Equal(t, "Hyper", hyperProvider.Name)

	// 验证总数
	allProviders := append(catwalkProviders, hyperProvider)
	require.Len(t, allProviders, 3)
}

// 测试提供者集成功能，使用缓存数据
func TestProviders_Integration_WithCachedData(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	// 创建缓存文件
	catwalkPath := tmpDir + "/crush/providers.json"
	hyperPath := tmpDir + "/crush/hyper.json"

	require.NoError(t, os.MkdirAll(tmpDir+"/crush", 0o755))

	// 写入 Catwalk 缓存
	catwalkProviders := []catwalk.Provider{
		{Name: "Cached1", ID: "c1"},
		{Name: "Cached2", ID: "c2"},
	}
	data, err := json.Marshal(catwalkProviders)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(catwalkPath, data, 0o644))

	// 写入 Hyper 缓存
	hyperProvider := catwalk.Provider{
		Name: "Cached Hyper",
		ID:   "hyper",
	}
	data, err = json.Marshal(hyperProvider)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(hyperPath, data, 0o644))

	// 创建新的同步器
	testCatwalkSyncer := &catwalkSync{}
	testHyperSyncer := &hyperSync{}

	// 返回 ErrNotModified 的模拟客户端
	mockCatwalkClient := &mockCatwalkClient{
		err: catwalk.ErrNotModified,
	}
	mockHyperClient := &mockHyperClient{
		err: catwalk.ErrNotModified,
	}

	testCatwalkSyncer.Init(mockCatwalkClient, catwalkPath, true)
	testHyperSyncer.Init(mockHyperClient, hyperPath, true)

	// 获取提供者 - 应该使用缓存
	catwalkResult, err := testCatwalkSyncer.Get(t.Context())
	require.NoError(t, err)
	require.Len(t, catwalkResult, 2)
	require.Equal(t, "Cached1", catwalkResult[0].Name)

	hyperResult, err := testHyperSyncer.Get(t.Context())
	require.NoError(t, err)
	require.Equal(t, "Cached Hyper", hyperResult.Name)
}

// 测试提供者集成功能，Catwalk 失败但 Hyper 成功
func TestProviders_Integration_CatwalkFailsHyperSucceeds(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	testCatwalkSyncer := &catwalkSync{}
	testHyperSyncer := &hyperSync{}

	// Catwalk 失败，Hyper 成功
	mockCatwalkClient := &mockCatwalkClient{
		err: catwalk.ErrNotModified, // 将使用内置提供者
	}
	mockHyperClient := &mockHyperClient{
		provider: catwalk.Provider{
			Name: "Hyper",
			ID:   "hyper",
			Models: []catwalk.Model{
				{ID: "hyper-1", Name: "Hyper Model"},
			},
		},
	}

	catwalkPath := tmpDir + "/crush/providers.json"
	hyperPath := tmpDir + "/crush/hyper.json"

	testCatwalkSyncer.Init(mockCatwalkClient, catwalkPath, true)
	testHyperSyncer.Init(mockHyperClient, hyperPath, true)

	catwalkResult, err := testCatwalkSyncer.Get(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, catwalkResult) // 应该有内置提供者

	hyperResult, err := testHyperSyncer.Get(t.Context())
	require.NoError(t, err)
	require.Equal(t, "Hyper", hyperResult.Name)
}

// 测试提供者集成功能，两者都失败
func TestProviders_Integration_BothFail(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	testCatwalkSyncer := &catwalkSync{}
	testHyperSyncer := &hyperSync{}

	// 两者都失败
	mockCatwalkClient := &mockCatwalkClient{
		err: catwalk.ErrNotModified,
	}
	mockHyperClient := &mockHyperClient{
		provider: catwalk.Provider{}, // 空提供者
	}

	catwalkPath := tmpDir + "/crush/providers.json"
	hyperPath := tmpDir + "/crush/hyper.json"

	testCatwalkSyncer.Init(mockCatwalkClient, catwalkPath, true)
	testHyperSyncer.Init(mockHyperClient, hyperPath, true)

	catwalkResult, err := testCatwalkSyncer.Get(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, catwalkResult) // 应该回退到内置提供者

	hyperResult, err := testHyperSyncer.Get(t.Context())
	require.NoError(t, err)
	require.Equal(t, "Charm Hyper", hyperResult.Name) // 当没有模型时回退到内置提供者
}

// 测试缓存的存储和获取功能
func TestCache_StoreAndGet(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	cachePath := tmpDir + "/test.json"

	cache := newCache[[]catwalk.Provider](cachePath)

	providers := []catwalk.Provider{
		{Name: "Provider1", ID: "p1"},
		{Name: "Provider2", ID: "p2"},
	}

	// 存储
	err := cache.Store(providers)
	require.NoError(t, err)

	// 获取
	result, etag, err := cache.Get()
	require.NoError(t, err)
	require.Len(t, result, 2)
	require.Equal(t, "Provider1", result[0].Name)
	require.NotEmpty(t, etag)
}

// 测试获取不存在的缓存
func TestCache_GetNonExistent(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	cachePath := tmpDir + "/nonexistent.json"

	cache := newCache[[]catwalk.Provider](cachePath)

	_, _, err := cache.Get()
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to read provider cache file")
}

// 测试获取无效的 JSON 缓存
func TestCache_GetInvalidJSON(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	cachePath := tmpDir + "/invalid.json"

	require.NoError(t, os.WriteFile(cachePath, []byte("invalid json"), 0o644))

	cache := newCache[[]catwalk.Provider](cachePath)

	_, _, err := cache.Get()
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to unmarshal provider data from cache")
}

// 测试缓存路径生成功能
func TestCachePathFor(t *testing.T) {
	tests := []struct {
		name        string
		xdgDataHome string
		expected    string
	}{
		{
			name:        "使用 XDG_DATA_HOME",
			xdgDataHome: "/custom/data",
			expected:    "/custom/data/crush/providers.json",
		},
		{
			name:        "不使用 XDG_DATA_HOME",
			xdgDataHome: "",
			expected:    "", // 将使用平台特定的默认值
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.xdgDataHome != "" {
				t.Setenv("XDG_DATA_HOME", tt.xdgDataHome)
			} else {
				t.Setenv("XDG_DATA_HOME", "")
			}

			result := cachePathFor("providers")
			if tt.expected != "" {
				require.Equal(t, tt.expected, filepath.ToSlash(result))
			} else {
				require.Contains(t, result, "crush")
				require.Contains(t, result, "providers.json")
			}
		})
	}
}
