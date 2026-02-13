package config

import (
	"context"
	"os"
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	"github.com/stretchr/testify/require"
)

type emptyProviderClient struct{}

func (m *emptyProviderClient) GetProviders(context.Context, string) ([]catwalk.Provider, error) {
	return []catwalk.Provider{}, nil
}

// TestCatwalkSync_GetEmptyResultFromClient 测试当客户端返回空列表时，
// 我们会回退到缓存的服务提供商并返回错误。
func TestCatwalkSync_GetEmptyResultFromClient(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path := tmpDir + "/providers.json"

	syncer := &catwalkSync{}
	client := &emptyProviderClient{}

	syncer.Init(client, path, true)

	providers, err := syncer.Get(t.Context())
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty providers list from catwalk")
	require.NotEmpty(t, providers) // 应该有嵌入的服务提供商作为回退方案。

	// 检查没有为空结果创建缓存文件。
	_, statErr := os.Stat(path)
	require.True(t, os.IsNotExist(statErr), "Cache file should not exist for empty results")
}
