package tools

import (
	"context"
	"fmt"

	"charm.land/fantasy"
	"github.com/purpose168/crush-cn/internal/agent/tools/mcp"
	"github.com/purpose168/crush-cn/internal/config"
	"github.com/purpose168/crush-cn/internal/permission"
)

// GetMCPTools 获取所有当前可用的MCP工具
// permissions: 权限服务
// cfg: 配置对象
// wd: 工作目录
// 返回MCP工具列表
func GetMCPTools(permissions permission.Service, cfg *config.Config, wd string) []*Tool {
	var result []*Tool
	for mcpName, tools := range mcp.Tools() {
		for _, tool := range tools {
			result = append(result, &Tool{
				mcpName:     mcpName,
				tool:        tool,
				permissions: permissions,
				workingDir:  wd,
				cfg:         cfg,
			})
		}
	}
	return result
}

// Tool 是来自MCP的工具
type Tool struct {
	mcpName         string
	tool            *mcp.Tool
	cfg             *config.Config
	permissions     permission.Service
	workingDir      string
	providerOptions fantasy.ProviderOptions
}

// SetProviderOptions 设置提供者选项
// opts: 提供者选项
func (m *Tool) SetProviderOptions(opts fantasy.ProviderOptions) {
	m.providerOptions = opts
}

// ProviderOptions 获取提供者选项
// 返回提供者选项
func (m *Tool) ProviderOptions() fantasy.ProviderOptions {
	return m.providerOptions
}

// Name 获取工具名称
// 返回工具名称
func (m *Tool) Name() string {
	return fmt.Sprintf("mcp_%s_%s", m.mcpName, m.tool.Name)
}

// MCP 获取MCP名称
// 返回MCP名称
func (m *Tool) MCP() string {
	return m.mcpName
}

// MCPToolName 获取MCP工具名称
// 返回MCP工具名称
func (m *Tool) MCPToolName() string {
	return m.tool.Name
}

// Info 获取工具信息
// 返回工具信息
func (m *Tool) Info() fantasy.ToolInfo {
	parameters := make(map[string]any)
	required := make([]string, 0)

	if input, ok := m.tool.InputSchema.(map[string]any); ok {
		if props, ok := input["properties"].(map[string]any); ok {
			parameters = props
		}
		if req, ok := input["required"].([]any); ok {
			// 将[]any转换为[]string（当元素是字符串时）
			for _, v := range req {
				if s, ok := v.(string); ok {
					required = append(required, s)
				}
			}
		} else if reqStr, ok := input["required"].([]string); ok {
			// 处理已经是[]string的情况
			required = reqStr
		}
	}

	return fantasy.ToolInfo{
		Name:        m.Name(),
		Description: m.tool.Description,
		Parameters:  parameters,
		Required:    required,
	}
}

// Run 运行工具
// ctx: 上下文对象
// params: 工具调用参数
// 返回工具响应
func (m *Tool) Run(ctx context.Context, params fantasy.ToolCall) (fantasy.ToolResponse, error) {
	sessionID := GetSessionFromContext(ctx)
	if sessionID == "" {
		return fantasy.ToolResponse{}, fmt.Errorf("创建新文件需要会话ID")
	}
	permissionDescription := fmt.Sprintf("执行 %s 并使用以下参数:", m.Info().Name)
	p, err := m.permissions.Request(ctx,
		permission.CreatePermissionRequest{
			SessionID:   sessionID,
			ToolCallID:  params.ID,
			Path:        m.workingDir,
			ToolName:    m.Info().Name,
			Action:      "execute",
			Description: permissionDescription,
			Params:      params.Input,
		},
	)
	if err != nil {
		return fantasy.ToolResponse{}, err
	}
	if !p {
		return fantasy.ToolResponse{}, permission.ErrorPermissionDenied
	}

	result, err := mcp.RunTool(ctx, m.cfg, m.mcpName, m.tool.Name, params.Input)
	if err != nil {
		return fantasy.NewTextErrorResponse(err.Error()), nil
	}

	switch result.Type {
	case "image", "media":
		if !GetSupportsImagesFromContext(ctx) {
			modelName := GetModelNameFromContext(ctx)
			return fantasy.NewTextErrorResponse(fmt.Sprintf("该模型 (%s) 不支持图像数据。", modelName)), nil
		}

		var response fantasy.ToolResponse
		if result.Type == "image" {
			response = fantasy.NewImageResponse(result.Data, result.MediaType)
		} else {
			response = fantasy.NewMediaResponse(result.Data, result.MediaType)
		}
		response.Content = result.Content
		return response, nil
	default:
		return fantasy.NewTextResponse(result.Content), nil
	}
}
