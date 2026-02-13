package config

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"charm.land/catwalk/pkg/catwalk"
	"github.com/invopop/jsonschema"
	hyperp "github.com/purpose168/crush-cn/internal/agent/hyper"
	"github.com/purpose168/crush-cn/internal/csync"
	"github.com/purpose168/crush-cn/internal/env"
	"github.com/purpose168/crush-cn/internal/oauth"
	"github.com/purpose168/crush-cn/internal/oauth/copilot"
	"github.com/purpose168/crush-cn/internal/oauth/hyper"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	appName              = "crush"
	defaultDataDirectory = ".crush"
	defaultInitializeAs  = "AGENTS.md"
)

var defaultContextPaths = []string{
	".github/copilot-instructions.md",
	".cursorrules",
	".cursor/rules/",
	"CLAUDE.md",
	"CLAUDE.local.md",
	"GEMINI.md",
	"gemini.md",
	"crush.md",
	"crush.local.md",
	"Crush.md",
	"Crush.local.md",
	"CRUSH.md",
	"CRUSH.local.md",
	"AGENTS.md",
	"agents.md",
	"Agents.md",
}

type SelectedModelType string

// String 返回 [SelectedModelType] 的字符串表示形式。
func (s SelectedModelType) String() string {
	return string(s)
}

const (
	SelectedModelTypeLarge SelectedModelType = "large"
	SelectedModelTypeSmall SelectedModelType = "small"
)

const (
	AgentCoder string = "coder"
	AgentTask  string = "task"
)

type SelectedModel struct {
	// 模型 ID，由提供者 API 使用。
	// 必填字段。
	Model string `json:"model" jsonschema:"required,description=The model ID as used by the provider API,example=gpt-4o"`
	// 模型提供者，与 providers 配置中使用的键/ID 相同。
	// 必填字段。
	Provider string `json:"provider" jsonschema:"required,description=The model provider ID that matches a key in the providers config,example=openai"`

	// 仅由使用 openai 提供者且需要设置此参数的模型使用。
	ReasoningEffort string `json:"reasoning_effort,omitempty" jsonschema:"description=Reasoning effort level for OpenAI models that support it,enum=low,enum=medium,enum=high"`

	// 由支持推理的 anthropic 模型使用，用于指示模型是否应该进行思考。
	Think bool `json:"think,omitempty" jsonschema:"description=Enable thinking mode for Anthropic models that support reasoning"`

	// 覆盖默认模型配置。
	MaxTokens        int64    `json:"max_tokens,omitempty" jsonschema:"description=Maximum number of tokens for model responses,maximum=200000,example=4096"`
	Temperature      *float64 `json:"temperature,omitempty" jsonschema:"description=Sampling temperature,minimum=0,maximum=1,example=0.7"`
	TopP             *float64 `json:"top_p,omitempty" jsonschema:"description=Top-p (nucleus) sampling parameter,minimum=0,maximum=1,example=0.9"`
	TopK             *int64   `json:"top_k,omitempty" jsonschema:"description=Top-k sampling parameter"`
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty" jsonschema:"description=Frequency penalty to reduce repetition"`
	PresencePenalty  *float64 `json:"presence_penalty,omitempty" jsonschema:"description=Presence penalty to increase topic diversity"`

	// 覆盖提供者特定的选项。
	ProviderOptions map[string]any `json:"provider_options,omitempty" jsonschema:"description=Additional provider-specific options for the model"`
}

