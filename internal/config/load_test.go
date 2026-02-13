package config

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	"github.com/purpose168/crush-cn/internal/csync"
	"github.com/purpose168/crush-cn/internal/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain 是测试的主入口函数，用于设置测试环境
func TestMain(m *testing.M) {
	// 设置默认日志处理器为丢弃所有日志输出
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	exitVal := m.Run()
	os.Exit(exitVal)
}

// TestConfig_LoadFromBytes 测试从字节数组加载配置的功能
func TestConfig_LoadFromBytes(t *testing.T) {
	// 定义三组测试数据，模拟从多个配置源加载配置
	data1 := []byte(`{"providers": {"openai": {"api_key": "key1", "base_url": "https://api.openai.com/v1"}}}`)
	data2 := []byte(`{"providers": {"openai": {"api_key": "key2", "base_url": "https://api.openai.com/v2"}}}`)
	data3 := []byte(`{"providers": {"openai": {}}}`)

	// 从字节数组加载配置，后面的配置会覆盖前面的配置
	loadedConfig, err := loadFromBytes([][]byte{data1, data2, data3})

	require.NoError(t, err)
	require.NotNil(t, loadedConfig)
	require.Equal(t, 1, loadedConfig.Providers.Len())
	pc, _ := loadedConfig.Providers.Get("openai")
	// 验证最终使用的是最后一个有效的配置
	require.Equal(t, "key2", pc.APIKey)
	require.Equal(t, "https://api.openai.com/v2", pc.BaseURL)
}

// TestConfig_setDefaults 测试设置默认值的功能
func TestConfig_setDefaults(t *testing.T) {
	cfg := &Config{}

	// 设置默认值，工作目录为 /tmp
	cfg.setDefaults("/tmp", "")

	// 验证所有必要的配置项都被初始化
	require.NotNil(t, cfg.Options)
	require.NotNil(t, cfg.Options.TUI)
	require.NotNil(t, cfg.Options.ContextPaths)
	require.NotNil(t, cfg.Providers)
	require.NotNil(t, cfg.Models)
	require.NotNil(t, cfg.LSP)
	require.NotNil(t, cfg.MCP)
	// 验证数据目录设置正确
	require.Equal(t, filepath.Join("/tmp", ".crush"), cfg.Options.DataDirectory)
	require.Equal(t, "AGENTS.md", cfg.Options.InitializeAs)
	// 验证默认上下文路径都被包含
	for _, path := range defaultContextPaths {
		require.Contains(t, cfg.Options.ContextPaths, path)
	}
	require.Equal(t, "/tmp", cfg.workingDir)
}

// TestConfig_configureProviders 测试配置提供商的功能
func TestConfig_configureProviders(t *testing.T) {
	knownProviders := []catwalk.Provider{
		{
			ID:          "openai",
			APIKey:      "$OPENAI_API_KEY",
			APIEndpoint: "https://api.openai.com/v1",
			Models: []catwalk.Model{{
				ID: "test-model",
			}},
		},
	}

	cfg := &Config{}
	cfg.setDefaults("/tmp", "")
	env := env.NewFromMap(map[string]string{
		"OPENAI_API_KEY": "test-key",
	})
	resolver := NewEnvironmentVariableResolver(env)
	err := cfg.configureProviders(env, resolver, knownProviders)
	require.NoError(t, err)
	require.Equal(t, 1, cfg.Providers.Len())

	// 我们要确保保留配置的 API 密钥作为占位符
	pc, _ := cfg.Providers.Get("openai")
	require.Equal(t, "$OPENAI_API_KEY", pc.APIKey)
}

// TestConfig_configureProvidersWithOverride 测试配置提供商时的覆盖功能
func TestConfig_configureProvidersWithOverride(t *testing.T) {
	knownProviders := []catwalk.Provider{
		{
			ID:          "openai",
			APIKey:      "$OPENAI_API_KEY",
			APIEndpoint: "https://api.openai.com/v1",
			Models: []catwalk.Model{{
				ID: "test-model",
			}},
		},
	}

	cfg := &Config{
		Providers: csync.NewMap[string, ProviderConfig](),
	}
	// 用户配置了自定义的提供商设置
	cfg.Providers.Set("openai", ProviderConfig{
		APIKey:  "xyz",
		BaseURL: "https://api.openai.com/v2",
		Models: []catwalk.Model{
			{
				ID:   "test-model",
				Name: "Updated",
			},
			{
				ID: "another-model",
			},
		},
	})
	cfg.setDefaults("/tmp", "")

	env := env.NewFromMap(map[string]string{
		"OPENAI_API_KEY": "test-key",
	})
	resolver := NewEnvironmentVariableResolver(env)
	err := cfg.configureProviders(env, resolver, knownProviders)
	require.NoError(t, err)
	require.Equal(t, 1, cfg.Providers.Len())

	// 我们要确保保留配置的 API 密钥作为占位符
	pc, _ := cfg.Providers.Get("openai")
	require.Equal(t, "xyz", pc.APIKey)
	require.Equal(t, "https://api.openai.com/v2", pc.BaseURL)
	require.Len(t, pc.Models, 2)
	require.Equal(t, "Updated", pc.Models[0].Name)
}

// TestConfig_configureProvidersWithNewProvider 测试添加新提供商的功能
func TestConfig_configureProvidersWithNewProvider(t *testing.T) {
	knownProviders := []catwalk.Provider{
		{
			ID:          "openai",
			APIKey:      "$OPENAI_API_KEY",
			APIEndpoint: "https://api.openai.com/v1",
			Models: []catwalk.Model{{
				ID: "test-model",
			}},
		},
	}

	cfg := &Config{
		Providers: csync.NewMapFrom(map[string]ProviderConfig{
			"custom": {
				APIKey:  "xyz",
				BaseURL: "https://api.someendpoint.com/v2",
				Models: []catwalk.Model{
					{
						ID: "test-model",
					},
				},
			},
		}),
	}
	cfg.setDefaults("/tmp", "")
	env := env.NewFromMap(map[string]string{
		"OPENAI_API_KEY": "test-key",
	})
	resolver := NewEnvironmentVariableResolver(env)
	err := cfg.configureProviders(env, resolver, knownProviders)
	require.NoError(t, err)
	// 应该是 2，因为有环境变量
	require.Equal(t, cfg.Providers.Len(), 2)

	// 我们要确保保留配置的 API 密钥作为占位符
	pc, _ := cfg.Providers.Get("custom")
	require.Equal(t, "xyz", pc.APIKey)
	// 确保我们正确设置了 ID
	require.Equal(t, "custom", pc.ID)
	require.Equal(t, "https://api.someendpoint.com/v2", pc.BaseURL)
	require.Len(t, pc.Models, 1)

	_, ok := cfg.Providers.Get("openai")
	require.True(t, ok, "OpenAI 提供商应该仍然存在")
}

