package config

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	powernapConfig "github.com/charmbracelet/x/powernap/pkg/config"
	"github.com/purpose168/crush-cn/internal/agent/hyper"
	"github.com/purpose168/crush-cn/internal/csync"
	"github.com/purpose168/crush-cn/internal/env"
	"github.com/purpose168/crush-cn/internal/fsext"
	"github.com/purpose168/crush-cn/internal/home"
	"github.com/purpose168/crush-cn/internal/log"
	"github.com/qjebbs/go-jsons"
)

const defaultCatwalkURL = "https://catwalk.charm.sh"

// Load 从默认路径加载配置
func Load(workingDir, dataDir string, debug bool) (*Config, error) {
	configPaths := lookupConfigs(workingDir)

	cfg, err := loadFromConfigPaths(configPaths)
	if err != nil {
		return nil, fmt.Errorf("从路径 %v 加载配置失败: %w", configPaths, err)
	}

	cfg.dataConfigDir = GlobalConfigData()

	cfg.setDefaults(workingDir, dataDir)

	if debug {
		cfg.Options.Debug = true
	}

	// 设置日志
	log.Setup(
		filepath.Join(cfg.Options.DataDirectory, "logs", fmt.Sprintf("%s.log", appName)),
		cfg.Options.Debug,
	)

	if !isInsideWorktree() {
		const depth = 2
		const items = 100
		slog.Warn("工作目录中未检测到git仓库，将限制文件遍历操作", "depth", depth, "items", items)
		assignIfNil(&cfg.Tools.Ls.MaxDepth, depth)
		assignIfNil(&cfg.Tools.Ls.MaxItems, items)
		assignIfNil(&cfg.Options.TUI.Completions.MaxDepth, depth)
		assignIfNil(&cfg.Options.TUI.Completions.MaxItems, items)
	}

	if isAppleTerminal() {
		slog.Warn("检测到Apple Terminal，启用透明模式")
		assignIfNil(&cfg.Options.TUI.Transparent, true)
	}

	// 加载已知提供商，从catwalk加载配置
	providers, err := Providers(cfg)
	if err != nil {
		return nil, err
	}
	cfg.knownProviders = providers

	env := env.New()
	// 配置提供商
	valueResolver := NewShellVariableResolver(env)
	cfg.resolver = valueResolver
	if err := cfg.configureProviders(env, valueResolver, cfg.knownProviders); err != nil {
		return nil, fmt.Errorf("配置提供商失败: %w", err)
	}

	if !cfg.IsConfigured() {
		slog.Warn("未配置提供商")
		return cfg, nil
	}

	if err := cfg.configureSelectedModels(cfg.knownProviders); err != nil {
		return nil, fmt.Errorf("配置选定的模型失败: %w", err)
	}
	cfg.SetupAgents()
	return cfg, nil
}

func PushPopCrushEnv() func() {
	var found []string
	for _, ev := range os.Environ() {
		if strings.HasPrefix(ev, "CRUSH_") {
			pair := strings.SplitN(ev, "=", 2)
			if len(pair) != 2 {
				continue
			}
			found = append(found, strings.TrimPrefix(pair[0], "CRUSH_"))
		}
	}
	backups := make(map[string]string)
	for _, ev := range found {
		backups[ev] = os.Getenv(ev)
	}

	for _, ev := range found {
		os.Setenv(ev, os.Getenv("CRUSH_"+ev))
	}

	restore := func() {
		for k, v := range backups {
			os.Setenv(k, v)
		}
	}
	return restore
}

