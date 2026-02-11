package update

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCheckForUpdate_Old 测试旧版本检查更新的情况
// 当当前版本为 v0.10.0，最新版本为 v0.11.0 时，应该检测到更新
func TestCheckForUpdate_Old(t *testing.T) {
	// 调用 Check 函数检查更新
	info, err := Check(t.Context(), "v0.10.0", testClient{"v0.11.0"})

	// 验证没有错误
	require.NoError(t, err)
	// 验证返回的 info 不为 nil
	require.NotNil(t, info)
	// 验证应该检测到更新
	require.True(t, info.Available())
}

// TestCheckForUpdate_Beta 测试与 Beta 版本相关的更新检查情况
func TestCheckForUpdate_Beta(t *testing.T) {
	// 测试场景 1：当前版本是稳定版，最新版本是测试版
	t.Run("current is stable", func(t *testing.T) {
		// 调用 Check 函数检查更新
		info, err := Check(t.Context(), "v0.10.0", testClient{"v0.11.0-beta.1"})

		// 验证没有错误
		require.NoError(t, err)
		// 验证返回的 info 不为 nil
		require.NotNil(t, info)
		// 验证不应该检测到更新（稳定版不应该更新到测试版）
		require.False(t, info.Available())
	})

	// 测试场景 2：当前版本和最新版本都是测试版
	t.Run("current is also beta", func(t *testing.T) {
		// 调用 Check 函数检查更新
		info, err := Check(t.Context(), "v0.11.0-beta.1", testClient{"v0.11.0-beta.2"})

		// 验证没有错误
		require.NoError(t, err)
		// 验证返回的 info 不为 nil
		require.NotNil(t, info)
		// 验证应该检测到更新（测试版应该更新到更新的测试版）
		require.True(t, info.Available())
	})

	// 测试场景 3：当前版本是测试版，最新版本是稳定版
	t.Run("current is beta, latest isn't", func(t *testing.T) {
		// 调用 Check 函数检查更新
		info, err := Check(t.Context(), "v0.11.0-beta.1", testClient{"v0.11.0"})

		// 验证没有错误
		require.NoError(t, err)
		// 验证返回的 info 不为 nil
		require.NotNil(t, info)
		// 验证应该检测到更新（测试版应该更新到稳定版）
		require.True(t, info.Available())
	})
}

// testClient 是用于测试的 Client 实现
// tag 字段存储要返回的版本标签

type testClient struct{ tag string }

// Latest 实现 Client 接口，返回预定义的 Release 对象
func (t testClient) Latest(ctx context.Context) (*Release, error) {
	return &Release{
		TagName: t.tag,                 // 使用预定义的版本标签
		HTMLURL: "https://example.org", // 固定的示例 URL
	}, nil
}