type ProviderConfig struct {
	// 提供者的 ID。
	ID string `json:"id,omitempty" jsonschema:"description=Unique identifier for the provider,example=openai"`
	// 提供者的名称，用于显示目的。
	Name string `json:"name,omitempty" jsonschema:"description=Human-readable name for the provider,example=OpenAI"`
	// 提供者的 API 端点。
	BaseURL string `json:"base_url,omitempty" jsonschema:"description=Base URL for the provider's API,format=uri,example=https://api.openai.com/v1"`
	// 提供者类型，例如 "openai"、"anthropic" 等。如果为空，则默认为 openai。
	Type catwalk.Type `json:"type,omitempty" jsonschema:"description=Provider type that determines the API format,enum=openai,enum=openai-compat,enum=anthropic,enum=gemini,enum=azure,enum=vertexai,default=openai"`
	// 提供者的 API 密钥。
	APIKey string `json:"api_key,omitempty" jsonschema:"description=API key for authentication with the provider,example=$OPENAI_API_KEY"`
	// 解析前的原始 API 密钥模板（用于在认证错误时重新解析）。
	APIKeyTemplate string `json:"-"`
	// 使用 OAuth2 认证的提供者的 OAuthToken。
	OAuthToken *oauth.Token `json:"oauth,omitempty" jsonschema:"description=OAuth2 token for authentication with the provider"`
	// 将提供者标记为已禁用。
	Disable bool `json:"disable,omitempty" jsonschema:"description=Whether this provider is disabled,default=false"`

	// 自定义系统提示前缀。
	SystemPromptPrefix string `json:"system_prompt_prefix,omitempty" jsonschema:"description=Custom prefix to add to system prompts for this provider"`

	// 发送给提供者的每个请求的额外请求头。
	ExtraHeaders map[string]string `json:"extra_headers,omitempty" jsonschema:"description=Additional HTTP headers to send with requests"`
	// 额外的请求体
	ExtraBody map[string]any `json:"extra_body,omitempty" jsonschema:"description=Additional fields to include in request bodies, only works with openai-compatible providers"`

	ProviderOptions map[string]any `json:"provider_options,omitempty" jsonschema:"description=Additional provider-specific options for this provider"`

	// 用于向提供者传递额外参数。
	ExtraParams map[string]string `json:"-"`

	// 提供者的模型列表
	Models []catwalk.Model `json:"models,omitempty" jsonschema:"description=List of models available from this provider"`
}

// ToProvider 将 [ProviderConfig] 转换为 [catwalk.Provider]。
func (pc *ProviderConfig) ToProvider() catwalk.Provider {
	// 将配置提供者转换为 provider.Provider 格式
	provider := catwalk.Provider{
		Name:   pc.Name,
		ID:     catwalk.InferenceProvider(pc.ID),
		Models: make([]catwalk.Model, len(pc.Models)),
	}

	// 转换模型
	for i, model := range pc.Models {
		provider.Models[i] = catwalk.Model{
			ID:                     model.ID,
			Name:                   model.Name,
			CostPer1MIn:            model.CostPer1MIn,
			CostPer1MOut:           model.CostPer1MOut,
			CostPer1MInCached:      model.CostPer1MInCached,
			CostPer1MOutCached:     model.CostPer1MOutCached,
			ContextWindow:          model.ContextWindow,
			DefaultMaxTokens:       model.DefaultMaxTokens,
			CanReason:              model.CanReason,
			ReasoningLevels:        model.ReasoningLevels,
			DefaultReasoningEffort: model.DefaultReasoningEffort,
			SupportsImages:         model.SupportsImages,
		}
	}

	return provider
}

func (pc *ProviderConfig) SetupGitHubCopilot() {
	maps.Copy(pc.ExtraHeaders, copilot.Headers())
}

type MCPType string

const (
	MCPStdio MCPType = "stdio"
	MCPSSE   MCPType = "sse"
	MCPHttp  MCPType = "http"
)

type MCPConfig struct {
	Command       string            `json:"command,omitempty" jsonschema:"description=Command to execute for stdio MCP servers,example=npx"`
	Env           map[string]string `json:"env,omitempty" jsonschema:"description=Environment variables to set for the MCP server"`
	Args          []string          `json:"args,omitempty" jsonschema:"description=Arguments to pass to the MCP server command"`
	Type          MCPType           `json:"type" jsonschema:"required,description=Type of MCP connection,enum=stdio,enum=sse,enum=http,default=stdio"`
	URL           string            `json:"url,omitempty" jsonschema:"description=URL for HTTP or SSE MCP servers,format=uri,example=http://localhost:3000/mcp"`
	Disabled      bool              `json:"disabled,omitempty" jsonschema:"description=Whether this MCP server is disabled,default=false"`
	DisabledTools []string          `json:"disabled_tools,omitempty" jsonschema:"description=List of tools from this MCP server to disable,example=get-library-doc"`
	Timeout       int               `json:"timeout,omitempty" jsonschema:"description=Timeout in seconds for MCP server connections,default=15,example=30,example=60,example=120"`

	// TODO: 也许可以使其能够从环境变量获取值
	Headers map[string]string `json:"headers,omitempty" jsonschema:"description=HTTP headers for HTTP/SSE MCP servers"`
}

