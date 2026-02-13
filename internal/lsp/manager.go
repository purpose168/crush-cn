// Package lsp 提供语言服务器协议（LSP）客户端的管理器
package lsp

import (
	"cmp"
	"context"
	"errors"
	"io"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	powernapconfig "github.com/charmbracelet/x/powernap/pkg/config"
	powernap "github.com/charmbracelet/x/powernap/pkg/lsp"
	"github.com/purpose168/crush-cn/internal/config"
	"github.com/purpose168/crush-cn/internal/csync"
	"github.com/purpose168/crush-cn/internal/fsext"
	"github.com/sourcegraph/jsonrpc2"
)

// Manager 处理基于文件类型的LSP客户端延迟初始化
type Manager struct {
	clients  *csync.Map[string, *Client]  // LSP客户端映射
	cfg      *config.Config               // 配置
	manager  *powernapconfig.Manager      // powernap配置管理器
	callback func(name string, client *Client)  // 客户端启动回调
	mu       sync.Mutex                   // 互斥锁
}

// NewManager 创建新的LSP管理器服务
func NewManager(cfg *config.Config) *Manager {
	manager := powernapconfig.NewManager()
	manager.LoadDefaults()

	// 将用户配置的LSP合并到管理器中
	for name, clientConfig := range cfg.LSP {
		if clientConfig.Disabled {
			slog.Debug("LSP已被用户配置禁用", "name", name)
			manager.RemoveServer(name)
			continue
		}

		// 技巧：用户可能在配置中使用命令名而不是实际名称
		// 查找并使用正确的名称
		actualName := resolveServerName(manager, name)
		manager.AddServer(actualName, &powernapconfig.ServerConfig{
			Command:     clientConfig.Command,
			Args:        clientConfig.Args,
			Environment: clientConfig.Env,
			FileTypes:   clientConfig.FileTypes,
			RootMarkers: clientConfig.RootMarkers,
			InitOptions: clientConfig.InitOptions,
			Settings:    clientConfig.Options,
		})
	}

	return &Manager{
		clients: csync.NewMap[string, *Client](),
		cfg:     cfg,
		manager: manager,
	}
}

// Clients 返回LSP客户端映射
func (m *Manager) Clients() *csync.Map[string, *Client] {
	return m.clients
}

// SetCallback 设置当新LSP客户端成功启动时调用的回调函数
// 这允许协调器添加LSP工具
func (s *Manager) SetCallback(cb func(name string, client *Client)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callback = cb
}

// Start 启动能够处理给定文件路径的LSP服务器
// 如果适当的LSP已在运行，则此操作为空操作
func (s *Manager) Start(ctx context.Context, filePath string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var wg sync.WaitGroup
	for name, server := range s.manager.GetServers() {
		if !handles(server, filePath, s.cfg.WorkingDir()) {
			continue
		}
		wg.Go(func() {
			s.startServer(ctx, name, server)
		})
	}
	wg.Wait()
}

// skipAutoStartCommands 包含过于通用或模糊的命令列表
// 这些命令在没有明确用户配置的情况下不应自动启动
var skipAutoStartCommands = map[string]bool{
	"buck2":   true,
	"buf":     true,
	"cue":     true,
	"dart":    true,
	"deno":    true,
	"dotnet":  true,
	"dprint":  true,
	"gleam":   true,
	"java":    true,
	"julia":   true,
	"koka":    true,
	"node":    true,
	"npx":     true,
	"perl":    true,
	"plz":     true,
	"python":  true,
	"python3": true,
	"R":       true,
	"racket":  true,
	"rome":    true,
	"rubocop": true,
	"ruff":    true,
	"scarb":   true,
	"solc":    true,
	"stylua":  true,
	"swipl":   true,
	"tflint":  true,
}

// startServer 启动指定的LSP服务器
func (s *Manager) startServer(ctx context.Context, name string, server *powernapconfig.ServerConfig) {
	userConfigured := s.isUserConfigured(name)

	if !userConfigured {
		if _, err := exec.LookPath(server.Command); err != nil {
			slog.Debug("LSP服务器未安装，跳过", "name", name, "command", server.Command)
			return
		}
		if skipAutoStartCommands[server.Command] {
			slog.Debug("LSP命令过于通用，无法自动启动，跳过", "name", name, "command", server.Command)
			return
		}
	}

	cfg := s.buildConfig(name, server)
	if client, ok := s.clients.Get(name); ok {
		switch client.GetServerState() {
		case StateReady, StateStarting:
			s.callback(name, client)
			// 已完成，返回
			return
		}
	}
	client, err := New(ctx, name, cfg, s.cfg.Resolver(), s.cfg.Options.DebugLSP)
	if err != nil {
		slog.Error("创建LSP客户端失败", "name", name, "error", err)
		return
	}
	s.callback(name, client)

	defer func() {
		s.clients.Set(name, client)
		s.callback(name, client)
	}()

	initCtx, cancel := context.WithTimeout(ctx, time.Duration(cmp.Or(cfg.Timeout, 30))*time.Second)
	defer cancel()

	if _, err := client.Initialize(initCtx, s.cfg.WorkingDir()); err != nil {
		slog.Error("LSP客户端初始化失败", "name", name, "error", err)
		client.Close(ctx)
		return
	}

	if err := client.WaitForServerReady(initCtx); err != nil {
		slog.Warn("LSP服务器未完全就绪，继续执行", "name", name, "error", err)
		client.SetServerState(StateError)
	} else {
		client.SetServerState(StateReady)
	}

	slog.Debug("LSP客户端已启动", "name", name)
}

