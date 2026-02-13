package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	powernap "github.com/charmbracelet/x/powernap/pkg/lsp"
	"github.com/charmbracelet/x/powernap/pkg/lsp/protocol"
	"github.com/charmbracelet/x/powernap/pkg/transport"
	"github.com/purpose168/crush-cn/internal/config"
	"github.com/purpose168/crush-cn/internal/csync"
	"github.com/purpose168/crush-cn/internal/home"
)

// DiagnosticCounts 按严重程度统计诊断信息的数量
type DiagnosticCounts struct {
	Error       int  // 错误数量
	Warning     int  // 警告数量
	Information int  // 信息数量
	Hint        int  // 提示数量
}

// Client LSP客户端结构体，封装了与语言服务器交互的所有功能
type Client struct {
	client *powernap.Client  // 底层powernap客户端实例
	name   string            // LSP客户端名称
	debug  bool              // 是否启用调试模式

	// 此LSP服务器的工作目录范围
	workDir string

	// 此LSP服务器处理的文件类型（如 .go, .rs, .py）
	fileTypes []string

	// 此LSP客户端的配置
	config config.LSPConfig

	// 原始上下文和解析器，用于重新创建客户端
	ctx      context.Context
	resolver config.VariableResolver

	// 诊断信息变更回调函数
	onDiagnosticsChanged func(name string, count int)

	// 诊断信息缓存
	diagnostics *csync.VersionedMap[protocol.DocumentURI, []protocol.Diagnostic]

	// 缓存的诊断计数，避免每次UI渲染时复制map
	diagCountsCache   DiagnosticCounts
	diagCountsVersion uint64
	diagCountsMu      sync.Mutex

	// 当前在LSP中打开的文件
	openFiles *csync.Map[string, *OpenFileInfo]

	// 服务器状态
	serverState atomic.Value
}

// New 使用powernap实现创建一个新的LSP客户端
// 参数:
//   - ctx: 上下文
//   - name: LSP客户端名称
//   - cfg: LSP配置
//   - resolver: 变量解析器
//   - debug: 是否启用调试模式
// 返回值: 创建的客户端实例和可能的错误
func New(ctx context.Context, name string, cfg config.LSPConfig, resolver config.VariableResolver, debug bool) (*Client, error) {
	client := &Client{
		name:        name,
		fileTypes:   cfg.FileTypes,
		diagnostics: csync.NewVersionedMap[protocol.DocumentURI, []protocol.Diagnostic](),
		openFiles:   csync.NewMap[string, *OpenFileInfo](),
		config:      cfg,
		ctx:         ctx,
		debug:       debug,
		resolver:    resolver,
	}
	client.serverState.Store(StateStarting)

	if err := client.createPowernapClient(); err != nil {
		return nil, err
	}

	return client, nil
}

// Initialize 初始化LSP客户端并返回服务器能力
// 参数:
//   - ctx: 上下文
//   - workspaceDir: 工作区目录路径
// 返回值: 初始化结果和可能的错误
func (c *Client) Initialize(ctx context.Context, workspaceDir string) (*protocol.InitializeResult, error) {
	if err := c.client.Initialize(ctx, false); err != nil {
		return nil, fmt.Errorf("初始化LSP客户端失败: %w", err)
	}

	// 将powernap能力转换为协议能力
	caps := c.client.GetCapabilities()
	protocolCaps := protocol.ServerCapabilities{
		TextDocumentSync: caps.TextDocumentSync,
		CompletionProvider: func() *protocol.CompletionOptions {
			if caps.CompletionProvider != nil {
				return &protocol.CompletionOptions{
					TriggerCharacters:   caps.CompletionProvider.TriggerCharacters,
					AllCommitCharacters: caps.CompletionProvider.AllCommitCharacters,
					ResolveProvider:     caps.CompletionProvider.ResolveProvider,
				}
			}
			return nil
		}(),
	}

	result := &protocol.InitializeResult{
		Capabilities: protocolCaps,
	}

	c.registerHandlers()

	return result, nil
}

// Kill 直接终止客户端，不执行其他操作
func (c *Client) Kill() { c.client.Kill() }

// Close 关闭客户端中所有打开的文件，然后关闭客户端
func (c *Client) Close(ctx context.Context) error {
	c.CloseAllFiles(ctx)

	// 关闭并退出客户端
	if err := c.client.Shutdown(ctx); err != nil {
		slog.Warn("关闭LSP客户端失败", "error", err)
	}

	return c.client.Exit()
}

