package agent

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"charm.land/catwalk/pkg/catwalk"
	"charm.land/fantasy"
	"charm.land/fantasy/providers/anthropic"
	"charm.land/fantasy/providers/openai"
	"charm.land/fantasy/providers/openaicompat"
	"charm.land/fantasy/providers/openrouter"
	"charm.land/x/vcr"
	"github.com/purpose168/crush-cn/internal/agent/prompt"
	"github.com/purpose168/crush-cn/internal/agent/tools"
	"github.com/purpose168/crush-cn/internal/config"
	"github.com/purpose168/crush-cn/internal/csync"
	"github.com/purpose168/crush-cn/internal/db"
	"github.com/purpose168/crush-cn/internal/filetracker"
	"github.com/purpose168/crush-cn/internal/history"
	"github.com/purpose168/crush-cn/internal/lsp"
	"github.com/purpose168/crush-cn/internal/message"
	"github.com/purpose168/crush-cn/internal/permission"
	"github.com/purpose168/crush-cn/internal/session"
	"github.com/stretchr/testify/require"

	_ "github.com/joho/godotenv/autoload"
)

// fakeEnv 是测试用的环境
type fakeEnv struct {
	workingDir  string // 工作目录
	sessions    session.Service // 会话服务
	messages    message.Service // 消息服务
	permissions permission.Service // 权限服务
	history     history.Service // 历史服务
	filetracker *filetracker.Service // 文件追踪服务
	lspClients  *csync.Map[string, *lsp.Client] // LSP 客户端映射
}

// builderFunc 构建语言模型的函数类型
type builderFunc func(t *testing.T, r *vcr.Recorder) (fantasy.LanguageModel, error)

// modelPair 模型对，包含大型模型和小型模型
type modelPair struct {
	name       string // 模型对名称
	largeModel builderFunc // 大型模型构建函数
	smallModel builderFunc // 小型模型构建函数
}

// anthropicBuilder 创建 Anthropic 模型构建器
func anthropicBuilder(model string) builderFunc {
	return func(t *testing.T, r *vcr.Recorder) (fantasy.LanguageModel, error) {
		provider, err := anthropic.New(
			anthropic.WithAPIKey(os.Getenv("CRUSH_ANTHROPIC_API_KEY")),
			anthropic.WithHTTPClient(&http.Client{Transport: r}),
		)
		if err != nil {
			return nil, err
		}
		return provider.LanguageModel(t.Context(), model)
	}
}

// openaiBuilder 创建 OpenAI 模型构建器
func openaiBuilder(model string) builderFunc {
	return func(t *testing.T, r *vcr.Recorder) (fantasy.LanguageModel, error) {
		provider, err := openai.New(
			openai.WithAPIKey(os.Getenv("CRUSH_OPENAI_API_KEY")),
			openai.WithHTTPClient(&http.Client{Transport: r}),
		)
		if err != nil {
			return nil, err
		}
		return provider.LanguageModel(t.Context(), model)
	}
}

// openRouterBuilder 创建 OpenRouter 模型构建器
func openRouterBuilder(model string) builderFunc {
	return func(t *testing.T, r *vcr.Recorder) (fantasy.LanguageModel, error) {
		provider, err := openrouter.New(
			openrouter.WithAPIKey(os.Getenv("CRUSH_OPENROUTER_API_KEY")),
			openrouter.WithHTTPClient(&http.Client{Transport: r}),
		)
		if err != nil {
			return nil, err
		}
		return provider.LanguageModel(t.Context(), model)
	}
}

// zAIBuilder 创建 ZAI 模型构建器
func zAIBuilder(model string) builderFunc {
	return func(t *testing.T, r *vcr.Recorder) (fantasy.LanguageModel, error) {
		provider, err := openaicompat.New(
			openaicompat.WithBaseURL("https://api.z.ai/api/coding/paas/v4"),
			openaicompat.WithAPIKey(os.Getenv("CRUSH_ZAI_API_KEY")),
			openaicompat.WithHTTPClient(&http.Client{Transport: r}),
		)
		if err != nil {
			return nil, err
		}
		return provider.LanguageModel(t.Context(), model)
	}
}