func (c *Config) configureProviders(env env.Env, resolver VariableResolver, knownProviders []catwalk.Provider) error {
	knownProviderNames := make(map[string]bool)
	restore := PushPopCrushEnv()
	defer restore()

	// 当启用disable_default_providers时，完全跳过所有默认/嵌入的提供商
	// 用户必须完全指定他们想要的任何提供商
	// 我们跳转到自定义提供商验证循环，该循环统一处理所有用户配置的提供商
	if c.Options.DisableDefaultProviders {
		knownProviders = nil
	}

	for _, p := range knownProviders {
		knownProviderNames[string(p.ID)] = true
		config, configExists := c.Providers.Get(string(p.ID))
		// 如果用户配置了已知提供商，我们需要允许它覆盖几个参数
		if configExists {
			if config.BaseURL != "" {
				p.APIEndpoint = config.BaseURL
			}
			if config.APIKey != "" {
				p.APIKey = config.APIKey
			}
			if len(config.Models) > 0 {
				models := []catwalk.Model{}
				seen := make(map[string]bool)

				for _, model := range config.Models {
					if seen[model.ID] {
						continue
					}
					seen[model.ID] = true
					if model.Name == "" {
						model.Name = model.ID
					}
					models = append(models, model)
				}
				for _, model := range p.Models {
					if seen[model.ID] {
						continue
					}
					seen[model.ID] = true
					if model.Name == "" {
						model.Name = model.ID
					}
					models = append(models, model)
				}

				p.Models = models
			}
		}

		headers := map[string]string{}
		if len(p.DefaultHeaders) > 0 {
			maps.Copy(headers, p.DefaultHeaders)
		}
		if len(config.ExtraHeaders) > 0 {
			maps.Copy(headers, config.ExtraHeaders)
		}
		for k, v := range headers {
			resolved, err := resolver.ResolveValue(v)
			if err != nil {
				slog.Error("无法解析提供商头部", "err", err.Error())
				continue
			}
			headers[k] = resolved
		}
		prepared := ProviderConfig{
			ID:                 string(p.ID),
			Name:               p.Name,
			BaseURL:            p.APIEndpoint,
			APIKey:             p.APIKey,
			APIKeyTemplate:     p.APIKey, // 存储原始模板以便重新解析
			OAuthToken:         config.OAuthToken,
			Type:               p.Type,
			Disable:            config.Disable,
			SystemPromptPrefix: config.SystemPromptPrefix,
			ExtraHeaders:       headers,
			ExtraBody:          config.ExtraBody,
			ExtraParams:        make(map[string]string),
			Models:             p.Models,
		}

		switch {
		case p.ID == catwalk.InferenceProviderAnthropic && config.OAuthToken != nil:
			// Claude Code订阅不再受支持。移除以显示入门指南。
			c.RemoveConfigField("providers.anthropic")
			c.Providers.Del(string(p.ID))
			continue
		case p.ID == catwalk.InferenceProviderCopilot && config.OAuthToken != nil:
			prepared.SetupGitHubCopilot()
		}

		switch p.ID {
		// 处理需要额外配置的特定提供商
		case catwalk.InferenceProviderVertexAI:
			if !hasVertexCredentials(env) {
				if configExists {
					slog.Warn("由于缺少凭据，跳过Vertex AI提供商")
					c.Providers.Del(string(p.ID))
				}
				continue
			}
			prepared.ExtraParams["project"] = env.Get("VERTEXAI_PROJECT")
			prepared.ExtraParams["location"] = env.Get("VERTEXAI_LOCATION")
		case catwalk.InferenceProviderAzure:
			endpoint, err := resolver.ResolveValue(p.APIEndpoint)
			if err != nil || endpoint == "" {
				if configExists {
					slog.Warn("由于缺少API端点，跳过Azure提供商", "provider", p.ID, "error", err)
					c.Providers.Del(string(p.ID))
				}
				continue
			}
			prepared.BaseURL = endpoint
			prepared.ExtraParams["apiVersion"] = env.Get("AZURE_OPENAI_API_VERSION")
		case catwalk.InferenceProviderBedrock:
			if !hasAWSCredentials(env) {
				if configExists {
					slog.Warn("由于缺少AWS凭据，跳过Bedrock提供商")
					c.Providers.Del(string(p.ID))
				}
				continue
			}
			prepared.ExtraParams["region"] = env.Get("AWS_REGION")
			if prepared.ExtraParams["region"] == "" {
				prepared.ExtraParams["region"] = env.Get("AWS_DEFAULT_REGION")
			}
			for _, model := range p.Models {
				if !strings.HasPrefix(model.ID, "anthropic.") {
					return fmt.Errorf("bedrock提供商目前仅支持anthropic模型，发现: %s", model.ID)
				}
			}
		default:
			// 如果提供商的API或端点缺失，我们跳过它们
			v, err := resolver.ResolveValue(p.APIKey)
			if v == "" || err != nil {
				if configExists {
					slog.Warn("由于缺少API密钥，跳过提供商", "provider", p.ID)
					c.Providers.Del(string(p.ID))
				}
				continue
			}
		}
		c.Providers.Set(string(p.ID), prepared)
	}

	// 验证自定义提供商
	for id, providerConfig := range c.Providers.Seq2() {
		if knownProviderNames[id] {
			continue
		}

		// 确保提供商ID已设置
		providerConfig.ID = id
		if providerConfig.Name == "" {
			providerConfig.Name = id // 如果未设置，使用ID作为名称
		}
		// 如果未设置，默认为OpenAI
		if providerConfig.Type == "" {
			providerConfig.Type = catwalk.TypeOpenAICompat
		}
		if !slices.Contains(catwalk.KnownProviderTypes(), providerConfig.Type) && providerConfig.Type != hyper.Name {
			slog.Warn("由于提供商类型不受支持，跳过自定义提供商", "provider", id)
			c.Providers.Del(id)
			continue
		}

		if providerConfig.Disable {
			slog.Debug("由于禁用标志，跳过自定义提供商", "provider", id)
			c.Providers.Del(id)
			continue
		}
		if providerConfig.APIKey == "" {
			slog.Warn("提供商缺少API密钥，这对于本地提供商可能是正常的", "provider", id)
		}
		if providerConfig.BaseURL == "" {
			slog.Warn("由于缺少API端点，跳过自定义提供商", "provider", id)
			c.Providers.Del(id)
			continue
		}
		if len(providerConfig.Models) == 0 {
			slog.Warn("跳过自定义提供商，因为该提供商没有模型", "provider", id)
			c.Providers.Del(id)
			continue
		}
		apiKey, err := resolver.ResolveValue(providerConfig.APIKey)
		if apiKey == "" || err != nil {
			slog.Warn("提供商缺少API密钥，这对于本地提供商可能是正常的", "provider", id)
		}
		baseURL, err := resolver.ResolveValue(providerConfig.BaseURL)
		if baseURL == "" || err != nil {
			slog.Warn("由于缺少API端点，跳过自定义提供商", "provider", id, "error", err)
			c.Providers.Del(id)
			continue
		}

		for k, v := range providerConfig.ExtraHeaders {
			resolved, err := resolver.ResolveValue(v)
			if err != nil {
				slog.Error("无法解析提供商头部", "err", err.Error())
				continue
			}
			providerConfig.ExtraHeaders[k] = resolved
		}

		c.Providers.Set(id, providerConfig)
	}
	return nil
}