// createPowernapClient 使用当前配置创建新的powernap客户端
func (c *Client) createPowernapClient() error {
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取工作目录失败: %w", err)
	}

	rootURI := string(protocol.URIFromPath(workDir))
	c.workDir = workDir

	command, err := c.resolver.ResolveValue(c.config.Command)
	if err != nil {
		return fmt.Errorf("无效的LSP命令: %w", err)
	}

	// 构建客户端配置
	clientConfig := powernap.ClientConfig{
		Command:     home.Long(command),
		Args:        c.config.Args,
		RootURI:     rootURI,
		Environment: maps.Clone(c.config.Env),
		Settings:    c.config.Options,
		InitOptions: c.config.InitOptions,
		WorkspaceFolders: []protocol.WorkspaceFolder{
			{
				URI:  rootURI,
				Name: filepath.Base(workDir),
			},
		},
	}

	powernapClient, err := powernap.NewClient(clientConfig)
	if err != nil {
		return fmt.Errorf("创建LSP客户端失败: %w", err)
	}

	c.client = powernapClient
	return nil
}

// registerHandlers 注册标准的LSP通知和请求处理器
func (c *Client) registerHandlers() {
	c.RegisterServerRequestHandler("workspace/applyEdit", HandleApplyEdit)
	c.RegisterServerRequestHandler("workspace/configuration", HandleWorkspaceConfiguration)
	c.RegisterServerRequestHandler("client/registerCapability", HandleRegisterCapability)
	c.RegisterNotificationHandler("window/showMessage", func(ctx context.Context, method string, params json.RawMessage) {
		if c.debug {
			HandleServerMessage(ctx, method, params)
		}
	})
	c.RegisterNotificationHandler("textDocument/publishDiagnostics", func(_ context.Context, _ string, params json.RawMessage) {
		HandleDiagnostics(c, params)
	})
}

// Restart 关闭当前LSP客户端并使用相同配置创建新的客户端
func (c *Client) Restart() error {
	var openFiles []string
	for uri := range c.openFiles.Seq2() {
		openFiles = append(openFiles, string(uri))
	}

	closeCtx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	if err := c.Close(closeCtx); err != nil {
		slog.Warn("重启时关闭客户端出错", "name", c.name, "error", err)
	}

	c.SetServerState(StateStopped)

	c.diagCountsCache = DiagnosticCounts{}
	c.diagCountsVersion = 0

	if err := c.createPowernapClient(); err != nil {
		return err
	}

	initCtx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	c.SetServerState(StateStarting)

	if err := c.client.Initialize(initCtx, false); err != nil {
		c.SetServerState(StateError)
		return fmt.Errorf("初始化LSP客户端失败: %w", err)
	}

	c.registerHandlers()

	if err := c.WaitForServerReady(initCtx); err != nil {
		slog.Error("重启后服务器未能就绪", "name", c.name, "error", err)
		c.SetServerState(StateError)
		return err
	}

	// 重新打开之前打开的文件
	for _, uri := range openFiles {
		if err := c.OpenFile(initCtx, uri); err != nil {
			slog.Warn("重启后重新打开文件失败", "file", uri, "error", err)
		}
	}
	return nil
}

// ServerState 表示LSP服务器的状态
type ServerState int

const (
	StateStopped  ServerState = iota  // 已停止
	StateStarting                      // 启动中
	StateReady                         // 已就绪
	StateError                         // 错误状态
	StateDisabled                      // 已禁用
)

// GetServerState 返回LSP服务器的当前状态
func (c *Client) GetServerState() ServerState {
	if val := c.serverState.Load(); val != nil {
		return val.(ServerState)
	}
	return StateStarting
}

// SetServerState 设置LSP服务器的当前状态
func (c *Client) SetServerState(state ServerState) {
	c.serverState.Store(state)
}

// GetName 返回LSP客户端的名称
func (c *Client) GetName() string {
	return c.name
}

// SetDiagnosticsCallback 设置诊断信息变更的回调函数
func (c *Client) SetDiagnosticsCallback(callback func(name string, count int)) {
	c.onDiagnosticsChanged = callback
}