// TestConfig_configureProvidersBedrockWithCredentials 测试使用凭证配置 Bedrock 提供商
func TestConfig_configureProvidersBedrockWithCredentials(t *testing.T) {
	knownProviders := []catwalk.Provider{
		{
			ID:          catwalk.InferenceProviderBedrock,
			APIKey:      "",
			APIEndpoint: "",
			Models: []catwalk.Model{{
				ID: "anthropic.claude-sonnet-4-20250514-v1:0",
			}},
		},
	}

	cfg := &Config{}
	cfg.setDefaults("/tmp", "")
	env := env.NewFromMap(map[string]string{
		"AWS_ACCESS_KEY_ID":     "test-key-id",
		"AWS_SECRET_ACCESS_KEY": "test-secret-key",
	})
	resolver := NewEnvironmentVariableResolver(env)
	err := cfg.configureProviders(env, resolver, knownProviders)
	require.NoError(t, err)
	require.Equal(t, cfg.Providers.Len(), 1)

	bedrockProvider, ok := cfg.Providers.Get("bedrock")
	require.True(t, ok, "Bedrock 提供商应该存在")
	require.Len(t, bedrockProvider.Models, 1)
	require.Equal(t, "anthropic.claude-sonnet-4-20250514-v1:0", bedrockProvider.Models[0].ID)
}

// TestConfig_configureProvidersBedrockWithoutCredentials 测试在没有凭证时配置 Bedrock 提供商
func TestConfig_configureProvidersBedrockWithoutCredentials(t *testing.T) {
	knownProviders := []catwalk.Provider{
		{
			ID:          catwalk.InferenceProviderBedrock,
			APIKey:      "",
			APIEndpoint: "",
			Models: []catwalk.Model{{
				ID: "anthropic.claude-sonnet-4-20250514-v1:0",
			}},
		},
	}

	cfg := &Config{}
	cfg.setDefaults("/tmp", "")
	env := env.NewFromMap(map[string]string{})
	resolver := NewEnvironmentVariableResolver(env)
	err := cfg.configureProviders(env, resolver, knownProviders)
	require.NoError(t, err)
	// 没有凭证时不应该配置提供商
	require.Equal(t, cfg.Providers.Len(), 0)
}

// TestConfig_configureProvidersBedrockWithoutUnsupportedModel 测试使用不支持的模型配置 Bedrock 提供商
func TestConfig_configureProvidersBedrockWithoutUnsupportedModel(t *testing.T) {
	knownProviders := []catwalk.Provider{
		{
			ID:          catwalk.InferenceProviderBedrock,
			APIKey:      "",
			APIEndpoint: "",
			Models: []catwalk.Model{{
				ID: "some-random-model",
			}},
		},
	}

	cfg := &Config{}
	cfg.setDefaults("/tmp", "")
	env := env.NewFromMap(map[string]string{
		"AWS_ACCESS_KEY_ID":     "test-key-id",
		"AWS_SECRET_ACCESS_KEY": "test-secret-key",
	})
	resolver := NewEnvironmentVariableResolver(env)
	err := cfg.configureProviders(env, resolver, knownProviders)
	require.Error(t, err)
}

// TestConfig_configureProvidersVertexAIWithCredentials 测试使用凭证配置 VertexAI 提供商
func TestConfig_configureProvidersVertexAIWithCredentials(t *testing.T) {
	knownProviders := []catwalk.Provider{
		{
			ID:          catwalk.InferenceProviderVertexAI,
			APIKey:      "",
			APIEndpoint: "",
			Models: []catwalk.Model{{
				ID: "gemini-pro",
			}},
		},
	}

	cfg := &Config{}
	cfg.setDefaults("/tmp", "")
	env := env.NewFromMap(map[string]string{
		"VERTEXAI_PROJECT":  "test-project",
		"VERTEXAI_LOCATION": "us-central1",
	})
	resolver := NewEnvironmentVariableResolver(env)
	err := cfg.configureProviders(env, resolver, knownProviders)
	require.NoError(t, err)
	require.Equal(t, cfg.Providers.Len(), 1)

	vertexProvider, ok := cfg.Providers.Get("vertexai")
	require.True(t, ok, "VertexAI 提供商应该存在")
	require.Len(t, vertexProvider.Models, 1)
	require.Equal(t, "gemini-pro", vertexProvider.Models[0].ID)
	require.Equal(t, "test-project", vertexProvider.ExtraParams["project"])
	require.Equal(t, "us-central1", vertexProvider.ExtraParams["location"])
}

// TestConfig_configureProvidersVertexAIWithoutCredentials 测试在没有凭证时配置 VertexAI 提供商
func TestConfig_configureProvidersVertexAIWithoutCredentials(t *testing.T) {
	knownProviders := []catwalk.Provider{
		{
			ID:          catwalk.InferenceProviderVertexAI,
			APIKey:      "",
			APIEndpoint: "",
			Models: []catwalk.Model{{
				ID: "gemini-pro",
			}},
		},
	}

	cfg := &Config{}
	cfg.setDefaults("/tmp", "")
	env := env.NewFromMap(map[string]string{
		"GOOGLE_GENAI_USE_VERTEXAI": "false",
		"GOOGLE_CLOUD_PROJECT":      "test-project",
		"GOOGLE_CLOUD_LOCATION":     "us-central1",
	})
	resolver := NewEnvironmentVariableResolver(env)
	err := cfg.configureProviders(env, resolver, knownProviders)
	require.NoError(t, err)
	// 没有正确的凭证时不应该配置提供商
	require.Equal(t, cfg.Providers.Len(), 0)
}

