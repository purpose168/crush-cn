package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestApplyLSPDefaults 测试应用LSP默认值的功能
// 验证当LSP配置未设置某些参数时，是否能够正确应用默认值
func TestApplyLSPDefaults(t *testing.T) {
	t.Parallel()

	// 创建一个包含LSP的配置，该LSP应该获取默认值
	config := &Config{
		LSP: map[string]LSPConfig{
			"gopls": {
				Command: "gopls", // 这应该从powernap获取默认值
			},
			"custom": {
				Command:     "custom-lsp",
				RootMarkers: []string{"custom.toml"}, // 这应该保留其显式配置
			},
		},
	}

	// 应用默认值
	config.applyLSPDefaults()

	// 检查gopls是否获取了默认值（它现在应该有一些根标记）
	goplsConfig := config.LSP["gopls"]
	require.NotEmpty(t, goplsConfig.RootMarkers, "gopls应该已接收默认的根标记")

	// 检查自定义LSP是否保留了其显式配置
	customConfig := config.LSP["custom"]
	require.Equal(t, []string{"custom.toml"}, customConfig.RootMarkers, "自定义LSP应该保留其显式根标记")
}