// WaitForServerReady 等待服务器就绪
func (c *Client) WaitForServerReady(ctx context.Context) error {
	// 设置初始状态
	c.SetServerState(StateStarting)

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 尝试用简单请求ping服务器
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	if c.debug {
		slog.Debug("等待LSP服务器就绪...")
	}

	c.openKeyConfigFiles(ctx)

	for {
		select {
		case <-ctx.Done():
			c.SetServerState(StateError)
			return fmt.Errorf("等待LSP服务器就绪超时")
		case <-ticker.C:
			// 检查客户端是否正在运行
			if !c.client.IsRunning() {
				if c.debug {
					slog.Debug("LSP服务器尚未就绪", "server", c.name)
				}
				continue
			}

			// 服务器已就绪
			c.SetServerState(StateReady)
			if c.debug {
				slog.Debug("LSP服务器已就绪")
			}
			return nil
		}
	}
}

// OpenFileInfo 包含打开文件的信息
type OpenFileInfo struct {
	Version int32                // 文件版本号
	URI     protocol.DocumentURI // 文档URI
}

// HandlesFile 检查此LSP客户端是否处理给定文件
// 基于文件扩展名和是否在工作目录内进行判断
func (c *Client) HandlesFile(path string) bool {
	// 检查文件是否在工作目录内
	absPath, err := filepath.Abs(path)
	if err != nil {
		slog.Debug("无法解析路径", "name", c.name, "file", path, "error", err)
		return false
	}
	relPath, err := filepath.Rel(c.workDir, absPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		slog.Debug("文件在工作区外", "name", c.name, "file", path, "workDir", c.workDir)
		return false
	}
	return handlesFiletype(c.name, c.fileTypes, path)
}

// OpenFile 在LSP服务器中打开文件
func (c *Client) OpenFile(ctx context.Context, filepath string) error {
	if !c.HandlesFile(filepath) {
		return nil
	}

	uri := string(protocol.URIFromPath(filepath))

	if _, exists := c.openFiles.Get(uri); exists {
		return nil // 已经打开
	}

	// 跳过不存在或无法读取的文件
	content, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("读取文件错误: %w", err)
	}

	// 通知服务器打开的文档
	if err = c.client.NotifyDidOpenTextDocument(ctx, uri, string(powernap.DetectLanguage(filepath)), 1, string(content)); err != nil {
		return err
	}

	c.openFiles.Set(uri, &OpenFileInfo{
		Version: 1,
		URI:     protocol.DocumentURI(uri),
	})

	return nil
}

// NotifyChange 通知服务器文件变更
func (c *Client) NotifyChange(ctx context.Context, filepath string) error {
	uri := string(protocol.URIFromPath(filepath))

	content, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("读取文件错误: %w", err)
	}

	fileInfo, isOpen := c.openFiles.Get(uri)
	if !isOpen {
		return fmt.Errorf("无法通知未打开文件的变更: %s", filepath)
	}

	// 递增版本号
	fileInfo.Version++

	// 创建变更事件
	changes := []protocol.TextDocumentContentChangeEvent{
		{
			Value: protocol.TextDocumentContentChangeWholeDocument{
				Text: string(content),
			},
		},
	}

	return c.client.NotifyDidChangeTextDocument(ctx, uri, int(fileInfo.Version), changes)
}

// IsFileOpen 检查文件当前是否打开
func (c *Client) IsFileOpen(filepath string) bool {
	uri := string(protocol.URIFromPath(filepath))
	_, exists := c.openFiles.Get(uri)
	return exists
}

// CloseAllFiles 关闭所有当前打开的文件
func (c *Client) CloseAllFiles(ctx context.Context) {
	for uri := range c.openFiles.Seq2() {
		if c.debug {
			slog.Debug("关闭文件", "file", uri)
		}
		if err := c.client.NotifyDidCloseTextDocument(ctx, uri); err != nil {
			slog.Warn("关闭文件出错", "uri", uri, "error", err)
			continue
		}
		c.openFiles.Del(uri)
	}
}

// GetFileDiagnostics 返回特定文件的诊断信息
func (c *Client) GetFileDiagnostics(uri protocol.DocumentURI) []protocol.Diagnostic {
	diags, _ := c.diagnostics.Get(uri)
	return diags
}

// GetDiagnostics 返回所有文件的所有诊断信息
func (c *Client) GetDiagnostics() map[protocol.DocumentURI][]protocol.Diagnostic {
	return c.diagnostics.Copy()
}

