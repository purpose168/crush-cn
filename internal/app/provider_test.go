package app

import (
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	"github.com/purpose168/crush-cn/internal/config"
	"github.com/stretchr/testify/require"
)

// TestParseModelStr 测试解析模型字符串的函数
func TestParseModelStr(t *testing.T) {
	tests := []struct {
		name            string // 测试用例名称
		modelStr        string // 模型字符串
		expectedFilter  string // 期望的提供者过滤器
		expectedModelID string // 期望的模型 ID
		setupProviders  func() map[string]config.ProviderConfig // 设置提供者的函数
	}{
		{
			name:            "无斜杠的简单模型",
			modelStr:        "gpt-4o",
			expectedFilter:  "",
			expectedModelID: "gpt-4o",
			setupProviders:  setupMockProviders,
		},
		{
			name:            "有效的提供者和模型",
			modelStr:        "openai/gpt-4o",
			expectedFilter:  "openai",
			expectedModelID: "gpt-4o",
			setupProviders:  setupMockProviders,
		},
		{
			name:            "带多个斜杠且第一部分为无效提供者的模型",
			modelStr:        "moonshot/kimi-k2",
			expectedFilter:  "",
			expectedModelID: "moonshot/kimi-k2",
			setupProviders:  setupMockProviders,
		},
		{
			name:            "带有效提供者和带斜杠模型的完整路径",
			modelStr:        "synthetic/moonshot/kimi-k2",
			expectedFilter:  "synthetic",
			expectedModelID: "moonshot/kimi-k2",
			setupProviders:  setupMockProvidersWithSlashes,
		},
		{
			name:            "空模型字符串",
			modelStr:        "",
			expectedFilter:  "",
			expectedModelID: "",
			setupProviders:  setupMockProviders,
		},
		{
			name:            "带尾部斜杠但提供者有效的模型",
			modelStr:        "openai/",
			expectedFilter:  "openai",
			expectedModelID: "",
			setupProviders:  setupMockProviders,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providers := tt.setupProviders()
			filter, modelID := parseModelStr(providers, tt.modelStr)

			require.Equal(t, tt.expectedFilter, filter, "提供者过滤器不匹配")
			require.Equal(t, tt.expectedModelID, modelID, "模型 ID 不匹配")
		})
	}
}

// setupMockProviders 设置模拟的提供者配置
func setupMockProviders() map[string]config.ProviderConfig {
	return map[string]config.ProviderConfig{
		"openai": {
			ID:     "openai",
			Name:   "OpenAI",
			Models: []catwalk.Model{{ID: "gpt-4o"}, {ID: "gpt-4o-mini"}},
		},
		"anthropic": {
			ID:     "anthropic",
			Name:   "Anthropic",
			Models: []catwalk.Model{{ID: "claude-3-sonnet"}, {ID: "claude-3-opus"}},
		},
	}
}

// setupMockProvidersWithSlashes 设置包含带斜杠模型的模拟提供者配置
func setupMockProvidersWithSlashes() map[string]config.ProviderConfig {
	return map[string]config.ProviderConfig{
		"synthetic": {
			ID:   "synthetic",
			Name: "Synthetic",
			Models: []catwalk.Model{
				{ID: "moonshot/kimi-k2"},
				{ID: "deepseek/deepseek-chat"},
			},
		},
		"openai": {
			ID:     "openai",
			Name:   "OpenAI",
			Models: []catwalk.Model{{ID: "gpt-4o"}},
		},
	}
}

// TestFindModels 测试查找模型的函数
func TestFindModels(t *testing.T) {
	tests := []struct {
		name             string // 测试用例名称
		modelStr         string // 模型字符串
		expectedProvider string // 期望的提供者
		expectedModelID  string // 期望的模型 ID
		expectError      bool   // 是否期望错误
		errorContains    string // 错误消息中应包含的内容
		setupProviders   func() map[string]config.ProviderConfig // 设置提供者的函数
	}{
		{
			name:             "在一个提供者中找到的简单模型",
			modelStr:         "gpt-4o",
			expectedProvider: "openai",
			expectedModelID:  "gpt-4o",
			expectError:      false,
			setupProviders:   setupMockProviders,
		},
		{
			name:             "ID 中带斜杠的模型",
			modelStr:         "moonshot/kimi-k2",
			expectedProvider: "synthetic",
			expectedModelID:  "moonshot/kimi-k2",
			expectError:      false,
			setupProviders:   setupMockProvidersWithSlashes,
		},
		{
			name:             "提供者和 ID 中带斜杠的模型",
			modelStr:         "synthetic/moonshot/kimi-k2",
			expectedProvider: "synthetic",
			expectedModelID:  "moonshot/kimi-k2",
			expectError:      false,
			setupProviders:   setupMockProvidersWithSlashes,
		},
		{
			name:           "未找到模型",
			modelStr:       "nonexistent-model",
			expectError:    true,
			errorContains:  "not found",
			setupProviders: setupMockProviders,
		},
		{
			name:           "指定了无效的提供者",
			modelStr:       "nonexistent-provider/gpt-4o",
			expectError:    true,
			errorContains:  "provider",
			setupProviders: setupMockProviders,
		},
		{
			name:          "在多个提供者中找到模型但未指定提供者过滤器",
			modelStr:      "shared-model",
			expectError:   true,
			errorContains: "multiple providers",
			setupProviders: func() map[string]config.ProviderConfig {
				return map[string]config.ProviderConfig{
					"openai": {
						ID:     "openai",
						Models: []catwalk.Model{{ID: "shared-model"}},
					},
					"anthropic": {
						ID:     "anthropic",
						Models: []catwalk.Model{{ID: "shared-model"}},
					},
				}
			},
		},
		{
			name:           "空模型字符串",
			modelStr:       "",
			expectError:    true,
			errorContains:  "not found",
			setupProviders: setupMockProviders,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providers := tt.setupProviders()

			// 使用 findModels，模型类型为 "large"，小型模型为空。
			matches, _, err := findModels(providers, tt.modelStr, "")
			if err != nil {
				if tt.expectError {
					require.Contains(t, err.Error(), tt.errorContains)
				} else {
					require.NoError(t, err)
				}
				return
			}

			// 验证匹配结果。
			match, err := validateMatches(matches, tt.modelStr, "large")

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedProvider, match.provider)
				require.Equal(t, tt.expectedModelID, match.modelID)
			}
		})
	}
}