func (c *Config) setDefaults(workingDir, dataDir string) {
	c.workingDir = workingDir
	if c.Options == nil {
		c.Options = &Options{}
	}
	if c.Options.TUI == nil {
		c.Options.TUI = &TUIOptions{}
	}
	if dataDir != "" {
		c.Options.DataDirectory = dataDir
	} else if c.Options.DataDirectory == "" {
		if path, ok := fsext.LookupClosest(workingDir, defaultDataDirectory); ok {
			c.Options.DataDirectory = path
		} else {
			c.Options.DataDirectory = filepath.Join(workingDir, defaultDataDirectory)
		}
	}
	if c.Providers == nil {
		c.Providers = csync.NewMap[string, ProviderConfig]()
	}
	if c.Models == nil {
		c.Models = make(map[SelectedModelType]SelectedModel)
	}
	if c.RecentModels == nil {
		c.RecentModels = make(map[SelectedModelType][]SelectedModel)
	}
	if c.MCP == nil {
		c.MCP = make(map[string]MCPConfig)
	}
	if c.LSP == nil {
		c.LSP = make(map[string]LSPConfig)
	}

	// 将默认值应用到LSP配置
	c.applyLSPDefaults()

	// 如果默认上下文路径尚未存在，则添加它们
	c.Options.ContextPaths = append(defaultContextPaths, c.Options.ContextPaths...)
	slices.Sort(c.Options.ContextPaths)
	c.Options.ContextPaths = slices.Compact(c.Options.ContextPaths)

	// 如果尚未存在，则添加默认技能目录
	for _, dir := range GlobalSkillsDirs() {
		if !slices.Contains(c.Options.SkillsPaths, dir) {
			c.Options.SkillsPaths = append(c.Options.SkillsPaths, dir)
		}
	}

	if str, ok := os.LookupEnv("CRUSH_DISABLE_PROVIDER_AUTO_UPDATE"); ok {
		c.Options.DisableProviderAutoUpdate, _ = strconv.ParseBool(str)
	}

	if str, ok := os.LookupEnv("CRUSH_DISABLE_DEFAULT_PROVIDERS"); ok {
		c.Options.DisableDefaultProviders, _ = strconv.ParseBool(str)
	}

	if c.Options.Attribution == nil {
		c.Options.Attribution = &Attribution{
			TrailerStyle:  TrailerStyleAssistedBy,
			GeneratedWith: true,
		}
	} else if c.Options.Attribution.TrailerStyle == "" {
		// 迁移已弃用的co_authored_by或应用默认值
		if c.Options.Attribution.CoAuthoredBy != nil {
			if *c.Options.Attribution.CoAuthoredBy {
				c.Options.Attribution.TrailerStyle = TrailerStyleCoAuthoredBy
			} else {
				c.Options.Attribution.TrailerStyle = TrailerStyleNone
			}
		} else {
			c.Options.Attribution.TrailerStyle = TrailerStyleAssistedBy
		}
	}
	if c.Options.InitializeAs == "" {
		c.Options.InitializeAs = defaultInitializeAs
	}
}