// GetDiagnosticCounts 返回按严重程度缓存的诊断计数
// 使用VersionedMap版本避免每次调用时重新计算
func (c *Client) GetDiagnosticCounts() DiagnosticCounts {
	currentVersion := c.diagnostics.Version()

	c.diagCountsMu.Lock()
	defer c.diagCountsMu.Unlock()

	if currentVersion == c.diagCountsVersion {
		return c.diagCountsCache
	}

	// 重新计算计数
	counts := DiagnosticCounts{}
	for _, diags := range c.diagnostics.Seq2() {
		for _, diag := range diags {
			switch diag.Severity {
			case protocol.SeverityError:
				counts.Error++
			case protocol.SeverityWarning:
				counts.Warning++
			case protocol.SeverityInformation:
				counts.Information++
			case protocol.SeverityHint:
				counts.Hint++
			}
		}
	}

	c.diagCountsCache = counts
	c.diagCountsVersion = currentVersion
	return counts
}

// OpenFileOnDemand 仅在文件未打开时打开文件
func (c *Client) OpenFileOnDemand(ctx context.Context, filepath string) error {
	// 检查文件是否已打开
	if c.IsFileOpen(filepath) {
		return nil
	}

	// 打开文件
	return c.OpenFile(ctx, filepath)
}

// GetDiagnosticsForFile 确保文件已打开并返回其诊断信息
func (c *Client) GetDiagnosticsForFile(ctx context.Context, filepath string) ([]protocol.Diagnostic, error) {
	documentURI := protocol.URIFromPath(filepath)

	// 确保文件已打开
	if !c.IsFileOpen(filepath) {
		if err := c.OpenFile(ctx, filepath); err != nil {
			return nil, fmt.Errorf("打开文件获取诊断信息失败: %w", err)
		}

		// 给LSP服务器一点时间处理文件
		time.Sleep(100 * time.Millisecond)
	}

	// 获取诊断信息
	diagnostics, _ := c.diagnostics.Get(documentURI)

	return diagnostics, nil
}

// ClearDiagnosticsForURI 从缓存中删除特定URI的诊断信息
func (c *Client) ClearDiagnosticsForURI(uri protocol.DocumentURI) {
	c.diagnostics.Del(uri)
}

// RegisterNotificationHandler 注册通知处理器
func (c *Client) RegisterNotificationHandler(method string, handler transport.NotificationHandler) {
	c.client.RegisterNotificationHandler(method, handler)
}

// RegisterServerRequestHandler 处理服务器请求
func (c *Client) RegisterServerRequestHandler(method string, handler transport.Handler) {
	c.client.RegisterHandler(method, handler)
}

// DidChangeWatchedFiles 向服务器发送workspace/didChangeWatchedFiles通知
func (c *Client) DidChangeWatchedFiles(ctx context.Context, params protocol.DidChangeWatchedFilesParams) error {
	return c.client.NotifyDidChangeWatchedFiles(ctx, params.Changes)
}

// openKeyConfigFiles 打开有助于初始化服务器的重要配置文件
func (c *Client) openKeyConfigFiles(ctx context.Context) {
	wd, err := os.Getwd()
	if err != nil {
		return
	}

	// 尝试打开每个文件，忽略不存在的错误
	for _, file := range c.config.RootMarkers {
		file = filepath.Join(wd, file)
		if _, err := os.Stat(file); err == nil {
			// 文件存在，尝试打开
			if err := c.OpenFile(ctx, file); err != nil {
				slog.Error("打开关键配置文件失败", "file", file, "error", err)
			} else {
				slog.Debug("为初始化打开关键配置文件", "file", file)
			}
		}
	}
}

// WaitForDiagnostics 等待诊断信息变更或超时
func (c *Client) WaitForDiagnostics(ctx context.Context, d time.Duration) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(d)
	pv := c.diagnostics.Version()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timeout:
			return
		case <-ticker.C:
			if pv != c.diagnostics.Version() {
				return
			}
		}
	}
}

// FindReferences 查找给定位置符号的所有引用
// 参数:
//   - ctx: 上下文
//   - filepath: 文件路径
//   - line: 行号
//   - character: 列号
//   - includeDeclaration: 是否包含声明
// 返回值: 引用位置列表和可能的错误
// 注意: line和character应该从0开始计数
// 参见: https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#position
func (c *Client) FindReferences(ctx context.Context, filepath string, line, character int, includeDeclaration bool) ([]protocol.Location, error) {
	if err := c.OpenFileOnDemand(ctx, filepath); err != nil {
		return nil, err
	}
	// 注意: line和character应该从0开始计数
	// 参见: https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#position
	return c.client.FindReferences(ctx, filepath, line-1, character-1, includeDeclaration)
}
