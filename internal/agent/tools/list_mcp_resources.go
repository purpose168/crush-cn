package tools

import (
	"context"
	_ "embed"
	"fmt"
	"sort"
	"strings"

	"charm.land/fantasy"
	"github.com/purpose168/crush-cn/internal/agent/tools/mcp"
	"github.com/purpose168/crush-cn/internal/config"
	"github.com/purpose168/crush-cn/internal/filepathext"
	"github.com/purpose168/crush-cn/internal/permission"
)

type ListMCPResourcesParams struct {
	MCPName string `json:"mcp_name" description:"MCP服务器名称"`
}

type ListMCPResourcesPermissionsParams struct {
	MCPName string `json:"mcp_name"`
}

const ListMCPResourcesToolName = "list_mcp_resources"

//go:embed list_mcp_resources.md
var listMCPResourcesDescription []byte

// NewListMCPResourcesTool 创建一个新的列出MCP资源工具实例
// cfg: 配置对象
// permissions: 权限服务
func NewListMCPResourcesTool(cfg *config.Config, permissions permission.Service) fantasy.AgentTool {
	return fantasy.NewParallelAgentTool(
		ListMCPResourcesToolName,
		string(listMCPResourcesDescription),
		func(ctx context.Context, params ListMCPResourcesParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			params.MCPName = strings.TrimSpace(params.MCPName)
			if params.MCPName == "" {
				return fantasy.NewTextErrorResponse("mcp_name参数是必需的"), nil
			}

			sessionID := GetSessionFromContext(ctx)
			if sessionID == "" {
				return fantasy.ToolResponse{}, fmt.Errorf("列出MCP资源需要会话ID")
			}

			relPath := filepathext.SmartJoin(cfg.WorkingDir(), params.MCPName)
			p, err := permissions.Request(ctx,
				permission.CreatePermissionRequest{
					SessionID:   sessionID,
					Path:        relPath,
					ToolCallID:  call.ID,
					ToolName:    ListMCPResourcesToolName,
					Action:      "list",
					Description: fmt.Sprintf("列出来自 %s 的MCP资源", params.MCPName),
					Params:      ListMCPResourcesPermissionsParams(params),
				},
			)
			if err != nil {
				return fantasy.ToolResponse{}, err
			}
			if !p {
				return fantasy.ToolResponse{}, permission.ErrorPermissionDenied
			}

			resources, err := mcp.ListResources(ctx, cfg, params.MCPName)
			if err != nil {
				return fantasy.NewTextErrorResponse(err.Error()), nil
			}
			if len(resources) == 0 {
				return fantasy.NewTextResponse("未找到资源"), nil
			}

			lines := make([]string, 0, len(resources))
			for _, resource := range resources {
				if resource == nil {
					continue
				}
				title := resource.Title
				if title == "" {
					title = resource.Name
				}
				if title == "" {
					title = resource.URI
				}
				line := fmt.Sprintf("- %s", title)
				if resource.URI != "" {
					line = fmt.Sprintf("%s (%s)", line, resource.URI)
				}
				if resource.Description != "" {
					line = fmt.Sprintf("%s: %s", line, resource.Description)
				}
				if resource.MIMEType != "" {
					line = fmt.Sprintf("%s [mime: %s]", line, resource.MIMEType)
				}
				if resource.Size > 0 {
					line = fmt.Sprintf("%s [size: %d]", line, resource.Size)
				}
				lines = append(lines, line)
			}

			sort.Strings(lines)
			return fantasy.NewTextResponse(strings.Join(lines, "\n")), nil
		},
	)
}