type LSPConfig struct {
	Disabled    bool              `json:"disabled,omitempty" jsonschema:"description=Whether this LSP server is disabled,default=false"`
	Command     string            `json:"command,omitempty" jsonschema:"description=Command to execute for the LSP server,example=gopls"`
	Args        []string          `json:"args,omitempty" jsonschema:"description=Arguments to pass to the LSP server command"`
	Env         map[string]string `json:"env,omitempty" jsonschema:"description=Environment variables to set to the LSP server command"`
	FileTypes   []string          `json:"filetypes,omitempty" jsonschema:"description=File types this LSP server handles,example=go,example=mod,example=rs,example=c,example=js,example=ts"`
	RootMarkers []string          `json:"root_markers,omitempty" jsonschema:"description=Files or directories that indicate the project root,example=go.mod,example=package.json,example=Cargo.toml"`
	InitOptions map[string]any    `json:"init_options,omitempty" jsonschema:"description=Initialization options passed to the LSP server during initialize request"`
	Options     map[string]any    `json:"options,omitempty" jsonschema:"description=LSP server-specific settings passed during initialization"`
	Timeout     int               `json:"timeout,omitempty" jsonschema:"description=Timeout in seconds for LSP server initialization,default=30,example=60,example=120"`
}

type TUIOptions struct {
	CompactMode bool   `json:"compact_mode,omitempty" jsonschema:"description=Enable compact mode for the TUI interface,default=false"`
	DiffMode    string `json:"diff_mode,omitempty" jsonschema:"description=Diff mode for the TUI interface,enum=unified,enum=split"`
	// 这里我们可以在以后添加主题或任何 TUI 相关的选项
	//

	Completions Completions `json:"completions,omitzero" jsonschema:"description=Completions UI options"`
	Transparent *bool       `json:"transparent,omitempty" jsonschema:"description=Enable transparent background for the TUI interface,default=false"`
}

// Completions 定义补全 UI 的选项。
type Completions struct {
	MaxDepth *int `json:"max_depth,omitempty" jsonschema:"description=Maximum depth for the ls tool,default=0,example=10"`
	MaxItems *int `json:"max_items,omitempty" jsonschema:"description=Maximum number of items to return for the ls tool,default=1000,example=100"`
}

func (c Completions) Limits() (depth, items int) {
	return ptrValOr(c.MaxDepth, 0), ptrValOr(c.MaxItems, 0)
}

type Permissions struct {
	AllowedTools []string `json:"allowed_tools,omitempty" jsonschema:"description=List of tools that don't require permission prompts,example=bash,example=view"` // 不需要权限提示的工具
	SkipRequests bool     `json:"-"`                                                                                                                              // 自动接受所有权限（YOLO 模式）
}

type TrailerStyle string

const (
	TrailerStyleNone         TrailerStyle = "none"
	TrailerStyleCoAuthoredBy TrailerStyle = "co-authored-by"
	TrailerStyleAssistedBy   TrailerStyle = "assisted-by"
)

type Attribution struct {
	TrailerStyle  TrailerStyle `json:"trailer_style,omitempty" jsonschema:"description=Style of attribution trailer to add to commits,enum=none,enum=co-authored-by,enum=assisted-by,default=assisted-by"`
	CoAuthoredBy  *bool        `json:"co_authored_by,omitempty" jsonschema:"description=Deprecated: use trailer_style instead"`
	GeneratedWith bool         `json:"generated_with,omitempty" jsonschema:"description=Add Generated with Crush line to commit messages and issues and PRs,default=true"`
}

// JSONSchemaExtend 在模式中将 co_authored_by 字段标记为已弃用。
func (Attribution) JSONSchemaExtend(schema *jsonschema.Schema) {
	if schema.Properties != nil {
		if prop, ok := schema.Properties.Get("co_authored_by"); ok {
			prop.Deprecated = true
		}
	}
}