// TestConfig_configureProvidersVertexAIMissingProject 测试缺少项目时配置 VertexAI 提供商
func TestConfig_configureProvidersVertexAIMissingProject(t *testing.T) {
	knownProviders := []catwalk.Provider{
		{
			ID:          catwalk.InferenceProviderVertexAI,
			APIKey:      "",
			APIEndpoint: "",
			Models: []catwalk.Model{{
				ID: "gemini-pro",
			}},
		},
	}

	cfg := &Config{}
	cfg.setDefaults("/tmp", "")
	env := env.NewFromMap(map[string]string{
		"GOOGLE_GENAI_USE_VERTEXAI": "true",
		"GOOGLE_CLOUD_LOCATION":     "us-central1",
	})
	resolver := NewEnvironmentVariableResolver(env)
	err := cfg.configureProviders(env, resolver, knownProviders)
	require.NoError(t, err)
	// 没有项目时不应该配置提供商
	require.Equal(t, cfg.Providers.Len(), 0)
}

// TestConfig_configureProvidersSetProviderID 测试设置提供商 ID 的功能
func TestConfig_configureProvidersSetProviderID(t *testing.T) {
	knownProviders := []catwalk.Provider{
		{
			ID:          "openai",
			APIKey:      "$OPENAI_API_KEY",
			APIEndpoint: "https://api.openai.com/v1",
			Models: []catwalk.Model{{
				ID: "test-model",
			}},
		},
	}

	cfg := &Config{}
	cfg.setDefaults("/tmp", "")
	env := env.NewFromMap(map[string]string{
		"OPENAI_API_KEY": "test-key",
	})
	resolver := NewEnvironmentVariableResolver(env)
	err := cfg.configureProviders(env, resolver, knownProviders)
	require.NoError(t, err)
	require.Equal(t, cfg.Providers.Len(), 1)

	// 提供商 ID 应该被设置
	pc, _ := cfg.Providers.Get("openai")
	require.Equal(t, "openai", pc.ID)
}

// TestConfig_EnabledProviders 测试获取已启用提供商的功能
func TestConfig_EnabledProviders(t *testing.T) {
	t.Run("all providers enabled", func(t *testing.T) {
		cfg := &Config{
			Providers: csync.NewMapFrom(map[string]ProviderConfig{
				"openai": {
					ID:      "openai",
					APIKey:  "key1",
					Disable: false,
				},
				"anthropic": {
					ID:      "anthropic",
					APIKey:  "key2",
					Disable: false,
				},
			}),
		}

		enabled := cfg.EnabledProviders()
		require.Len(t, enabled, 2)
	})

	t.Run("some providers disabled", func(t *testing.T) {
		cfg := &Config{
			Providers: csync.NewMapFrom(map[string]ProviderConfig{
				"openai": {
					ID:      "openai",
					APIKey:  "key1",
					Disable: false,
				},
				"anthropic": {
					ID:      "anthropic",
					APIKey:  "key2",
					Disable: true,
				},
			}),
		}

		enabled := cfg.EnabledProviders()
		require.Len(t, enabled, 1)
		require.Equal(t, "openai", enabled[0].ID)
	})

	t.Run("empty providers map", func(t *testing.T) {
		cfg := &Config{
			Providers: csync.NewMap[string, ProviderConfig](),
		}

		enabled := cfg.EnabledProviders()
		require.Len(t, enabled, 0)
	})
}

// TestConfig_IsConfigured 测试检查配置是否已配置的功能
func TestConfig_IsConfigured(t *testing.T) {
	t.Run("returns true when at least one provider is enabled", func(t *testing.T) {
		cfg := &Config{
			Providers: csync.NewMapFrom(map[string]ProviderConfig{
				"openai": {
					ID:      "openai",
					APIKey:  "key1",
					Disable: false,
				},
			}),
		}

		require.True(t, cfg.IsConfigured())
	})

	t.Run("returns false when no providers are configured", func(t *testing.T) {
		cfg := &Config{
			Providers: csync.NewMap[string, ProviderConfig](),
		}

		require.False(t, cfg.IsConfigured())
	})

	t.Run("returns false when all providers are disabled", func(t *testing.T) {
		cfg := &Config{
			Providers: csync.NewMapFrom(map[string]ProviderConfig{
				"openai": {
					ID:      "openai",
					APIKey:  "key1",
					Disable: true,
				},
				"anthropic": {
					ID:      "anthropic",
					APIKey:  "key2",
					Disable: true,
				},
			}),
		}

		require.False(t, cfg.IsConfigured())
	})
}

// TestConfig_setupAgentsWithNoDisabledTools 测试在没有禁用工具的情况下设置代理
func TestConfig_setupAgentsWithNoDisabledTools(t *testing.T) {
	cfg := &Config{
		Options: &Options{
			DisabledTools: []string{},
		},
	}

	cfg.SetupAgents()
	coderAgent, ok := cfg.Agents[AgentCoder]
	require.True(t, ok)
	assert.Equal(t, allToolNames(), coderAgent.AllowedTools)

	taskAgent, ok := cfg.Agents[AgentTask]
	require.True(t, ok)
	assert.Equal(t, []string{"glob", "grep", "ls", "sourcegraph", "view"}, taskAgent.AllowedTools)
}

// TestConfig_setupAgentsWithDisabledTools 测试在有禁用工具的情况下设置代理
func TestConfig_setupAgentsWithDisabledTools(t *testing.T) {
	cfg := &Config{
		Options: &Options{
			DisabledTools: []string{
				"edit",
				"download",
				"grep",
			},
		},
	}

	cfg.SetupAgents()
	coderAgent, ok := cfg.Agents[AgentCoder]
	require.True(t, ok)

	assert.Equal(t, []string{"agent", "bash", "job_output", "job_kill", "multiedit", "lsp_diagnostics", "lsp_references", "lsp_restart", "fetch", "agentic_fetch", "glob", "ls", "sourcegraph", "todos", "view", "write", "list_mcp_resources", "read_mcp_resource"}, coderAgent.AllowedTools)

	taskAgent, ok := cfg.Agents[AgentTask]
	require.True(t, ok)
	assert.Equal(t, []string{"glob", "ls", "sourcegraph", "view"}, taskAgent.AllowedTools)
}