// testEnv 创建测试环境
func testEnv(t *testing.T) fakeEnv {
	workingDir := filepath.Join("/tmp/crush-test/", t.Name())
	os.RemoveAll(workingDir)

	err := os.MkdirAll(workingDir, 0o755)
	require.NoError(t, err)

	conn, err := db.Connect(t.Context(), t.TempDir())
	require.NoError(t, err)

	q := db.New(conn)
	sessions := session.NewService(q, conn)
	messages := message.NewService(q)

	permissions := permission.NewPermissionService(workingDir, true, []string{})
	history := history.NewService(q, conn)
	filetrackerService := filetracker.NewService(q)
	lspClients := csync.NewMap[string, *lsp.Client]()

	t.Cleanup(func() {
		conn.Close()
		os.RemoveAll(workingDir)
	})

	return fakeEnv{
		workingDir,
		sessions,
		messages,
		permissions,
		history,
		&filetrackerService,
		lspClients,
	}
}

// testSessionAgent 创建测试会话代理
func testSessionAgent(env fakeEnv, large, small fantasy.LanguageModel, systemPrompt string, tools ...fantasy.AgentTool) SessionAgent {
	largeModel := Model{
		Model: large,
		CatwalkCfg: catwalk.Model{
			ContextWindow:    200000,
			DefaultMaxTokens: 10000,
		},
	}
	smallModel := Model{
		Model: small,
		CatwalkCfg: catwalk.Model{
			ContextWindow:    200000,
			DefaultMaxTokens: 10000,
		},
	}
	agent := NewSessionAgent(SessionAgentOptions{largeModel, smallModel, "", systemPrompt, false, false, true, env.sessions, env.messages, tools})
	return agent
}

// coderAgent 创建编码代理
func coderAgent(r *vcr.Recorder, env fakeEnv, large, small fantasy.LanguageModel) (SessionAgent, error) {
	fixedTime := func() time.Time {
		t, _ := time.Parse("1/2/2006", "1/1/2025")
		return t
	}
	prompt, err := coderPrompt(
		prompt.WithTimeFunc(fixedTime),
		prompt.WithPlatform("linux"),
		prompt.WithWorkingDir(filepath.ToSlash(env.workingDir)),
	)
	if err != nil {
		return nil, err
	}
	cfg, err := config.Init(env.workingDir, "", false)
	if err != nil {
		return nil, err
	}

	// NOTE(@andreynering): 设置固定配置以确保磁带匹配
	// 独立于 `$HOME/.config/crush/crush.json` 上的用户配置。
	cfg.Options.Attribution = &config.Attribution{
		TrailerStyle:  "co-authored-by",
		GeneratedWith: true,
	}

	// 清除技能路径以确保测试可重复性 - 用户的技能
	// 会包含在提示中并破坏 VCR 磁带匹配。
	cfg.Options.SkillsPaths = []string{}

	// 清除 LSP 配置以确保测试可重复性 - 用户的 LSP 配置
	// 会包含在提示中并破坏 VCR 磁带匹配。
	cfg.LSP = nil

	systemPrompt, err := prompt.Build(context.TODO(), large.Provider(), large.Model(), *cfg)
	if err != nil {
		return nil, err
	}

	// 获取 bash 工具的模型名称
	modelName := large.Model() // 如果 Name 不可用，则回退到 ID
	if model := cfg.GetModel(large.Provider(), large.Model()); model != nil {
		modelName = model.Name
	}

	allTools := []fantasy.AgentTool{
		tools.NewBashTool(env.permissions, env.workingDir, cfg.Options.Attribution, modelName),
		tools.NewDownloadTool(env.permissions, env.workingDir, r.GetDefaultClient()),
		tools.NewEditTool(nil, env.permissions, env.history, *env.filetracker, env.workingDir),
		tools.NewMultiEditTool(nil, env.permissions, env.history, *env.filetracker, env.workingDir),
		tools.NewFetchTool(env.permissions, env.workingDir, r.GetDefaultClient()),
		tools.NewGlobTool(env.workingDir),
		tools.NewGrepTool(env.workingDir),
		tools.NewLsTool(env.permissions, env.workingDir, cfg.Tools.Ls),
		tools.NewSourcegraphTool(r.GetDefaultClient()),
		tools.NewViewTool(nil, env.permissions, *env.filetracker, env.workingDir),
		tools.NewWriteTool(nil, env.permissions, env.history, *env.filetracker, env.workingDir),
	}

	return testSessionAgent(env, large, small, systemPrompt, allTools...), nil
}

// createSimpleGoProject 在给定目录中创建简单的 Go 项目结构
// 创建 go.mod 文件和带有基本 hello world 程序的 main.go 文件
func createSimpleGoProject(t *testing.T, dir string) {
	goMod := `module example.com/testproject

go 1.23
`
	err := os.WriteFile(dir+"/go.mod", []byte(goMod), 0o644)
	require.NoError(t, err)

	mainGo := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	err = os.WriteFile(dir+"/main.go", []byte(mainGo), 0o644)
	require.NoError(t, err)
}
