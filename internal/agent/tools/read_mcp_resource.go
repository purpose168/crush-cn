package tools

import (
	"cmp"
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"strings"

	"charm.land/fantasy"
	"github.com/purpose168/crush-cn/internal/agent/tools/mcp"
	"github.com/purpose168/crush-cn/internal/config"
	"github.com/purpose168/crush-cn/internal/filepathext"
	"github.com/purpose168/crush-cn/internal/permission"
)

type ReadMCPResourceParams struct {
	MCPName string `json:"mcp_name" description:"MCP服务器名称"`
	URI     string `json:"uri" description:"要读取的资源URI"`
}

type ReadMCPResourcePermissionsParams struct {
	MCPName string `json:"mcp_name"`
	URI     string `json:"uri"`
}

const ReadMCPResourceToolName = "read_mcp_resource"

//go:embed read_mcp_resource.md
var readMCPResourceDescription []byte

// NewReadMCPResourceTool 创建一个新的读取MCP资源工具实例
// cfg: 配置对象
// permissions: 权限服务
func NewReadMCPResourceTool(cfg *config.Config, permissions permission.Service) fantasy.AgentTool {
	return fantasy.NewParallelAgentTool(
		ReadMCPResourceToolName,
		string(readMCPResourceDescription),
		func(ctx context.Context, params ReadMCPResourceParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			params.MCPName = strings.TrimSpace(params.MCPName)
			params.URI = strings.TrimSpace(params.URI)
			if params.MCPName == "" {
				return fantasy.NewTextErrorResponse("mcp_name参数是必需的"), nil
			}
			if params.URI == "" {
				return fantasy.NewTextErrorResponse("uri参数是必需的"), nil
			}

			sessionID := GetSessionFromContext(ctx)
			if sessionID == "" {
				return fantasy.ToolResponse{}, fmt.Errorf("读取MCP资源需要会话ID")
			}

			relPath := filepathext.SmartJoin(cfg.WorkingDir(), cmp.Or(params.URI, "mcp-resource"))
			p, err := permissions.Request(ctx,
				permission.CreatePermissionRequest{
					SessionID:   sessionID,
					Path:        relPath,
					ToolCallID:  call.ID,
					ToolName:    ReadMCPResourceToolName,
					Action:      "read",
					Description: fmt.Sprintf("从 %s 读取MCP资源", params.MCPName),
					Params:      ReadMCPResourcePermissionsParams(params),
				},
			)
			if err != nil {
				return fantasy.ToolResponse{}, err
			}
			if !p {
				return fantasy.ToolResponse{}, permission.ErrorPermissionDenied
			}

			contents, err := mcp.ReadResource(ctx, cfg, params.MCPName, params.URI)
			if err != nil {
				return fantasy.NewTextErrorResponse(err.Error()), nil
			}
			if len(contents) == 0 {
				return fantasy.NewTextResponse(""), nil
			}

			var textParts []string
			for _, content := range contents {
				if content == nil {
					continue
				}
				if content.Text != "" {
					textParts = append(textParts, content.Text)
					continue
				}
				if len(content.Blob) > 0 {
					textParts = append(textParts, string(content.Blob))
					continue
				}
				slog.Debug("MCP资源内容缺少text/blob", "uri", content.URI)
			}

			if len(textParts) == 0 {
				return fantasy.NewTextResponse(""), nil
			}

			return fantasy.NewTextResponse(strings.Join(textParts, "\n")), nil
		},
	)
}