// TestConfig_setupAgentsWithEveryReadOnlyToolDisabled 测试在所有只读工具都被禁用的情况下设置代理
func TestConfig_setupAgentsWithEveryReadOnlyToolDisabled(t *testing.T) {
	cfg := &Config{
		Options: &Options{
			DisabledTools: []string{
				"glob",
				"grep",
				"ls",
				"sourcegraph",
				"view",
			},
		},
	}

	cfg.SetupAgents()
	coderAgent, ok := cfg.Agents[AgentCoder]
	require.True(t, ok)
	assert.Equal(t, []string{"agent", "bash", "job_output", "job_kill", "download", "edit", "multiedit", "lsp_diagnostics", "lsp_references", "lsp_restart", "fetch", "agentic_fetch", "todos", "write", "list_mcp_resources", "read_mcp_resource"}, coderAgent.AllowedTools)

	taskAgent, ok := cfg.Agents[AgentTask]
	require.True(t, ok)
	assert.Len(t, taskAgent.AllowedTools, 0)
}

// TestConfig_configureProvidersWithDisabledProvider 测试配置禁用的提供商
func TestConfig_configureProvidersWithDisabledProvider(t *testing.T) {
	knownProviders := []catwalk.Provider{
		{
			ID:          "openai",
			APIKey:      "$OPENAI_API_KEY",
			APIEndpoint: "https://api.openai.com/v1",
			Models: []catwalk.Model{{
				ID: "test-model",
			}},
		},
	}

	cfg := &Config{
		Providers: csync.NewMapFrom(map[string]ProviderConfig{
			"openai": {
				Disable: true,
			},
		}),
	}
	cfg.setDefaults("/tmp", "")

	env := env.NewFromMap(map[string]string{
		"OPENAI_API_KEY": "test-key",
	})
	resolver := NewEnvironmentVariableResolver(env)
	err := cfg.configureProviders(env, resolver, knownProviders)
	require.NoError(t, err)

	require.Equal(t, cfg.Providers.Len(), 1)
	prov, exists := cfg.Providers.Get("openai")
	require.True(t, exists)
	require.True(t, prov.Disable)
}

// TestConfig_configureProvidersCustomProviderValidation 测试自定义提供商的验证
func TestConfig_configureProvidersCustomProviderValidation(t *testing.T) {
	t.Run("custom provider with missing API key is allowed, but not known providers", func(t *testing.T) {
		cfg := &Config{
			Providers: csync.NewMapFrom(map[string]ProviderConfig{
				"custom": {
					BaseURL: "https://api.custom.com/v1",
					Models: []catwalk.Model{{
						ID: "test-model",
					}},
				},
				"openai": {
					APIKey: "$MISSING",
				},
			}),
		}
		cfg.setDefaults("/tmp", "")

		env := env.NewFromMap(map[string]string{})
		resolver := NewEnvironmentVariableResolver(env)
		err := cfg.configureProviders(env, resolver, []catwalk.Provider{})
		require.NoError(t, err)

		require.Equal(t, cfg.Providers.Len(), 1)
		_, exists := cfg.Providers.Get("custom")
		require.True(t, exists)
	})

	t.Run("custom provider with missing BaseURL is removed", func(t *testing.T) {
		cfg := &Config{
			Providers: csync.NewMapFrom(map[string]ProviderConfig{
				"custom": {
					APIKey: "test-key",
					Models: []catwalk.Model{{
						ID: "test-model",
					}},
				},
			}),
		}
		cfg.setDefaults("/tmp", "")

		env := env.NewFromMap(map[string]string{})
		resolver := NewEnvironmentVariableResolver(env)
		err := cfg.configureProviders(env, resolver, []catwalk.Provider{})
		require.NoError(t, err)

		require.Equal(t, cfg.Providers.Len(), 0)
		_, exists := cfg.Providers.Get("custom")
		require.False(t, exists)
	})

	t.Run("custom provider with no models is removed", func(t *testing.T) {
		cfg := &Config{
			Providers: csync.NewMapFrom(map[string]ProviderConfig{
				"custom": {
					APIKey:  "test-key",
					BaseURL: "https://api.custom.com/v1",
					Models:  []catwalk.Model{},
				},
			}),
		}
		cfg.setDefaults("/tmp", "")

		env := env.NewFromMap(map[string]string{})
		resolver := NewEnvironmentVariableResolver(env)
		err := cfg.configureProviders(env, resolver, []catwalk.Provider{})
		require.NoError(t, err)

		require.Equal(t, cfg.Providers.Len(), 0)
		_, exists := cfg.Providers.Get("custom")
		require.False(t, exists)
	})

	t.Run("custom provider with unsupported type is removed", func(t *testing.T) {
		cfg := &Config{
			Providers: csync.NewMapFrom(map[string]ProviderConfig{
				"custom": {
					APIKey:  "test-key",
					BaseURL: "https://api.custom.com/v1",
					Type:    "unsupported",
					Models: []catwalk.Model{{
						ID: "test-model",
					}},
				},
			}),
		}
		cfg.setDefaults("/tmp", "")

		env := env.NewFromMap(map[string]string{})
		resolver := NewEnvironmentVariableResolver(env)
		err := cfg.configureProviders(env, resolver, []catwalk.Provider{})
		require.NoError(t, err)

		require.Equal(t, cfg.Providers.Len(), 0)
		_, exists := cfg.Providers.Get("custom")
		require.False(t, exists)
	})

	t.Run("valid custom provider is kept and ID is set", func(t *testing.T) {
		cfg := &Config{
			Providers: csync.NewMapFrom(map[string]ProviderConfig{
				"custom": {
					APIKey:  "test-key",
					BaseURL: "https://api.custom.com/v1",
					Type:    catwalk.TypeOpenAI,
					Models: []catwalk.Model{{
						ID: "test-model",
					}},
				},
			}),
		}
		cfg.setDefaults("/tmp", "")

		env := env.NewFromMap(map[string]string{})
		resolver := NewEnvironmentVariableResolver(env)
		err := cfg.configureProviders(env, resolver, []catwalk.Provider{})
		require.NoError(t, err)

		require.Equal(t, cfg.Providers.Len(), 1)
		customProvider, exists := cfg.Providers.Get("custom")
		require.True(t, exists)
		require.Equal(t, "custom", customProvider.ID)
		require.Equal(t, "test-key", customProvider.APIKey)
		require.Equal(t, "https://api.custom.com/v1", customProvider.BaseURL)
	})

	t.Run("custom anthropic provider is supported", func(t *testing.T) {
		cfg := &Config{
			Providers: csync.NewMapFrom(map[string]ProviderConfig{
				"custom-anthropic": {
					APIKey:  "test-key",
					BaseURL: "https://api.anthropic.com/v1",
					Type:    catwalk.TypeAnthropic,
					Models: []catwalk.Model{{
						ID: "claude-3-sonnet",
					}},
				},
			}),
		}
		cfg.setDefaults("/tmp", "")

		env := env.NewFromMap(map[string]string{})
		resolver := NewEnvironmentVariableResolver(env)
		err := cfg.configureProviders(env, resolver, []catwalk.Provider{})
		require.NoError(t, err)

		require.Equal(t, cfg.Providers.Len(), 1)
		customProvider, exists := cfg.Providers.Get("custom-anthropic")
		require.True(t, exists)
		require.Equal(t, "custom-anthropic", customProvider.ID)
		require.Equal(t, "test-key", customProvider.APIKey)
		require.Equal(t, "https://api.anthropic.com/v1", customProvider.BaseURL)
		require.Equal(t, catwalk.TypeAnthropic, customProvider.Type)
	})

	t.Run("disabled custom provider is removed", func(t *testing.T) {
		cfg := &Config{
			Providers: csync.NewMapFrom(map[string]ProviderConfig{
				"custom": {
					APIKey:  "test-key",
					BaseURL: "https://api.custom.com/v1",
					Type:    catwalk.TypeOpenAI,
					Disable: true,
					Models: []catwalk.Model{{
						ID: "test-model",
					}},
				},
			}),
		}
		cfg.setDefaults("/tmp", "")

		env := env.NewFromMap(map[string]string{})
		resolver := NewEnvironmentVariableResolver(env)
		err := cfg.configureProviders(env, resolver, []catwalk.Provider{})
		require.NoError(t, err)

		require.Equal(t, cfg.Providers.Len(), 0)
		_, exists := cfg.Providers.Get("custom")
		require.False(t, exists)
	})
}

