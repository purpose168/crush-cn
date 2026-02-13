// Package app 负责连接服务、协调代理并管理应用程序生命周期。
package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/catwalk/pkg/catwalk"
	"charm.land/fantasy"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/charmtone"
	"github.com/charmbracelet/x/term"
	"github.com/purpose168/crush-cn/internal/agent"
	"github.com/purpose168/crush-cn/internal/agent/tools/mcp"
	"github.com/purpose168/crush-cn/internal/config"
	"github.com/purpose168/crush-cn/internal/db"
	"github.com/purpose168/crush-cn/internal/event"
	"github.com/purpose168/crush-cn/internal/filetracker"
	"github.com/purpose168/crush-cn/internal/format"
	"github.com/purpose168/crush-cn/internal/history"
	"github.com/purpose168/crush-cn/internal/log"
	"github.com/purpose168/crush-cn/internal/lsp"
	"github.com/purpose168/crush-cn/internal/message"
	"github.com/purpose168/crush-cn/internal/permission"
	"github.com/purpose168/crush-cn/internal/pubsub"
	"github.com/purpose168/crush-cn/internal/session"
	"github.com/purpose168/crush-cn/internal/shell"
	"github.com/purpose168/crush-cn/internal/ui/anim"
	"github.com/purpose168/crush-cn/internal/ui/styles"
	"github.com/purpose168/crush-cn/internal/update"
	"github.com/purpose168/crush-cn/internal/version"
)

// UpdateAvailableMsg 在有新版本可用时发送。
type UpdateAvailableMsg struct {
	CurrentVersion string
	LatestVersion  string
	IsDevelopment  bool
}

type App struct {
	Sessions    session.Service
	Messages    message.Service
	History     history.Service
	Permissions permission.Service
	FileTracker filetracker.Service

	AgentCoordinator agent.Coordinator

	LSPManager *lsp.Manager

	config *config.Config

	serviceEventsWG *sync.WaitGroup
	eventsCtx       context.Context
	events          chan tea.Msg
	tuiWG           *sync.WaitGroup

	// global context and cleanup functions
	globalCtx    context.Context
	cleanupFuncs []func(context.Context) error
}

// New 初始化一个新的应用程序实例。
func New(ctx context.Context, conn *sql.DB, cfg *config.Config) (*App, error) {
	q := db.New(conn)
	sessions := session.NewService(q, conn)
	messages := message.NewService(q)
	files := history.NewService(q, conn)
	skipPermissionsRequests := cfg.Permissions != nil && cfg.Permissions.SkipRequests
	var allowedTools []string
	if cfg.Permissions != nil && cfg.Permissions.AllowedTools != nil {
		allowedTools = cfg.Permissions.AllowedTools
	}

	app := &App{
		Sessions:    sessions,
		Messages:    messages,
		History:     files,
		Permissions: permission.NewPermissionService(cfg.WorkingDir(), skipPermissionsRequests, allowedTools),
		FileTracker: filetracker.NewService(q),
		LSPManager:  lsp.NewManager(cfg),

		globalCtx: ctx,

		config: cfg,

		events:          make(chan tea.Msg, 100),
		serviceEventsWG: &sync.WaitGroup{},
		tuiWG:           &sync.WaitGroup{},
	}

	app.setupEvents()

	// Check for updates in the background.
	go app.checkForUpdates(ctx)

	go mcp.Initialize(ctx, app.Permissions, cfg)

	// cleanup database upon app shutdown
	app.cleanupFuncs = append(
		app.cleanupFuncs,
		func(context.Context) error { return conn.Close() },
		mcp.Close,
	)

	// TODO: remove the concept of agent config, most likely.
	if !cfg.IsConfigured() {
		slog.Warn("未找到代理配置")
		return app, nil
	}
	if err := app.InitCoderAgent(ctx); err != nil {
		return nil, fmt.Errorf("初始化代码代理失败: %w", err)
	}

	// Set up callback for LSP state updates.
	app.LSPManager.SetCallback(func(name string, client *lsp.Client) {
		client.SetDiagnosticsCallback(updateLSPDiagnostics)
		updateLSPState(name, client.GetServerState(), nil, client, 0)
	})

	return app, nil
}

// Config 返回应用程序配置。
func (app *App) Config() *config.Config {
	return app.config
}