type Options struct {
	ContextPaths              []string     `json:"context_paths,omitempty" jsonschema:"description=Paths to files containing context information for the AI,example=.cursorrules,example=CRUSH.md"`
	SkillsPaths               []string     `json:"skills_paths,omitempty" jsonschema:"description=Paths to directories containing Agent Skills (folders with SKILL.md files),example=~/.config/crush/skills,example=./skills"`
	TUI                       *TUIOptions  `json:"tui,omitempty" jsonschema:"description=Terminal user interface options"`
	Debug                     bool         `json:"debug,omitempty" jsonschema:"description=Enable debug logging,default=false"`
	DebugLSP                  bool         `json:"debug_lsp,omitempty" jsonschema:"description=Enable debug logging for LSP servers,default=false"`
	DisableAutoSummarize      bool         `json:"disable_auto_summarize,omitempty" jsonschema:"description=Disable automatic conversation summarization,default=false"`
	DataDirectory             string       `json:"data_directory,omitempty" jsonschema:"description=Directory for storing application data (relative to working directory),default=.crush,example=.crush"` // 相对于当前工作目录
	DisabledTools             []string     `json:"disabled_tools,omitempty" jsonschema:"description=List of built-in tools to disable and hide from the agent,example=bash,example=sourcegraph"`
	DisableProviderAutoUpdate bool         `json:"disable_provider_auto_update,omitempty" jsonschema:"description=Disable providers auto-update,default=false"`
	DisableDefaultProviders   bool         `json:"disable_default_providers,omitempty" jsonschema:"description=Ignore all default/embedded providers. When enabled, providers must be fully specified in the config file with base_url, models, and api_key - no merging with defaults occurs,default=false"`
	Attribution               *Attribution `json:"attribution,omitempty" jsonschema:"description=Attribution settings for generated content"`
	DisableMetrics            bool         `json:"disable_metrics,omitempty" jsonschema:"description=Disable sending metrics,default=false"`
	InitializeAs              string       `json:"initialize_as,omitempty" jsonschema:"description=Name of the context file to create/update during project initialization,default=AGENTS.md,example=AGENTS.md,example=CRUSH.md,example=CLAUDE.md,example=docs/LLMs.md"`
	AutoLSP                   *bool        `json:"auto_lsp,omitempty" jsonschema:"description=Automatically setup LSPs based on root markers,default=true"`
	Progress                  *bool        `json:"progress,omitempty" jsonschema:"description=Show indeterminate progress updates during long operations,default=true"`
}

type MCPs map[string]MCPConfig

type MCP struct {
	Name string    `json:"name"`
	MCP  MCPConfig `json:"mcp"`
}

func (m MCPs) Sorted() []MCP {
	sorted := make([]MCP, 0, len(m))
	for k, v := range m {
		sorted = append(sorted, MCP{
			Name: k,
			MCP:  v,
		})
	}
	slices.SortFunc(sorted, func(a, b MCP) int {
		return strings.Compare(a.Name, b.Name)
	})
	return sorted
}

type LSPs map[string]LSPConfig

type LSP struct {
	Name string    `json:"name"`
	LSP  LSPConfig `json:"lsp"`
}

func (l LSPs) Sorted() []LSP {
	sorted := make([]LSP, 0, len(l))
	for k, v := range l {
		sorted = append(sorted, LSP{
			Name: k,
			LSP:  v,
		})
	}
	slices.SortFunc(sorted, func(a, b LSP) int {
		return strings.Compare(a.Name, b.Name)
	})
	return sorted
}

func (l LSPConfig) ResolvedEnv() []string {
	return resolveEnvs(l.Env)
}

func (m MCPConfig) ResolvedEnv() []string {
	return resolveEnvs(m.Env)
}

func (m MCPConfig) ResolvedHeaders() map[string]string {
	resolver := NewShellVariableResolver(env.New())
	for e, v := range m.Headers {
		var err error
		m.Headers[e], err = resolver.ResolveValue(v)
		if err != nil {
			slog.Error("解析请求头变量时出错", "error", err, "variable", e, "value", v)
			continue
		}
	}
	return m.Headers
}

type Agent struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	// 这是智能体使用的系统提示的 ID
	Disabled bool `json:"disabled,omitempty"`

	Model SelectedModelType `json:"model" jsonschema:"required,description=The model type to use for this agent,enum=large,enum=small,default=large"`

	// 智能体可用的工具
	//  如果为 nil，则所有工具都可用
	AllowedTools []string `json:"allowed_tools,omitempty"`

	// 这告诉我们哪些 MCP 对此智能体可用
	//  如果为空，则所有 MCP 都可用
	//  字符串数组是智能体可用的 AllowedMCP 中的工具列表
	//  如果字符串数组为 nil，则 AllowedMCP 中的所有工具都可用
	AllowedMCP map[string][]string `json:"allowed_mcp,omitempty"`

	// 覆盖此智能体的上下文路径
	ContextPaths []string `json:"context_paths,omitempty"`
}