// applyLSPDefaults 将powernap的默认值应用到LSP配置
func (c *Config) applyLSPDefaults() {
	// 获取powernap的默认配置
	configManager := powernapConfig.NewManager()
	configManager.LoadDefaults()

	// 将默认值应用到每个LSP配置
	for name, cfg := range c.LSP {
		// 尝试根据名称或命令名称从powernap获取默认值
		base, ok := configManager.GetServer(name)
		if !ok {
			base, ok = configManager.GetServer(cfg.Command)
			if !ok {
				continue
			}
		}
		if cfg.Options == nil {
			cfg.Options = base.Settings
		}
		if cfg.InitOptions == nil {
			cfg.InitOptions = base.InitOptions
		}
		if len(cfg.FileTypes) == 0 {
			cfg.FileTypes = base.FileTypes
		}
		if len(cfg.RootMarkers) == 0 {
			cfg.RootMarkers = base.RootMarkers
		}
		if cfg.Command == "" {
			cfg.Command = base.Command
		}
		if len(cfg.Args) == 0 {
			cfg.Args = base.Args
		}
		if len(cfg.Env) == 0 {
			cfg.Env = base.Environment
		}
		// 更新映射中的配置
		c.LSP[name] = cfg
	}
}

func (c *Config) defaultModelSelection(knownProviders []catwalk.Provider) (largeModel SelectedModel, smallModel SelectedModel, err error) {
	if len(knownProviders) == 0 && c.Providers.Len() == 0 {
		err = fmt.Errorf("未配置提供商，请至少配置一个提供商")
		return largeModel, smallModel, err
	}

	// 根据已知提供商的顺序使用第一个启用的提供商
	// 如果未找到已知提供商，则使用第一个配置的提供商
	for _, p := range knownProviders {
		providerConfig, ok := c.Providers.Get(string(p.ID))
		if !ok || providerConfig.Disable {
			continue
		}
		defaultLargeModel := c.GetModel(string(p.ID), p.DefaultLargeModelID)
		if defaultLargeModel == nil {
			err = fmt.Errorf("提供商 %s 的默认大模型 %s 未找到", p.DefaultLargeModelID, p.ID)
			return largeModel, smallModel, err
		}
		largeModel = SelectedModel{
			Provider:        string(p.ID),
			Model:           defaultLargeModel.ID,
			MaxTokens:       defaultLargeModel.DefaultMaxTokens,
			ReasoningEffort: defaultLargeModel.DefaultReasoningEffort,
		}

		defaultSmallModel := c.GetModel(string(p.ID), p.DefaultSmallModelID)
		if defaultSmallModel == nil {
			err = fmt.Errorf("提供商 %s 的默认小模型 %s 未找到", p.DefaultSmallModelID, p.ID)
			return largeModel, smallModel, err
		}
		smallModel = SelectedModel{
			Provider:        string(p.ID),
			Model:           defaultSmallModel.ID,
			MaxTokens:       defaultSmallModel.DefaultMaxTokens,
			ReasoningEffort: defaultSmallModel.DefaultReasoningEffort,
		}
		return largeModel, smallModel, err
	}

	enabledProviders := c.EnabledProviders()
	slices.SortFunc(enabledProviders, func(a, b ProviderConfig) int {
		return strings.Compare(a.ID, b.ID)
	})

	if len(enabledProviders) == 0 {
		err = fmt.Errorf("未配置提供商，请至少配置一个提供商")
		return largeModel, smallModel, err
	}

	providerConfig := enabledProviders[0]
	if len(providerConfig.Models) == 0 {
		err = fmt.Errorf("提供商 %s 没有配置模型", providerConfig.ID)
		return largeModel, smallModel, err
	}
	defaultLargeModel := c.GetModel(providerConfig.ID, providerConfig.Models[0].ID)
	largeModel = SelectedModel{
		Provider:  providerConfig.ID,
		Model:     defaultLargeModel.ID,
		MaxTokens: defaultLargeModel.DefaultMaxTokens,
	}
	defaultSmallModel := c.GetModel(providerConfig.ID, providerConfig.Models[0].ID)
	smallModel = SelectedModel{
		Provider:  providerConfig.ID,
		Model:     defaultSmallModel.ID,
		MaxTokens: defaultSmallModel.DefaultMaxTokens,
	}
	return largeModel, smallModel, err
}