// RunNonInteractive 以非交互模式运行应用程序，使用给定的提示词并输出到标准输出。
func (app *App) RunNonInteractive(ctx context.Context, output io.Writer, prompt, largeModel, smallModel string, hideSpinner bool) error {
	slog.Info("以非交互模式运行")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if largeModel != "" || smallModel != "" {
		if err := app.overrideModelsForNonInteractive(ctx, largeModel, smallModel); err != nil {
			return fmt.Errorf("覆盖模型失败: %w", err)
		}
	}

	var (
		spinner   *format.Spinner
		stdoutTTY bool
		stderrTTY bool
		stdinTTY  bool
		progress  bool
	)

	if f, ok := output.(*os.File); ok {
		stdoutTTY = term.IsTerminal(f.Fd())
	}
	stderrTTY = term.IsTerminal(os.Stderr.Fd())
	stdinTTY = term.IsTerminal(os.Stdin.Fd())
	progress = app.config.Options.Progress == nil || *app.config.Options.Progress

	if !hideSpinner && stderrTTY {
		t := styles.DefaultStyles()

		// Detect background color to set the appropriate color for the
		// spinner's 'Generating...' text. Without this, that text would be
		// unreadable in light terminals.
		hasDarkBG := true
		if f, ok := output.(*os.File); ok && stdinTTY && stdoutTTY {
			hasDarkBG = lipgloss.HasDarkBackground(os.Stdin, f)
		}
		defaultFG := lipgloss.LightDark(hasDarkBG)(charmtone.Pepper, t.FgBase)

		spinner = format.NewSpinner(ctx, cancel, anim.Settings{
			Size:        10,
			Label:       "生成中",
			LabelColor:  defaultFG,
			GradColorA:  t.Primary,
			GradColorB:  t.Secondary,
			CycleColors: true,
		})
		spinner.Start()
	}

	// Helper function to stop spinner once.
	stopSpinner := func() {
		if !hideSpinner && spinner != nil {
			spinner.Stop()
			spinner = nil
		}
	}

	// Wait for MCP initialization to complete before reading MCP tools.
	if err := mcp.WaitForInit(ctx); err != nil {
		return fmt.Errorf("等待MCP初始化失败: %w", err)
	}

	// force update of agent models before running so mcp tools are loaded
	app.AgentCoordinator.UpdateModels(ctx)

	defer stopSpinner()

	const maxPromptLengthForTitle = 100
	const titlePrefix = "非交互: "
	var titleSuffix string

	if len(prompt) > maxPromptLengthForTitle {
		titleSuffix = prompt[:maxPromptLengthForTitle] + "..."
	} else {
		titleSuffix = prompt
	}
	title := titlePrefix + titleSuffix

	sess, err := app.Sessions.Create(ctx, title)
	if err != nil {
		return fmt.Errorf("为非交互模式创建会话失败: %w", err)
	}
	slog.Info("为非交互运行创建会话", "session_id", sess.ID)

	// Automatically approve all permission requests for this non-interactive
	// session.
	app.Permissions.AutoApproveSession(sess.ID)

	type response struct {
		result *fantasy.AgentResult
		err    error
	}
	done := make(chan response, 1)

	go func(ctx context.Context, sessionID, prompt string) {
		result, err := app.AgentCoordinator.Run(ctx, sess.ID, prompt)
		if err != nil {
			done <- response{
				err: fmt.Errorf("启动代理处理流失败: %w", err),
			}
		}
		done <- response{
			result: result,
		}
	}(ctx, sess.ID, prompt)

	messageEvents := app.Messages.Subscribe(ctx)
	messageReadBytes := make(map[string]int)
	var printed bool

	defer func() {
		if progress && stderrTTY {
			_, _ = fmt.Fprintf(os.Stderr, ansi.ResetProgressBar)
		}

		// Always print a newline at the end. If output is a TTY this will
		// prevent the prompt from overwriting the last line of output.
		_, _ = fmt.Fprintln(output)
	}()

	for {
		if progress && stderrTTY {
			// HACK: Reinitialize the terminal progress bar on every iteration
			// so it doesn't get hidden by the terminal due to inactivity.
			_, _ = fmt.Fprintf(os.Stderr, ansi.SetIndeterminateProgressBar)
		}

		select {
		case result := <-done:
			stopSpinner()
			if result.err != nil {
				if errors.Is(result.err, context.Canceled) || errors.Is(result.err, agent.ErrRequestCancelled) {
					slog.Debug("非交互: 代理处理已取消", "session_id", sess.ID)
					return nil
				}
				return fmt.Errorf("代理处理失败: %w", result.err)
			}
			return nil

		case event := <-messageEvents:
			msg := event.Payload
			if msg.SessionID == sess.ID && msg.Role == message.Assistant && len(msg.Parts) > 0 {
				stopSpinner()

				content := msg.Content().String()
				readBytes := messageReadBytes[msg.ID]

				if len(content) < readBytes {
					slog.Error("非交互: 消息内容短于已读字节", "message_length", len(content), "read_bytes", readBytes)
					return fmt.Errorf("消息内容短于已读字节: %d < %d", len(content), readBytes)
				}

				part := content[readBytes:]
				// Trim leading whitespace. Sometimes the LLM includes leading
				// formatting and intentation, which we don't want here.
				if readBytes == 0 {
					part = strings.TrimLeft(part, " \t")
				}
				// Ignore initial whitespace-only messages.
				if printed || strings.TrimSpace(part) != "" {
					printed = true
					fmt.Fprint(output, part)
				}
				messageReadBytes[msg.ID] = len(content)
			}

		case <-ctx.Done():
			stopSpinner()
			return ctx.Err()
		}
	}
}