type Tools struct {
	Ls ToolLs `json:"ls,omitempty"`
}

type ToolLs struct {
	MaxDepth *int `json:"max_depth,omitempty" jsonschema:"description=Maximum depth for the ls tool,default=0,example=10"`
	MaxItems *int `json:"max_items,omitempty" jsonschema:"description=Maximum number of items to return for the ls tool,default=1000,example=100"`
}

func (t ToolLs) Limits() (depth, items int) {
	return ptrValOr(t.MaxDepth, 0), ptrValOr(t.MaxItems, 0)
}

// Config 保存 crush 的配置。
type Config struct {
	Schema string `json:"$schema,omitempty"`

	// 我们目前仅支持 large/small 作为此处的值。
	Models map[SelectedModelType]SelectedModel `json:"models,omitempty" jsonschema:"description=Model configurations for different model types,example={\"large\":{\"model\":\"gpt-4o\",\"provider\":\"openai\"}}"`

	// 最近使用的模型存储在数据目录配置中。
	RecentModels map[SelectedModelType][]SelectedModel `json:"recent_models,omitempty" jsonschema:"-"`

	// 已配置的提供者
	Providers *csync.Map[string, ProviderConfig] `json:"providers,omitempty" jsonschema:"description=AI provider configurations"`

	MCP MCPs `json:"mcp,omitempty" jsonschema:"description=Model Context Protocol server configurations"`

	LSP LSPs `json:"lsp,omitempty" jsonschema:"description=Language Server Protocol configurations"`

	Options *Options `json:"options,omitempty" jsonschema:"description=General application options"`

	Permissions *Permissions `json:"permissions,omitempty" jsonschema:"description=Permission settings for tool usage"`

	Tools Tools `json:"tools,omitempty" jsonschema:"description=Tool configurations"`

	Agents map[string]Agent `json:"-"`

	// 内部字段
	workingDir string `json:"-"`
	// TODO: 找到更好的方法来实现这一点，这可能不应该成为配置的一部分
	resolver       VariableResolver
	dataConfigDir  string             `json:"-"`
	knownProviders []catwalk.Provider `json:"-"`
}

func (c *Config) WorkingDir() string {
	return c.workingDir
}

func (c *Config) EnabledProviders() []ProviderConfig {
	var enabled []ProviderConfig
	for p := range c.Providers.Seq() {
		if !p.Disable {
			enabled = append(enabled, p)
		}
	}
	return enabled
}

// IsConfigured 如果至少配置了一个提供者，则返回 true
func (c *Config) IsConfigured() bool {
	return len(c.EnabledProviders()) > 0
}

func (c *Config) GetModel(provider, model string) *catwalk.Model {
	if providerConfig, ok := c.Providers.Get(provider); ok {
		for _, m := range providerConfig.Models {
			if m.ID == model {
				return &m
			}
		}
	}
	return nil
}

func (c *Config) GetProviderForModel(modelType SelectedModelType) *ProviderConfig {
	model, ok := c.Models[modelType]
	if !ok {
		return nil
	}
	if providerConfig, ok := c.Providers.Get(model.Provider); ok {
		return &providerConfig
	}
	return nil
}

func (c *Config) GetModelByType(modelType SelectedModelType) *catwalk.Model {
	model, ok := c.Models[modelType]
	if !ok {
		return nil
	}
	return c.GetModel(model.Provider, model.Model)
}

func (c *Config) LargeModel() *catwalk.Model {
	model, ok := c.Models[SelectedModelTypeLarge]
	if !ok {
		return nil
	}
	return c.GetModel(model.Provider, model.Model)
}

func (c *Config) SmallModel() *catwalk.Model {
	model, ok := c.Models[SelectedModelTypeSmall]
	if !ok {
		return nil
	}
	return c.GetModel(model.Provider, model.Model)
}