func (c *Config) configureSelectedModels(knownProviders []catwalk.Provider) error {
	defaultLarge, defaultSmall, err := c.defaultModelSelection(knownProviders)
	if err != nil {
		return fmt.Errorf("选择默认模型失败: %w", err)
	}
	large, small := defaultLarge, defaultSmall

	largeModelSelected, largeModelConfigured := c.Models[SelectedModelTypeLarge]
	if largeModelConfigured {
		if largeModelSelected.Model != "" {
			large.Model = largeModelSelected.Model
		}
		if largeModelSelected.Provider != "" {
			large.Provider = largeModelSelected.Provider
		}
		model := c.GetModel(large.Provider, large.Model)
		if model == nil {
			large = defaultLarge
			// 覆盖模型类型为大模型
			err := c.UpdatePreferredModel(SelectedModelTypeLarge, large)
			if err != nil {
				return fmt.Errorf("更新首选大模型失败: %w", err)
			}
		} else {
			if largeModelSelected.MaxTokens > 0 {
				large.MaxTokens = largeModelSelected.MaxTokens
			} else {
				large.MaxTokens = model.DefaultMaxTokens
			}
			if largeModelSelected.ReasoningEffort != "" {
				large.ReasoningEffort = largeModelSelected.ReasoningEffort
			}
			large.Think = largeModelSelected.Think
			if largeModelSelected.Temperature != nil {
				large.Temperature = largeModelSelected.Temperature
			}
			if largeModelSelected.TopP != nil {
				large.TopP = largeModelSelected.TopP
			}
			if largeModelSelected.TopK != nil {
				large.TopK = largeModelSelected.TopK
			}
			if largeModelSelected.FrequencyPenalty != nil {
				large.FrequencyPenalty = largeModelSelected.FrequencyPenalty
			}
			if largeModelSelected.PresencePenalty != nil {
				large.PresencePenalty = largeModelSelected.PresencePenalty
			}
		}
	}
	smallModelSelected, smallModelConfigured := c.Models[SelectedModelTypeSmall]
	if smallModelConfigured {
		if smallModelSelected.Model != "" {
			small.Model = smallModelSelected.Model
		}
		if smallModelSelected.Provider != "" {
			small.Provider = smallModelSelected.Provider
		}

		model := c.GetModel(small.Provider, small.Model)
		if model == nil {
			small = defaultSmall
			// 覆盖模型类型为小模型
			err := c.UpdatePreferredModel(SelectedModelTypeSmall, small)
			if err != nil {
				return fmt.Errorf("更新首选小模型失败: %w", err)
			}
		} else {
			if smallModelSelected.MaxTokens > 0 {
				small.MaxTokens = smallModelSelected.MaxTokens
			} else {
				small.MaxTokens = model.DefaultMaxTokens
			}
			if smallModelSelected.ReasoningEffort != "" {
				small.ReasoningEffort = smallModelSelected.ReasoningEffort
			}
			if smallModelSelected.Temperature != nil {
				small.Temperature = smallModelSelected.Temperature
			}
			if smallModelSelected.TopP != nil {
				small.TopP = smallModelSelected.TopP
			}
			if smallModelSelected.TopK != nil {
				small.TopK = smallModelSelected.TopK
			}
			if smallModelSelected.FrequencyPenalty != nil {
				small.FrequencyPenalty = smallModelSelected.FrequencyPenalty
			}
			if smallModelSelected.PresencePenalty != nil {
				small.PresencePenalty = smallModelSelected.PresencePenalty
			}
			small.Think = smallModelSelected.Think
		}
	}
	c.Models[SelectedModelTypeLarge] = large
	c.Models[SelectedModelTypeSmall] = small
	return nil
}

