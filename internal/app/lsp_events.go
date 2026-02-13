package app

import (
	"context"
	"time"

	"github.com/purpose168/crush-cn/internal/csync"
	"github.com/purpose168/crush-cn/internal/lsp"
	"github.com/purpose168/crush-cn/internal/pubsub"
)

// LSPEventType 表示 LSP 事件的类型
type LSPEventType string

const (
	LSPEventStateChanged       LSPEventType = "state_changed"
	LSPEventDiagnosticsChanged LSPEventType = "diagnostics_changed"
)

// LSPEvent 表示 LSP 系统中的一个事件
type LSPEvent struct {
	Type            LSPEventType
	Name            string
	State           lsp.ServerState
	Error           error
	DiagnosticCount int
}

// LSPClientInfo 保存有关 LSP 客户端状态的信息
type LSPClientInfo struct {
	Name            string
	State           lsp.ServerState
	Error           error
	Client          *lsp.Client
	DiagnosticCount int
	ConnectedAt     time.Time
}

var (
	lspStates = csync.NewMap[string, LSPClientInfo]()
	lspBroker = pubsub.NewBroker[LSPEvent]()
)

// SubscribeLSPEvents 返回一个用于接收 LSP 事件的通道
func SubscribeLSPEvents(ctx context.Context) <-chan pubsub.Event[LSPEvent] {
	return lspBroker.Subscribe(ctx)
}

// GetLSPStates 返回所有 LSP 客户端的当前状态
func GetLSPStates() map[string]LSPClientInfo {
	return lspStates.Copy()
}

// GetLSPState 返回特定 LSP 客户端的状态
func GetLSPState(name string) (LSPClientInfo, bool) {
	return lspStates.Get(name)
}

// updateLSPState 更新 LSP 客户端的状态并发布事件
func updateLSPState(name string, state lsp.ServerState, err error, client *lsp.Client, diagnosticCount int) {
	info := LSPClientInfo{
		Name:            name,
		State:           state,
		Error:           err,
		Client:          client,
		DiagnosticCount: diagnosticCount,
	}
	if state == lsp.StateReady {
		info.ConnectedAt = time.Now()
	}
	lspStates.Set(name, info)

	// 发布状态变更事件
	lspBroker.Publish(pubsub.UpdatedEvent, LSPEvent{
		Type:            LSPEventStateChanged,
		Name:            name,
		State:           state,
		Error:           err,
		DiagnosticCount: diagnosticCount,
	})
}

// updateLSPDiagnostics 更新 LSP 客户端的诊断计数并发布事件
func updateLSPDiagnostics(name string, diagnosticCount int) {
	if info, exists := lspStates.Get(name); exists {
		info.DiagnosticCount = diagnosticCount
		lspStates.Set(name, info)

		// 发布诊断变更事件
		lspBroker.Publish(pubsub.UpdatedEvent, LSPEvent{
			Type:            LSPEventDiagnosticsChanged,
			Name:            name,
			State:           info.State,
			Error:           info.Error,
			DiagnosticCount: diagnosticCount,
		})
	}
}