func (c *Config) SetCompactMode(enabled bool) error {
	if c.Options == nil {
		c.Options = &Options{}
	}
	c.Options.TUI.CompactMode = enabled
	return c.SetConfigField("options.tui.compact_mode", enabled)
}

func (c *Config) Resolve(key string) (string, error) {
	if c.resolver == nil {
		return "", fmt.Errorf("未配置变量解析器")
	}
	return c.resolver.ResolveValue(key)
}

func (c *Config) UpdatePreferredModel(modelType SelectedModelType, model SelectedModel) error {
	c.Models[modelType] = model
	if err := c.SetConfigField(fmt.Sprintf("models.%s", modelType), model); err != nil {
		return fmt.Errorf("更新首选模型失败: %w", err)
	}
	if err := c.recordRecentModel(modelType, model); err != nil {
		return err
	}
	return nil
}

func (c *Config) HasConfigField(key string) bool {
	data, err := os.ReadFile(c.dataConfigDir)
	if err != nil {
		return false
	}
	return gjson.Get(string(data), key).Exists()
}

func (c *Config) SetConfigField(key string, value any) error {
	data, err := os.ReadFile(c.dataConfigDir)
	if err != nil {
		if os.IsNotExist(err) {
			data = []byte("{}")
		} else {
			return fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	newValue, err := sjson.Set(string(data), key, value)
	if err != nil {
		return fmt.Errorf("设置配置字段 %s 失败: %w", key, err)
	}
	if err := os.MkdirAll(filepath.Dir(c.dataConfigDir), 0o755); err != nil {
		return fmt.Errorf("创建配置目录 %q 失败: %w", c.dataConfigDir, err)
	}
	if err := os.WriteFile(c.dataConfigDir, []byte(newValue), 0o600); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}
	return nil
}

func (c *Config) RemoveConfigField(key string) error {
	data, err := os.ReadFile(c.dataConfigDir)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	newValue, err := sjson.Delete(string(data), key)
	if err != nil {
		return fmt.Errorf("删除配置字段 %s 失败: %w", key, err)
	}
	if err := os.MkdirAll(filepath.Dir(c.dataConfigDir), 0o755); err != nil {
		return fmt.Errorf("创建配置目录 %q 失败: %w", c.dataConfigDir, err)
	}
	if err := os.WriteFile(c.dataConfigDir, []byte(newValue), 0o600); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}
	return nil
}

// RefreshOAuthToken 刷新给定提供者的 OAuth 令牌。
func (c *Config) RefreshOAuthToken(ctx context.Context, providerID string) error {
	providerConfig, exists := c.Providers.Get(providerID)
	if !exists {
		return fmt.Errorf("未找到提供者 %s", providerID)
	}

	if providerConfig.OAuthToken == nil {
		return fmt.Errorf("提供者 %s 没有 OAuth 令牌", providerID)
	}

	var newToken *oauth.Token
	var refreshErr error
	switch providerID {
	case string(catwalk.InferenceProviderCopilot):
		newToken, refreshErr = copilot.RefreshToken(ctx, providerConfig.OAuthToken.RefreshToken)
	case hyperp.Name:
		newToken, refreshErr = hyper.ExchangeToken(ctx, providerConfig.OAuthToken.RefreshToken)
	default:
		return fmt.Errorf("提供者 %s 不支持 OAuth 刷新", providerID)
	}
	if refreshErr != nil {
		return fmt.Errorf("刷新提供者 %s 的 OAuth 令牌失败: %w", providerID, refreshErr)
	}

	slog.Info("成功刷新 OAuth 令牌", "provider", providerID)
	providerConfig.OAuthToken = newToken
	providerConfig.APIKey = newToken.AccessToken

	switch providerID {
	case string(catwalk.InferenceProviderCopilot):
		providerConfig.SetupGitHubCopilot()
	}

	c.Providers.Set(providerID, providerConfig)

	if err := cmp.Or(
		c.SetConfigField(fmt.Sprintf("providers.%s.api_key", providerID), newToken.AccessToken),
		c.SetConfigField(fmt.Sprintf("providers.%s.oauth", providerID), newToken),
	); err != nil {
		return fmt.Errorf("持久化刷新后的令牌失败: %w", err)
	}

	return nil
}