// lookupConfigs 从当前工作目录向上递归搜索配置文件
func lookupConfigs(cwd string) []string {
	// 在前面添加默认配置路径
	configPaths := []string{
		GlobalConfig(),
		GlobalConfigData(),
	}

	configNames := []string{appName + ".json", "." + appName + ".json"}

	foundConfigs, err := fsext.Lookup(cwd, configNames...)
	if err != nil {
		// 至少返回默认配置
		return configPaths
	}

	// 反转顺序，使最后一个配置具有更高优先级
	slices.Reverse(foundConfigs)

	return append(configPaths, foundConfigs...)
}

func loadFromConfigPaths(configPaths []string) (*Config, error) {
	var configs [][]byte

	for _, path := range configPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("打开配置文件 %s 失败: %w", path, err)
		}
		if len(data) == 0 {
			continue
		}
		configs = append(configs, data)
	}

	return loadFromBytes(configs)
}

func loadFromBytes(configs [][]byte) (*Config, error) {
	if len(configs) == 0 {
		return &Config{}, nil
	}

	data, err := jsons.Merge(configs)
	if err != nil {
		return nil, err
	}
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func hasVertexCredentials(env env.Env) bool {
	hasProject := env.Get("VERTEXAI_PROJECT") != ""
	hasLocation := env.Get("VERTEXAI_LOCATION") != ""
	return hasProject && hasLocation
}

func hasAWSCredentials(env env.Env) bool {
	if env.Get("AWS_BEARER_TOKEN_BEDROCK") != "" {
		return true
	}

	if env.Get("AWS_ACCESS_KEY_ID") != "" && env.Get("AWS_SECRET_ACCESS_KEY") != "" {
		return true
	}

	if env.Get("AWS_PROFILE") != "" || env.Get("AWS_DEFAULT_PROFILE") != "" {
		return true
	}

	if env.Get("AWS_REGION") != "" || env.Get("AWS_DEFAULT_REGION") != "" {
		return true
	}

	if env.Get("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI") != "" ||
		env.Get("AWS_CONTAINER_CREDENTIALS_FULL_URI") != "" {
		return true
	}

	if _, err := os.Stat(filepath.Join(home.Dir(), ".aws/credentials")); err == nil && !testing.Testing() {
		return true
	}

	return false
}

// GlobalConfig 返回应用程序的全局配置文件路径
func GlobalConfig() string {
	if crushGlobal := os.Getenv("CRUSH_GLOBAL_CONFIG"); crushGlobal != "" {
		return filepath.Join(crushGlobal, fmt.Sprintf("%s.json", appName))
	}
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, appName, fmt.Sprintf("%s.json", appName))
	}
	return filepath.Join(home.Dir(), ".config", appName, fmt.Sprintf("%s.json", appName))
}

