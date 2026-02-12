package tools

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"maps"
	"strings"
	"sync"

	"charm.land/fantasy"
	"github.com/purpose168/crush-cn/internal/lsp"
)

const LSPRestartToolName = "lsp_restart"

//go:embed lsp_restart.md
var lspRestartDescription []byte

type LSPRestartParams struct {
	// Name 是要重启的特定LSP客户端的可选名称
	// 如果为空，将重启所有LSP客户端
	Name string `json:"name,omitempty"`
}

// NewLSPRestartTool 创建一个新的LSP重启工具实例
// lspManager: LSP客户端管理器
func NewLSPRestartTool(lspManager *lsp.Manager) fantasy.AgentTool {
	return fantasy.NewAgentTool(
		LSPRestartToolName,
		string(lspRestartDescription),
		func(ctx context.Context, params LSPRestartParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if lspManager.Clients().Len() == 0 {
				return fantasy.NewTextErrorResponse("没有可用的LSP客户端可重启"), nil
			}

			clientsToRestart := make(map[string]*lsp.Client)
			if params.Name == "" {
				maps.Insert(clientsToRestart, lspManager.Clients().Seq2())
			} else {
				client, exists := lspManager.Clients().Get(params.Name)
				if !exists {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("未找到LSP客户端 '%s'", params.Name)), nil
				}
				clientsToRestart[params.Name] = client
			}

			var restarted []string
			var failed []string
			var mu sync.Mutex
			var wg sync.WaitGroup
			for name, client := range clientsToRestart {
				wg.Go(func() {
					if err := client.Restart(); err != nil {
						slog.Error("重启LSP客户端失败", "name", name, "error", err)
						mu.Lock()
						failed = append(failed, name)
						mu.Unlock()
						return
					}
					mu.Lock()
					restarted = append(restarted, name)
					mu.Unlock()
				})
			}

			wg.Wait()

			var output string
			if len(restarted) > 0 {
				output = fmt.Sprintf("成功重启 %d 个LSP客户端: %s\n", len(restarted), strings.Join(restarted, ", "))
			}
			if len(failed) > 0 {
				output += fmt.Sprintf("重启 %d 个LSP客户端失败: %s\n", len(failed), strings.Join(failed, ", "))
				return fantasy.NewTextErrorResponse(output), nil
			}

			return fantasy.NewTextResponse(output), nil
		})
}