// TestConfig_configureProvidersEnhancedCredentialValidation 测试增强的凭证验证
func TestConfig_configureProvidersEnhancedCredentialValidation(t *testing.T) {
	t.Run("VertexAI provider removed when credentials missing with existing config", func(t *testing.T) {
		knownProviders := []catwalk.Provider{
			{
				ID:          catwalk.InferenceProviderVertexAI,
				APIKey:      "",
				APIEndpoint: "",
				Models: []catwalk.Model{{
					ID: "gemini-pro",
				}},
			},
		}

		cfg := &Config{
			Providers: csync.NewMapFrom(map[string]ProviderConfig{
				"vertexai": {
					BaseURL: "custom-url",
				},
			}),
		}
		cfg.setDefaults("/tmp", "")

		env := env.NewFromMap(map[string]string{
			"GOOGLE_GENAI_USE_VERTEXAI": "false",
		})
		resolver := NewEnvironmentVariableResolver(env)
		err := cfg.configureProviders(env, resolver, knownProviders)
		require.NoError(t, err)

		require.Equal(t, cfg.Providers.Len(), 0)
		_, exists := cfg.Providers.Get("vertexai")
		require.False(t, exists)
	})

	t.Run("Bedrock provider removed when AWS credentials missing with existing config", func(t *testing.T) {
		knownProviders := []catwalk.Provider{
			{
				ID:          catwalk.InferenceProviderBedrock,
				APIKey:      "",
				APIEndpoint: "",
				Models: []catwalk.Model{{
					ID: "anthropic.claude-sonnet-4-20250514-v1:0",
				}},
			},
		}

		cfg := &Config{
			Providers: csync.NewMapFrom(map[string]ProviderConfig{
				"bedrock": {
					BaseURL: "custom-url",
				},
			}),
		}
		cfg.setDefaults("/tmp", "")

		env := env.NewFromMap(map[string]string{})
		resolver := NewEnvironmentVariableResolver(env)
		err := cfg.configureProviders(env, resolver, knownProviders)
		require.NoError(t, err)

		require.Equal(t, cfg.Providers.Len(), 0)
		_, exists := cfg.Providers.Get("bedrock")
		require.False(t, exists)
	})

	t.Run("provider removed when API key missing with existing config", func(t *testing.T) {
		knownProviders := []catwalk.Provider{
			{
				ID:          "openai",
				APIKey:      "$MISSING_API_KEY",
				APIEndpoint: "https://api.openai.com/v1",
				Models: []catwalk.Model{{
					ID: "test-model",
				}},
			},
		}

		cfg := &Config{
			Providers: csync.NewMapFrom(map[string]ProviderConfig{
				"openai": {
					BaseURL: "custom-url",
				},
			}),
		}
		cfg.setDefaults("/tmp", "")

		env := env.NewFromMap(map[string]string{})
		resolver := NewEnvironmentVariableResolver(env)
		err := cfg.configureProviders(env, resolver, knownProviders)
		require.NoError(t, err)

		require.Equal(t, cfg.Providers.Len(), 0)
		_, exists := cfg.Providers.Get("openai")
		require.False(t, exists)
	})

	t.Run("known provider should still be added if the endpoint is missing the client will use default endpoints", func(t *testing.T) {
		knownProviders := []catwalk.Provider{
			{
				ID:          "openai",
				APIKey:      "$OPENAI_API_KEY",
				APIEndpoint: "$MISSING_ENDPOINT",
				Models: []catwalk.Model{{
					ID: "test-model",
				}},
			},
		}

		cfg := &Config{
			Providers: csync.NewMapFrom(map[string]ProviderConfig{
				"openai": {
					APIKey: "test-key",
				},
			}),
		}
		cfg.setDefaults("/tmp", "")

		env := env.NewFromMap(map[string]string{
			"OPENAI_API_KEY": "test-key",
		})
		resolver := NewEnvironmentVariableResolver(env)
		err := cfg.configureProviders(env, resolver, knownProviders)
		require.NoError(t, err)

		require.Equal(t, cfg.Providers.Len(), 1)
		_, exists := cfg.Providers.Get("openai")
		require.True(t, exists)
	})
}