func (app *App) UpdateAgentModel(ctx context.Context) error {
	if app.AgentCoordinator == nil {
		return fmt.Errorf("代理配置缺失")
	}
	return app.AgentCoordinator.UpdateModels(ctx)
}

// overrideModelsForNonInteractive 解析模型字符串并临时覆盖模型配置，然后重建代理。
// 格式："model-name"（搜索所有提供商）或 "provider/model-name"。
// 模型匹配不区分大小写。
// 如果提供了 largeModel 但未提供 smallModel，则小型模型默认为提供商的默认小型模型。
func (app *App) overrideModelsForNonInteractive(ctx context.Context, largeModel, smallModel string) error {
	providers := app.config.Providers.Copy()

	largeMatches, smallMatches, err := findModels(providers, largeModel, smallModel)
	if err != nil {
		return err
	}

	var largeProviderID string

	// Override large model.
	if largeModel != "" {
		found, err := validateMatches(largeMatches, largeModel, "large")
		if err != nil {
			return err
		}
		largeProviderID = found.provider
		slog.Info("为非交互运行覆盖大型模型", "provider", found.provider, "model", found.modelID)
		app.config.Models[config.SelectedModelTypeLarge] = config.SelectedModel{
			Provider: found.provider,
			Model:    found.modelID,
		}
	}

	// Override small model.
	switch {
	case smallModel != "":
		found, err := validateMatches(smallMatches, smallModel, "small")
		if err != nil {
			return err
		}
		slog.Info("为非交互运行覆盖小型模型", "provider", found.provider, "model", found.modelID)
		app.config.Models[config.SelectedModelTypeSmall] = config.SelectedModel{
			Provider: found.provider,
			Model:    found.modelID,
		}

	case largeModel != "":
		// No small model specified, but large model was - use provider's default.
		smallCfg := app.GetDefaultSmallModel(largeProviderID)
		app.config.Models[config.SelectedModelTypeSmall] = smallCfg
	}

	return app.AgentCoordinator.UpdateModels(ctx)
}

// GetDefaultSmallModel 返回给定提供商的默认小型模型。如果未找到默认值，则回退到大型模型。
func (app *App) GetDefaultSmallModel(providerID string) config.SelectedModel {
	cfg := app.config
	largeModelCfg := cfg.Models[config.SelectedModelTypeLarge]

	// 在已知提供商列表中查找提供商以获取其默认小型模型。
	knownProviders, _ := config.Providers(cfg)
	var knownProvider *catwalk.Provider
	for _, p := range knownProviders {
		if string(p.ID) == providerID {
			knownProvider = &p
			break
		}
	}

	// 对于未知/本地提供商，使用大型模型作为小型模型。
	if knownProvider == nil {
		slog.Warn("为未知提供商使用大型模型作为小型模型", "provider", providerID, "model", largeModelCfg.Model)
		return largeModelCfg
	}

	defaultSmallModelID := knownProvider.DefaultSmallModelID
	model := cfg.GetModel(providerID, defaultSmallModelID)
	if model == nil {
		slog.Warn("未找到默认小型模型，使用大型模型", "provider", providerID, "model", largeModelCfg.Model)
		return largeModelCfg
	}

	slog.Info("使用提供商默认小型模型", "provider", providerID, "model", defaultSmallModelID)
	return config.SelectedModel{
		Provider:        providerID,
		Model:           defaultSmallModelID,
		MaxTokens:       model.DefaultMaxTokens,
		ReasoningEffort: model.DefaultReasoningEffort,
	}
}

func (app *App) setupEvents() {
	ctx, cancel := context.WithCancel(app.globalCtx)
	app.eventsCtx = ctx
	setupSubscriber(ctx, app.serviceEventsWG, "sessions", app.Sessions.Subscribe, app.events)
	setupSubscriber(ctx, app.serviceEventsWG, "messages", app.Messages.Subscribe, app.events)
	setupSubscriber(ctx, app.serviceEventsWG, "permissions", app.Permissions.Subscribe, app.events)
	setupSubscriber(ctx, app.serviceEventsWG, "permissions-notifications", app.Permissions.SubscribeNotifications, app.events)
	setupSubscriber(ctx, app.serviceEventsWG, "history", app.History.Subscribe, app.events)
	setupSubscriber(ctx, app.serviceEventsWG, "mcp", mcp.SubscribeEvents, app.events)
	setupSubscriber(ctx, app.serviceEventsWG, "lsp", SubscribeLSPEvents, app.events)
	cleanupFunc := func(context.Context) error {
		cancel()
		app.serviceEventsWG.Wait()
		return nil
	}
	app.cleanupFuncs = append(app.cleanupFuncs, cleanupFunc)
}