// GlobalConfigData 返回应用程序主数据目录的路径
// 当应用程序覆盖配置而不是更新全局配置时，使用此配置
func GlobalConfigData() string {
	if crushData := os.Getenv("CRUSH_GLOBAL_DATA"); crushData != "" {
		return filepath.Join(crushData, fmt.Sprintf("%s.json", appName))
	}
	if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
		return filepath.Join(xdgDataHome, appName, fmt.Sprintf("%s.json", appName))
	}

	// 返回主数据目录的路径
	// 对于Windows，应该在 `%LOCALAPPDATA%/crush/`
	// 对于Linux和macOS，应该在 `$HOME/.local/share/crush/`
	if runtime.GOOS == "windows" {
		localAppData := cmp.Or(
			os.Getenv("LOCALAPPDATA"),
			filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local"),
		)
		return filepath.Join(localAppData, appName, fmt.Sprintf("%s.json", appName))
	}

	return filepath.Join(home.Dir(), ".local", "share", appName, fmt.Sprintf("%s.json", appName))
}

func assignIfNil[T any](ptr **T, val T) {
	if *ptr == nil {
		*ptr = &val
	}
}

func isInsideWorktree() bool {
	bts, err := exec.CommandContext(
		context.Background(),
		"git", "rev-parse",
		"--is-inside-work-tree",
	).CombinedOutput()
	return err == nil && strings.TrimSpace(string(bts)) == "true"
}

// GlobalSkillsDirs 返回Agent Skills的默认目录
// 这些目录中的技能会被自动发现，它们的文件可以在没有权限提示的情况下读取
func GlobalSkillsDirs() []string {
	if crushSkills := os.Getenv("CRUSH_SKILLS_DIR"); crushSkills != "" {
		return []string{crushSkills}
	}

	// 确定基础配置目录
	var configBase string
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		configBase = xdgConfigHome
	} else if runtime.GOOS == "windows" {
		configBase = cmp.Or(
			os.Getenv("LOCALAPPDATA"),
			filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local"),
		)
	} else {
		configBase = filepath.Join(home.Dir(), ".config")
	}

	return []string{
		filepath.Join(configBase, appName, "skills"),
		filepath.Join(configBase, "agents", "skills"),
	}
}

func isAppleTerminal() bool { return os.Getenv("TERM_PROGRAM") == "Apple_Terminal" }