func (c *Config) SetProviderAPIKey(providerID string, apiKey any) error {
	var providerConfig ProviderConfig
	var exists bool
	var setKeyOrToken func()

	switch v := apiKey.(type) {
	case string:
		if err := c.SetConfigField(fmt.Sprintf("providers.%s.api_key", providerID), v); err != nil {
			return fmt.Errorf("将 API 密钥保存到配置文件失败: %w", err)
		}
		setKeyOrToken = func() { providerConfig.APIKey = v }
	case *oauth.Token:
		if err := cmp.Or(
			c.SetConfigField(fmt.Sprintf("providers.%s.api_key", providerID), v.AccessToken),
			c.SetConfigField(fmt.Sprintf("providers.%s.oauth", providerID), v),
		); err != nil {
			return err
		}
		setKeyOrToken = func() {
			providerConfig.APIKey = v.AccessToken
			providerConfig.OAuthToken = v
			switch providerID {
			case string(catwalk.InferenceProviderCopilot):
				providerConfig.SetupGitHubCopilot()
			}
		}
	}

	providerConfig, exists = c.Providers.Get(providerID)
	if exists {
		setKeyOrToken()
		c.Providers.Set(providerID, providerConfig)
		return nil
	}

	var foundProvider *catwalk.Provider
	for _, p := range c.knownProviders {
		if string(p.ID) == providerID {
			foundProvider = &p
			break
		}
	}

	if foundProvider != nil {
		// 基于已知提供者创建新的提供者配置
		providerConfig = ProviderConfig{
			ID:           providerID,
			Name:         foundProvider.Name,
			BaseURL:      foundProvider.APIEndpoint,
			Type:         foundProvider.Type,
			Disable:      false,
			ExtraHeaders: make(map[string]string),
			ExtraParams:  make(map[string]string),
			Models:       foundProvider.Models,
		}
		setKeyOrToken()
	} else {
		return fmt.Errorf("在已知提供者中未找到 ID 为 %s 的提供者", providerID)
	}
	// 存储更新后的提供者配置
	c.Providers.Set(providerID, providerConfig)
	return nil
}

const maxRecentModelsPerType = 5

func (c *Config) recordRecentModel(modelType SelectedModelType, model SelectedModel) error {
	if model.Provider == "" || model.Model == "" {
		return nil
	}

	if c.RecentModels == nil {
		c.RecentModels = make(map[SelectedModelType][]SelectedModel)
	}

	eq := func(a, b SelectedModel) bool {
		return a.Provider == b.Provider && a.Model == b.Model
	}

	entry := SelectedModel{
		Provider: model.Provider,
		Model:    model.Model,
	}

	current := c.RecentModels[modelType]
	withoutCurrent := slices.DeleteFunc(slices.Clone(current), func(existing SelectedModel) bool {
		return eq(existing, entry)
	})

	updated := append([]SelectedModel{entry}, withoutCurrent...)
	if len(updated) > maxRecentModelsPerType {
		updated = updated[:maxRecentModelsPerType]
	}

	if slices.EqualFunc(current, updated, eq) {
		return nil
	}

	c.RecentModels[modelType] = updated

	if err := c.SetConfigField(fmt.Sprintf("recent_models.%s", modelType), updated); err != nil {
		return fmt.Errorf("持久化最近模型失败: %w", err)
	}

	return nil
}

func allToolNames() []string {
	return []string{
		"agent",
		"bash",
		"job_output",
		"job_kill",
		"download",
		"edit",
		"multiedit",
		"lsp_diagnostics",
		"lsp_references",
		"lsp_restart",
		"fetch",
		"agentic_fetch",
		"glob",
		"grep",
		"ls",
		"sourcegraph",
		"todos",
		"view",
		"write",
		"list_mcp_resources",
		"read_mcp_resource",
	}
}

func resolveAllowedTools(allTools []string, disabledTools []string) []string {
	if disabledTools == nil {
		return allTools
	}
	// 过滤掉已禁用的工具（排除模式）
	return filterSlice(allTools, disabledTools, false)
}

func resolveReadOnlyTools(tools []string) []string {
	readOnlyTools := []string{"glob", "grep", "ls", "sourcegraph", "view"}
	// 过滤以仅包含在 allowedtools 中的工具（包含模式）
	return filterSlice(tools, readOnlyTools, true)
}