const subscriberSendTimeout = 2 * time.Second

func setupSubscriber[T any](
	ctx context.Context,
	wg *sync.WaitGroup,
	name string,
	subscriber func(context.Context) <-chan pubsub.Event[T],
	outputCh chan<- tea.Msg,
) {
	wg.Go(func() {
		subCh := subscriber(ctx)
		sendTimer := time.NewTimer(0)
		<-sendTimer.C
		defer sendTimer.Stop()

		for {
			select {
			case event, ok := <-subCh:
				if !ok {
					slog.Debug("订阅通道已关闭", "name", name)
					return
				}
				var msg tea.Msg = event
				if !sendTimer.Stop() {
					select {
					case <-sendTimer.C:
					default:
					}
				}
				sendTimer.Reset(subscriberSendTimeout)

				select {
				case outputCh <- msg:
				case <-sendTimer.C:
					slog.Debug("消息因消费者缓慢而丢弃", "name", name)
				case <-ctx.Done():
					slog.Debug("订阅已取消", "name", name)
					return
				}
			case <-ctx.Done():
				slog.Debug("订阅已取消", "name", name)
				return
			}
		}
	})
}

func (app *App) InitCoderAgent(ctx context.Context) error {
	coderAgentCfg := app.config.Agents[config.AgentCoder]
	if coderAgentCfg.ID == "" {
		return fmt.Errorf("代码代理配置缺失")
	}
	var err error
	app.AgentCoordinator, err = agent.NewCoordinator(
		ctx,
		app.config,
		app.Sessions,
		app.Messages,
		app.Permissions,
		app.History,
		app.FileTracker,
		app.LSPManager,
	)
	if err != nil {
		slog.Error("创建代码代理失败", "err", err)
		return err
	}
	return nil
}

// Subscribe 将事件作为 tea.Msgs 发送到 TUI。
func (app *App) Subscribe(program *tea.Program) {
	defer log.RecoverPanic("app.Subscribe", func() {
		slog.Info("TUI订阅 panic: 尝试优雅关闭")
		program.Quit()
	})

	app.tuiWG.Add(1)
	tuiCtx, tuiCancel := context.WithCancel(app.globalCtx)
	app.cleanupFuncs = append(app.cleanupFuncs, func(context.Context) error {
		slog.Debug("取消TUI消息处理器")
		tuiCancel()
		app.tuiWG.Wait()
		return nil
	})
	defer app.tuiWG.Done()

	for {
		select {
		case <-tuiCtx.Done():
			slog.Debug("TUI消息处理器正在关闭")
			return
		case msg, ok := <-app.events:
			if !ok {
				slog.Debug("TUI消息通道已关闭")
				return
			}
			program.Send(msg)
		}
	}
}

// Shutdown 执行应用程序的优雅关闭。
func (app *App) Shutdown() {
	start := time.Now()
	defer func() { slog.Debug("关闭耗时 " + time.Since(start).String()) }()

	// 首先，取消所有代理并等待它们完成。这必须在关闭数据库之前完成，以便代理可以完成其状态写入。
	if app.AgentCoordinator != nil {
		app.AgentCoordinator.CancelAll()
	}

	// 现在并行运行剩余的清理任务。
	var wg sync.WaitGroup

	// 所有有超时限制的清理任务共享的关闭上下文。
	shutdownCtx, cancel := context.WithTimeout(app.globalCtx, 5*time.Second)
	defer cancel()

	// 发送退出事件
	wg.Go(func() {
		event.AppExited()
	})

	// 终止所有后台 shell。
	wg.Go(func() {
		shell.GetBackgroundShellManager().KillAll(shutdownCtx)
	})

	// 关闭所有 LSP 客户端。
	wg.Go(func() {
		app.LSPManager.KillAll(shutdownCtx)
	})

	// 调用所有清理函数。
	for _, cleanup := range app.cleanupFuncs {
		if cleanup != nil {
			wg.Go(func() {
				if err := cleanup(shutdownCtx); err != nil {
					slog.Error("应用程序关闭时清理失败", "error", err)
				}
			})
		}
	}
	wg.Wait()
}

// checkForUpdates 检查可用更新。
func (app *App) checkForUpdates(ctx context.Context) {
	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	info, err := update.Check(checkCtx, version.Version, update.Default)
	if err != nil || !info.Available() {
		return
	}
	app.events <- UpdateAvailableMsg{
		CurrentVersion: info.Current,
		LatestVersion:  info.Latest,
		IsDevelopment:  info.IsDevelopment(),
	}
}