// TestConfig_defaultModelSelection 测试默认模型选择功能
func TestConfig_defaultModelSelection(t *testing.T) {
	t.Run("default behavior uses the default models for given provider", func(t *testing.T) {
		knownProviders := []catwalk.Provider{
			{
				ID:                  "openai",
				APIKey:              "abc",
				DefaultLargeModelID: "large-model",
				DefaultSmallModelID: "small-model",
				Models: []catwalk.Model{
					{
						ID:               "large-model",
						DefaultMaxTokens: 1000,
					},
					{
						ID:               "small-model",
						DefaultMaxTokens: 500,
					},
				},
			},
		}

		cfg := &Config{}
		cfg.setDefaults("/tmp", "")
		env := env.NewFromMap(map[string]string{})
		resolver := NewEnvironmentVariableResolver(env)
		err := cfg.configureProviders(env, resolver, knownProviders)
		require.NoError(t, err)

		large, small, err := cfg.defaultModelSelection(knownProviders)
		require.NoError(t, err)
		require.Equal(t, "large-model", large.Model)
		require.Equal(t, "openai", large.Provider)
		require.Equal(t, int64(1000), large.MaxTokens)
		require.Equal(t, "small-model", small.Model)
		require.Equal(t, "openai", small.Provider)
		require.Equal(t, int64(500), small.MaxTokens)
	})
	t.Run("should error if no providers configured", func(t *testing.T) {
		knownProviders := []catwalk.Provider{
			{
				ID:                  "openai",
				APIKey:              "$MISSING_KEY",
				DefaultLargeModelID: "large-model",
				DefaultSmallModelID: "small-model",
				Models: []catwalk.Model{
					{
						ID:               "large-model",
						DefaultMaxTokens: 1000,
					},
					{
						ID:               "small-model",
						DefaultMaxTokens: 500,
					},
				},
			},
		}

		cfg := &Config{}
		cfg.setDefaults("/tmp", "")
		env := env.NewFromMap(map[string]string{})
		resolver := NewEnvironmentVariableResolver(env)
		err := cfg.configureProviders(env, resolver, knownProviders)
		require.NoError(t, err)

		_, _, err = cfg.defaultModelSelection(knownProviders)
		require.Error(t, err)
	})
	t.Run("should error if model is missing", func(t *testing.T) {
		knownProviders := []catwalk.Provider{
			{
				ID:                  "openai",
				APIKey:              "abc",
				DefaultLargeModelID: "large-model",
				DefaultSmallModelID: "small-model",
				Models: []catwalk.Model{
					{
						ID:               "not-large-model",
						DefaultMaxTokens: 1000,
					},
					{
						ID:               "small-model",
						DefaultMaxTokens: 500,
					},
				},
			},
		}

		cfg := &Config{}
		cfg.setDefaults("/tmp", "")
		env := env.NewFromMap(map[string]string{})
		resolver := NewEnvironmentVariableResolver(env)
		err := cfg.configureProviders(env, resolver, knownProviders)
		require.NoError(t, err)
		_, _, err = cfg.defaultModelSelection(knownProviders)
		require.Error(t, err)
	})

	t.Run("should configure the default models with a custom provider", func(t *testing.T) {
		knownProviders := []catwalk.Provider{
			{
				ID:                  "openai",
				APIKey:              "$MISSING", // will not be included in the config
				DefaultLargeModelID: "large-model",
				DefaultSmallModelID: "small-model",
				Models: []catwalk.Model{
					{
						ID:               "not-large-model",
						DefaultMaxTokens: 1000,
					},
					{
						ID:               "small-model",
						DefaultMaxTokens: 500,
					},
				},
			},
		}

		cfg := &Config{
			Providers: csync.NewMapFrom(map[string]ProviderConfig{
				"custom": {
					APIKey:  "test-key",
					BaseURL: "https://api.custom.com/v1",
					Models: []catwalk.Model{
						{
							ID:               "model",
							DefaultMaxTokens: 600,
						},
					},
				},
			}),
		}
		cfg.setDefaults("/tmp", "")
		env := env.NewFromMap(map[string]string{})
		resolver := NewEnvironmentVariableResolver(env)
		err := cfg.configureProviders(env, resolver, knownProviders)
		require.NoError(t, err)
		large, small, err := cfg.defaultModelSelection(knownProviders)
		require.NoError(t, err)
		require.Equal(t, "model", large.Model)
		require.Equal(t, "custom", large.Provider)
		require.Equal(t, int64(600), large.MaxTokens)
		require.Equal(t, "model", small.Model)
		require.Equal(t, "custom", small.Provider)
		require.Equal(t, int64(600), small.MaxTokens)
	})

	t.Run("should fail if no model configured", func(t *testing.T) {
		knownProviders := []catwalk.Provider{
			{
				ID:                  "openai",
				APIKey:              "$MISSING", // will not be included in the config
				DefaultLargeModelID: "large-model",
				DefaultSmallModelID: "small-model",
				Models: []catwalk.Model{
					{
						ID:               "not-large-model",
						DefaultMaxTokens: 1000,
					},
					{
						ID:               "small-model",
						DefaultMaxTokens: 500,
					},
				},
			},
		}

		cfg := &Config{
			Providers: csync.NewMapFrom(map[string]ProviderConfig{
				"custom": {
					APIKey:  "test-key",
					BaseURL: "https://api.custom.com/v1",
					Models:  []catwalk.Model{},
				},
			}),
		}
		cfg.setDefaults("/tmp", "")
		env := env.NewFromMap(map[string]string{})
		resolver := NewEnvironmentVariableResolver(env)
		err := cfg.configureProviders(env, resolver, knownProviders)
		require.NoError(t, err)
		_, _, err = cfg.defaultModelSelection(knownProviders)
		require.Error(t, err)
	})
	t.Run("should use the default provider first", func(t *testing.T) {
		knownProviders := []catwalk.Provider{
			{
				ID:                  "openai",
				APIKey:              "set",
				DefaultLargeModelID: "large-model",
				DefaultSmallModelID: "small-model",
				Models: []catwalk.Model{
					{
						ID:               "large-model",
						DefaultMaxTokens: 1000,
					},
					{
						ID:               "small-model",
						DefaultMaxTokens: 500,
					},
				},
			},
		}

		cfg := &Config{
			Providers: csync.NewMapFrom(map[string]ProviderConfig{
				"custom": {
					APIKey:  "test-key",
					BaseURL: "https://api.custom.com/v1",
					Models: []catwalk.Model{
						{
							ID:               "large-model",
							DefaultMaxTokens: 1000,
						},
					},
				},
			}),
		}
		cfg.setDefaults("/tmp", "")
		env := env.NewFromMap(map[string]string{})
		resolver := NewEnvironmentVariableResolver(env)
		err := cfg.configureProviders(env, resolver, knownProviders)
		require.NoError(t, err)
		large, small, err := cfg.defaultModelSelection(knownProviders)
		require.NoError(t, err)
		require.Equal(t, "large-model", large.Model)
		require.Equal(t, "openai", large.Provider)
		require.Equal(t, int64(1000), large.MaxTokens)
		require.Equal(t, "small-model", small.Model)
		require.Equal(t, "openai", small.Provider)
		require.Equal(t, int64(500), small.MaxTokens)
	})
}

