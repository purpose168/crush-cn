package lsp

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/charmbracelet/x/powernap/pkg/lsp/protocol"
	"github.com/purpose168/crush-cn/internal/lsp/util"
)

// HandleWorkspaceConfiguration 处理工作区配置请求
// 返回空配置映射，让LSP服务器使用默认配置
func HandleWorkspaceConfiguration(_ context.Context, _ string, params json.RawMessage) (any, error) {
	return []map[string]any{{}}, nil
}

// HandleRegisterCapability 处理能力注册请求
// 当LSP服务器请求注册新能力时调用此函数
func HandleRegisterCapability(_ context.Context, _ string, params json.RawMessage) (any, error) {
	var registerParams protocol.RegistrationParams
	if err := json.Unmarshal(params, &registerParams); err != nil {
		slog.Error("解析注册参数错误", "error", err)
		return nil, err
	}

	for _, reg := range registerParams.Registrations {
		switch reg.Method {
		case "workspace/didChangeWatchedFiles":
			// 解析注册选项
			optionsJSON, err := json.Marshal(reg.RegisterOptions)
			if err != nil {
				slog.Error("序列化注册选项错误", "error", err)
				continue
			}
			var options protocol.DidChangeWatchedFilesRegistrationOptions
			if err := json.Unmarshal(optionsJSON, &options); err != nil {
				slog.Error("解析注册选项错误", "error", err)
				continue
			}
			// 存储文件监视器注册信息
			notifyFileWatchRegistration(reg.ID, options.Watchers)
		}
	}
	return nil, nil
}

// HandleApplyEdit 处理工作区编辑请求
// 当LSP服务器请求应用工作区编辑时调用此函数
func HandleApplyEdit(_ context.Context, _ string, params json.RawMessage) (any, error) {
	var edit protocol.ApplyWorkspaceEditParams
	if err := json.Unmarshal(params, &edit); err != nil {
		return nil, err
	}

	err := util.ApplyWorkspaceEdit(edit.Edit)
	if err != nil {
		slog.Error("应用工作区编辑错误", "error", err)
		return protocol.ApplyWorkspaceEditResult{Applied: false, FailureReason: err.Error()}, nil
	}

	return protocol.ApplyWorkspaceEditResult{Applied: true}, nil
}

// FileWatchRegistrationHandler 文件监视注册处理函数类型
// 当收到文件监视注册时调用此函数
type FileWatchRegistrationHandler func(id string, watchers []protocol.FileSystemWatcher)

// fileWatchHandler 保存当前的文件监视注册处理器
var fileWatchHandler FileWatchRegistrationHandler

// RegisterFileWatchHandler 设置文件监视注册的处理器
func RegisterFileWatchHandler(handler FileWatchRegistrationHandler) {
	fileWatchHandler = handler
}

// notifyFileWatchRegistration 通知处理器关于新的文件监视注册
func notifyFileWatchRegistration(id string, watchers []protocol.FileSystemWatcher) {
	if fileWatchHandler != nil {
		fileWatchHandler(id, watchers)
	}
}

// HandleServerMessage 处理服务器消息
// 根据消息类型记录不同级别的日志
func HandleServerMessage(_ context.Context, method string, params json.RawMessage) {
	var msg protocol.ShowMessageParams
	if err := json.Unmarshal(params, &msg); err != nil {
		slog.Debug("服务器消息", "type", msg.Type, "message", msg.Message)
		return
	}

	switch msg.Type {
	case protocol.Error:
		slog.Error("LSP服务器", "message", msg.Message)
	case protocol.Warning:
		slog.Warn("LSP服务器", "message", msg.Message)
	case protocol.Info:
		slog.Info("LSP服务器", "message", msg.Message)
	case protocol.Log:
		slog.Debug("LSP服务器", "message", msg.Message)
	}
}

// HandleDiagnostics 处理来自LSP服务器的诊断通知
// 更新客户端的诊断信息缓存并触发回调
func HandleDiagnostics(client *Client, params json.RawMessage) {
	var diagParams protocol.PublishDiagnosticsParams
	if err := json.Unmarshal(params, &diagParams); err != nil {
		slog.Error("解析诊断参数错误", "error", err)
		return
	}

	client.diagnostics.Set(diagParams.URI, diagParams.Diagnostics)

	// 计算诊断信息总数
	totalCount := 0
	for _, diagnostics := range client.diagnostics.Seq2() {
		totalCount += len(diagnostics)
	}

	// 如果设置了回调，则触发回调
	if client.onDiagnosticsChanged != nil {
		client.onDiagnosticsChanged(client.name, totalCount)
	}
}