// isUserConfigured 检查指定的LSP是否由用户配置
func (s *Manager) isUserConfigured(name string) bool {
	cfg, ok := s.cfg.LSP[name]
	return ok && !cfg.Disabled
}

// buildConfig 构建LSP配置
func (s *Manager) buildConfig(name string, server *powernapconfig.ServerConfig) config.LSPConfig {
	cfg := config.LSPConfig{
		Command:     server.Command,
		Args:        server.Args,
		Env:         server.Environment,
		FileTypes:   server.FileTypes,
		RootMarkers: server.RootMarkers,
		InitOptions: server.InitOptions,
		Options:     server.Settings,
	}
	if userCfg, ok := s.cfg.LSP[name]; ok {
		cfg.Timeout = userCfg.Timeout
	}
	return cfg
}

// resolveServerName 解析服务器名称
// 如果用户使用命令名而不是实际名称，此函数会找到正确的名称
func resolveServerName(manager *powernapconfig.Manager, name string) string {
	if _, ok := manager.GetServer(name); ok {
		return name
	}
	for sname, server := range manager.GetServers() {
		if server.Command == name {
			return sname
		}
	}
	return name
}

// handlesFiletype 检查服务器是否处理指定的文件类型
func handlesFiletype(sname string, fileTypes []string, filePath string) bool {
	if len(fileTypes) == 0 {
		return true
	}

	kind := powernap.DetectLanguage(filePath)
	name := strings.ToLower(filepath.Base(filePath))
	for _, filetype := range fileTypes {
		suffix := strings.ToLower(filetype)
		if !strings.HasPrefix(suffix, ".") {
			suffix = "." + suffix
		}
		if strings.HasSuffix(name, suffix) || filetype == string(kind) {
			slog.Debug("处理文件", "name", sname, "file", name, "filetype", filetype, "kind", kind)
			return true
		}
	}

	slog.Debug("不处理文件", "name", sname, "file", name)
	return false
}

// hasRootMarkers 检查目录中是否存在根标记文件
func hasRootMarkers(dir string, markers []string) bool {
	if len(markers) == 0 {
		return true
	}
	for _, pattern := range markers {
		// 使用fsext.GlobWithDoubleStar查找匹配项
		matches, _, err := fsext.GlobWithDoubleStar(pattern, dir, 1)
		if err == nil && len(matches) > 0 {
			return true
		}
	}
	return false
}

// handles 检查服务器是否处理指定的文件
func handles(server *powernapconfig.ServerConfig, filePath, workDir string) bool {
	return handlesFiletype(server.Command, server.FileTypes, filePath) &&
		hasRootMarkers(workDir, server.RootMarkers)
}

// KillAll 强制终止所有LSP客户端
//
// 这通常比[Manager.StopAll]更快，因为它不等待服务器优雅退出，
// 但如果服务器正在写入内容，可能会导致数据丢失。
// 不过关闭Crush时这通常无关紧要。
func (s *Manager) KillAll(context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var wg sync.WaitGroup
	for name, client := range s.clients.Seq2() {
		wg.Go(func() {
			defer func() { s.callback(name, client) }()
			client.client.Kill()
			client.SetServerState(StateStopped)
			slog.Debug("已终止LSP客户端", "name", name)
		})
	}
	wg.Wait()
}

// StopAll 停止所有运行中的LSP客户端并清空客户端映射
func (s *Manager) StopAll(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var wg sync.WaitGroup
	for name, client := range s.clients.Seq2() {
		wg.Go(func() {
			defer func() { s.callback(name, client) }()
			if err := client.Close(ctx); err != nil &&
				!errors.Is(err, io.EOF) &&
				!errors.Is(err, context.Canceled) &&
				!errors.Is(err, jsonrpc2.ErrClosed) &&
				err.Error() != "signal: killed" {
				slog.Warn("停止LSP客户端失败", "name", name, "error", err)
			}
			client.SetServerState(StateStopped)
			slog.Debug("已停止LSP客户端", "name", name)
		})
	}
	wg.Wait()
}