// TestConfig_configureProvidersDisableDefaultProviders 测试禁用默认提供商的功能
func TestConfig_configureProvidersDisableDefaultProviders(t *testing.T) {
	t.Run("when enabled, ignores all default providers and requires full specification", func(t *testing.T) {
		knownProviders := []catwalk.Provider{
			{
				ID:          "openai",
				APIKey:      "$OPENAI_API_KEY",
				APIEndpoint: "https://api.openai.com/v1",
				Models: []catwalk.Model{{
					ID: "gpt-4",
				}},
			},
		}

		// 用户引用了 openai 但没有完全指定它（没有 base_url，没有
		// models）。这应该被拒绝，因为 disable_default_providers
		// 将所有提供商视为自定义提供商。
		cfg := &Config{
			Options: &Options{
				DisableDefaultProviders: true,
			},
			Providers: csync.NewMapFrom(map[string]ProviderConfig{
				"openai": {
					APIKey: "$OPENAI_API_KEY",
				},
			}),
		}
		cfg.setDefaults("/tmp", "")

		env := env.NewFromMap(map[string]string{
			"OPENAI_API_KEY": "test-key",
		})
		resolver := NewEnvironmentVariableResolver(env)
		err := cfg.configureProviders(env, resolver, knownProviders)
		require.NoError(t, err)

		// openai 不应该存在，因为它缺少 base_url 和 models。
		require.Equal(t, 0, cfg.Providers.Len())
		_, exists := cfg.Providers.Get("openai")
		require.False(t, exists, "openai should not be present without full specification")
	})

	t.Run("when enabled, fully specified providers work", func(t *testing.T) {
		knownProviders := []catwalk.Provider{
			{
				ID:          "openai",
				APIKey:      "$OPENAI_API_KEY",
				APIEndpoint: "https://api.openai.com/v1",
				Models: []catwalk.Model{{
					ID: "gpt-4",
				}},
			},
		}

		// 用户完全指定了他们的提供商。
		cfg := &Config{
			Options: &Options{
				DisableDefaultProviders: true,
			},
			Providers: csync.NewMapFrom(map[string]ProviderConfig{
				"my-llm": {
					APIKey:  "$MY_API_KEY",
					BaseURL: "https://my-llm.example.com/v1",
					Models: []catwalk.Model{{
						ID: "my-model",
					}},
				},
			}),
		}
		cfg.setDefaults("/tmp", "")

		env := env.NewFromMap(map[string]string{
			"MY_API_KEY":     "test-key",
			"OPENAI_API_KEY": "test-key",
		})
		resolver := NewEnvironmentVariableResolver(env)
		err := cfg.configureProviders(env, resolver, knownProviders)
		require.NoError(t, err)

		// 只有完全指定的提供商应该存在。
		require.Equal(t, 1, cfg.Providers.Len())
		provider, exists := cfg.Providers.Get("my-llm")
		require.True(t, exists, "my-llm should be present")
		require.Equal(t, "https://my-llm.example.com/v1", provider.BaseURL)
		require.Len(t, provider.Models, 1)

		// 默认的 openai 不应该存在。
		_, exists = cfg.Providers.Get("openai")
		require.False(t, exists, "openai should not be present")
	})

	t.Run("when disabled, includes all known providers with valid credentials", func(t *testing.T) {
		knownProviders := []catwalk.Provider{
			{
				ID:          "openai",
				APIKey:      "$OPENAI_API_KEY",
				APIEndpoint: "https://api.openai.com/v1",
				Models: []catwalk.Model{{
					ID: "gpt-4",
				}},
			},
			{
				ID:          "anthropic",
				APIKey:      "$ANTHROPIC_API_KEY",
				APIEndpoint: "https://api.anthropic.com/v1",
				Models: []catwalk.Model{{
					ID: "claude-3",
				}},
			},
		}

		// 用户只配置了 openai，两个 API 密钥都可用，但选项
		// 被禁用。
		cfg := &Config{
			Options: &Options{
				DisableDefaultProviders: false,
			},
			Providers: csync.NewMapFrom(map[string]ProviderConfig{
				"openai": {
					APIKey: "$OPENAI_API_KEY",
				},
			}),
		}
		cfg.setDefaults("/tmp", "")

		env := env.NewFromMap(map[string]string{
			"OPENAI_API_KEY":    "test-key",
			"ANTHROPIC_API_KEY": "test-key",
		})
		resolver := NewEnvironmentVariableResolver(env)
		err := cfg.configureProviders(env, resolver, knownProviders)
		require.NoError(t, err)

		// 两个提供商都应该存在。
		require.Equal(t, 2, cfg.Providers.Len())
		_, exists := cfg.Providers.Get("openai")
		require.True(t, exists, "openai should be present")
		_, exists = cfg.Providers.Get("anthropic")
		require.True(t, exists, "anthropic should be present")
	})

	t.Run("when enabled, provider missing models is rejected", func(t *testing.T) {
		cfg := &Config{
			Options: &Options{
				DisableDefaultProviders: true,
			},
			Providers: csync.NewMapFrom(map[string]ProviderConfig{
				"my-llm": {
					APIKey:  "test-key",
					BaseURL: "https://my-llm.example.com/v1",
					Models:  []catwalk.Model{}, // No models.
				},
			}),
		}
		cfg.setDefaults("/tmp", "")

		env := env.NewFromMap(map[string]string{})
		resolver := NewEnvironmentVariableResolver(env)
		err := cfg.configureProviders(env, resolver, []catwalk.Provider{})
		require.NoError(t, err)

		// 提供商应该因为缺少 models 而被拒绝。
		require.Equal(t, 0, cfg.Providers.Len())
	})

	t.Run("when enabled, provider missing base_url is rejected", func(t *testing.T) {
		cfg := &Config{
			Options: &Options{
				DisableDefaultProviders: true,
			},
			Providers: csync.NewMapFrom(map[string]ProviderConfig{
				"my-llm": {
					APIKey: "test-key",
					Models: []catwalk.Model{{ID: "model"}},
					// No BaseURL.
				},
			}),
		}
		cfg.setDefaults("/tmp", "")

		env := env.NewFromMap(map[string]string{})
		resolver := NewEnvironmentVariableResolver(env)
		err := cfg.configureProviders(env, resolver, []catwalk.Provider{})
		require.NoError(t, err)

		// 提供商应该因为缺少 base_url 而被拒绝。
		require.Equal(t, 0, cfg.Providers.Len())
	})
}