func filterSlice(data []string, mask []string, include bool) []string {
	var filtered []string
	for _, s := range data {
		// 如果 include 为 true，我们包含在 mask 中的项
		// 如果 include 为 false，我们包含不在 mask 中的项
		if include == slices.Contains(mask, s) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func (c *Config) SetupAgents() {
	allowedTools := resolveAllowedTools(allToolNames(), c.Options.DisabledTools)

	agents := map[string]Agent{
		AgentCoder: {
			ID:           AgentCoder,
			Name:         "Coder",
			Description:  "一个帮助执行编码任务的智能体。",
			Model:        SelectedModelTypeLarge,
			ContextPaths: c.Options.ContextPaths,
			AllowedTools: allowedTools,
		},

		AgentTask: {
			ID:           AgentCoder,
			Name:         "Task",
			Description:  "一个帮助搜索上下文和查找实现细节的智能体。",
			Model:        SelectedModelTypeLarge,
			ContextPaths: c.Options.ContextPaths,
			AllowedTools: resolveReadOnlyTools(allowedTools),
			// 默认情况下没有 MCP 或 LSP
			AllowedMCP: map[string][]string{},
		},
	}
	c.Agents = agents
}

func (c *Config) Resolver() VariableResolver {
	return c.resolver
}

func (c *ProviderConfig) TestConnection(resolver VariableResolver) error {
	var (
		providerID = catwalk.InferenceProvider(c.ID)
		testURL    = ""
		headers    = make(map[string]string)
		apiKey, _  = resolver.ResolveValue(c.APIKey)
	)

	switch providerID {
	case catwalk.InferenceProviderMiniMax:
		// 注意：MiniMax 没有好的端点可以用来验证 API 密钥。
		// 让我们至少检查一下格式。
		if !strings.HasPrefix(apiKey, "sk-") {
			return fmt.Errorf("提供者 %s 的 API 密钥格式无效", c.ID)
		}
		return nil
	}

	switch c.Type {
	case catwalk.TypeOpenAI, catwalk.TypeOpenAICompat, catwalk.TypeOpenRouter:
		baseURL, _ := resolver.ResolveValue(c.BaseURL)
		baseURL = cmp.Or(baseURL, "https://api.openai.com/v1")

		switch providerID {
		case catwalk.InferenceProviderOpenRouter:
			testURL = baseURL + "/credits"
		default:
			testURL = baseURL + "/models"
		}

		headers["Authorization"] = "Bearer " + apiKey
	case catwalk.TypeAnthropic:
		baseURL, _ := resolver.ResolveValue(c.BaseURL)
		baseURL = cmp.Or(baseURL, "https://api.anthropic.com/v1")

		switch providerID {
		case catwalk.InferenceKimiCoding:
			testURL = baseURL + "/v1/models"
		default:
			testURL = baseURL + "/models"
		}

		headers["x-api-key"] = apiKey
		headers["anthropic-version"] = "2023-06-01"
	case catwalk.TypeGoogle:
		baseURL, _ := resolver.ResolveValue(c.BaseURL)
		baseURL = cmp.Or(baseURL, "https://generativelanguage.googleapis.com")
		testURL = baseURL + "/v1beta/models?key=" + url.QueryEscape(apiKey)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", testURL, nil)
	if err != nil {
		return fmt.Errorf("为提供者 %s 创建请求失败: %w", c.ID, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	for k, v := range c.ExtraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("为提供者 %s 创建请求失败: %w", c.ID, err)
	}
	defer resp.Body.Close()

	switch providerID {
	case catwalk.InferenceProviderZAI:
		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("连接提供者 %s 失败: %s", c.ID, resp.Status)
		}
	default:
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("连接提供者 %s 失败: %s", c.ID, resp.Status)
		}
	}
	return nil
}

func resolveEnvs(envs map[string]string) []string {
	resolver := NewShellVariableResolver(env.New())
	for e, v := range envs {
		var err error
		envs[e], err = resolver.ResolveValue(v)
		if err != nil {
			slog.Error("解析环境变量时出错", "error", err, "variable", e, "value", v)
			continue
		}
	}

	res := make([]string, 0, len(envs))
	for k, v := range envs {
		res = append(res, fmt.Sprintf("%s=%s", k, v))
	}
	return res
}

func ptrValOr[T any](t *T, el T) T {
	if t == nil {
		return el
	}
	return *t
}