// TestConfig_setDefaultsDisableDefaultProvidersEnvVar 测试从环境变量设置禁用默认提供商选项
func TestConfig_setDefaultsDisableDefaultProvidersEnvVar(t *testing.T) {
	t.Run("sets option from environment variable", func(t *testing.T) {
		t.Setenv("CRUSH_DISABLE_DEFAULT_PROVIDERS", "true")

		cfg := &Config{}
		cfg.setDefaults("/tmp", "")

		require.True(t, cfg.Options.DisableDefaultProviders)
	})

	t.Run("does not override when env var is not set", func(t *testing.T) {
		cfg := &Config{
			Options: &Options{
				DisableDefaultProviders: true,
			},
		}
		cfg.setDefaults("/tmp", "")

		require.True(t, cfg.Options.DisableDefaultProviders)
	})
}

// TestConfig_configureSelectedModels 测试配置选定模型的功能
func TestConfig_configureSelectedModels(t *testing.T) {
	t.Run("should override defaults", func(t *testing.T) {
		knownProviders := []catwalk.Provider{
			{
				ID:                  "openai",
				APIKey:              "abc",
				DefaultLargeModelID: "large-model",
				DefaultSmallModelID: "small-model",
				Models: []catwalk.Model{
					{
						ID:               "larger-model",
						DefaultMaxTokens: 2000,
					},
					{
						ID:               "large-model",
						DefaultMaxTokens: 1000,
					},
					{
						ID:               "small-model",
						DefaultMaxTokens: 500,
					},
				},
			},
		}

		cfg := &Config{
			Models: map[SelectedModelType]SelectedModel{
				"large": {
					Model: "larger-model",
				},
			},
		}
		cfg.setDefaults("/tmp", "")
		env := env.NewFromMap(map[string]string{})
		resolver := NewEnvironmentVariableResolver(env)
		err := cfg.configureProviders(env, resolver, knownProviders)
		require.NoError(t, err)

		err = cfg.configureSelectedModels(knownProviders)
		require.NoError(t, err)
		large := cfg.Models[SelectedModelTypeLarge]
		small := cfg.Models[SelectedModelTypeSmall]
		require.Equal(t, "larger-model", large.Model)
		require.Equal(t, "openai", large.Provider)
		require.Equal(t, int64(2000), large.MaxTokens)
		require.Equal(t, "small-model", small.Model)
		require.Equal(t, "openai", small.Provider)
		require.Equal(t, int64(500), small.MaxTokens)
	})
	t.Run("should be possible to use multiple providers", func(t *testing.T) {
		knownProviders := []catwalk.Provider{
			{
				ID:                  "openai",
				APIKey:              "abc",
				DefaultLargeModelID: "large-model",
				DefaultSmallModelID: "small-model",
				Models: []catwalk.Model{
					{
						ID:               "large-model",
						DefaultMaxTokens: 1000,
					},
					{
						ID:               "small-model",
						DefaultMaxTokens: 500,
					},
				},
			},
			{
				ID:                  "anthropic",
				APIKey:              "abc",
				DefaultLargeModelID: "a-large-model",
				DefaultSmallModelID: "a-small-model",
				Models: []catwalk.Model{
					{
						ID:               "a-large-model",
						DefaultMaxTokens: 1000,
					},
					{
						ID:               "a-small-model",
						DefaultMaxTokens: 200,
					},
				},
			},
		}

		cfg := &Config{
			Models: map[SelectedModelType]SelectedModel{
				"small": {
					Model:     "a-small-model",
					Provider:  "anthropic",
					MaxTokens: 300,
				},
			},
		}
		cfg.setDefaults("/tmp", "")
		env := env.NewFromMap(map[string]string{})
		resolver := NewEnvironmentVariableResolver(env)
		err := cfg.configureProviders(env, resolver, knownProviders)
		require.NoError(t, err)

		err = cfg.configureSelectedModels(knownProviders)
		require.NoError(t, err)
		large := cfg.Models[SelectedModelTypeLarge]
		small := cfg.Models[SelectedModelTypeSmall]
		require.Equal(t, "large-model", large.Model)
		require.Equal(t, "openai", large.Provider)
		require.Equal(t, int64(1000), large.MaxTokens)
		require.Equal(t, "a-small-model", small.Model)
		require.Equal(t, "anthropic", small.Provider)
		require.Equal(t, int64(300), small.MaxTokens)
	})

	t.Run("should override the max tokens only", func(t *testing.T) {
		knownProviders := []catwalk.Provider{
			{
				ID:                  "openai",
				APIKey:              "abc",
				DefaultLargeModelID: "large-model",
				DefaultSmallModelID: "small-model",
				Models: []catwalk.Model{
					{
						ID:               "large-model",
						DefaultMaxTokens: 1000,
					},
					{
						ID:               "small-model",
						DefaultMaxTokens: 500,
					},
				},
			},
		}

		cfg := &Config{
			Models: map[SelectedModelType]SelectedModel{
				"large": {
					MaxTokens: 100,
				},
			},
		}
		cfg.setDefaults("/tmp", "")
		env := env.NewFromMap(map[string]string{})
		resolver := NewEnvironmentVariableResolver(env)
		err := cfg.configureProviders(env, resolver, knownProviders)
		require.NoError(t, err)

		err = cfg.configureSelectedModels(knownProviders)
		require.NoError(t, err)
		large := cfg.Models[SelectedModelTypeLarge]
		require.Equal(t, "large-model", large.Model)
		require.Equal(t, "openai", large.Provider)
		require.Equal(t, int64(100), large.MaxTokens)
	})
}
