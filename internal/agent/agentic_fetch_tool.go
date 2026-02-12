package agent

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"charm.land/fantasy"

	"github.com/purpose168/crush-cn/internal/agent/prompt"
	"github.com/purpose168/crush-cn/internal/agent/tools"
	"github.com/purpose168/crush-cn/internal/permission"
)

//go:embed templates/agentic_fetch.md
var agenticFetchToolDescription []byte

// agenticFetchValidationResult 保存从工具调用上下文中验证后的参数。
type agenticFetchValidationResult struct {
	SessionID      string
	AgentMessageID string
}

// validateAgenticFetchParams 验证工具调用参数并提取所需的上下文值。
func validateAgenticFetchParams(ctx context.Context, params tools.AgenticFetchParams) (agenticFetchValidationResult, error) {
	if params.Prompt == "" {
		return agenticFetchValidationResult{}, errors.New("提示词是必需的")
	}

	sessionID := tools.GetSessionFromContext(ctx)
	if sessionID == "" {
		return agenticFetchValidationResult{}, errors.New("上下文中缺少会话 ID")
	}

	agentMessageID := tools.GetMessageFromContext(ctx)
	if agentMessageID == "" {
		return agenticFetchValidationResult{}, errors.New("上下文中缺少代理消息 ID")
	}

	return agenticFetchValidationResult{
		SessionID:      sessionID,
		AgentMessageID: agentMessageID,
	}, nil
}

//go:embed templates/agentic_fetch_prompt.md.tpl
var agenticFetchPromptTmpl []byte

// agenticFetchTool 创建一个用于获取和分析网络内容的代理工具。
func (c *coordinator) agenticFetchTool(_ context.Context, client *http.Client) (fantasy.AgentTool, error) {
	if client == nil {
		// 如果没有提供 HTTP 客户端，创建一个带有合理配置的客户端
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.MaxIdleConns = 100
		transport.MaxIdleConnsPerHost = 10
		transport.IdleConnTimeout = 90 * time.Second

		client = &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		}
	}

	return fantasy.NewParallelAgentTool(
		tools.AgenticFetchToolName,
		string(agenticFetchToolDescription),
		func(ctx context.Context, params tools.AgenticFetchParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			// 验证工具调用参数
			validationResult, err := validateAgenticFetchParams(ctx, params)
			if err != nil {
				return fantasy.NewTextErrorResponse(err.Error()), nil
			}

			// 根据模式确定描述
			var description string
			if params.URL != "" {
				description = fmt.Sprintf("获取并分析来自 URL 的内容: %s", params.URL)
			} else {
				description = "搜索网络并分析结果"
			}

			// 请求权限
			p, err := c.permissions.Request(ctx,
				permission.CreatePermissionRequest{
					SessionID:   validationResult.SessionID,
					Path:        c.cfg.WorkingDir(),
					ToolCallID:  call.ID,
					ToolName:    tools.AgenticFetchToolName,
					Action:      "fetch",
					Description: description,
					Params:      tools.AgenticFetchPermissionsParams(params),
				},
			)
			if err != nil {
				return fantasy.ToolResponse{}, err
			}
			if !p {
				return fantasy.ToolResponse{}, permission.ErrorPermissionDenied
			}

			// 创建临时目录用于存储获取的内容
			tmpDir, err := os.MkdirTemp(c.cfg.Options.DataDirectory, "crush-fetch-*")
			if err != nil {
				return fantasy.NewTextErrorResponse(fmt.Sprintf("创建临时目录失败: %s", err)), nil
			}
			defer os.RemoveAll(tmpDir)

			var fullPrompt string

			if params.URL != "" {
				// URL 模式: 先获取 URL 内容
				content, err := tools.FetchURLAndConvert(ctx, client, params.URL)
				if err != nil {
					return fantasy.NewTextErrorResponse(fmt.Sprintf("获取 URL 失败: %s", err)), nil
				}

				hasLargeContent := len(content) > tools.LargeContentThreshold

				if hasLargeContent {
					// 内容过大，保存到临时文件
					tempFile, err := os.CreateTemp(tmpDir, "page-*.md")
					if err != nil {
						return fantasy.NewTextErrorResponse(fmt.Sprintf("创建临时文件失败: %s", err)), nil
					}
					tempFilePath := tempFile.Name()

					if _, err := tempFile.WriteString(content); err != nil {
						tempFile.Close()
						return fantasy.NewTextErrorResponse(fmt.Sprintf("写入内容到文件失败: %s", err)), nil
					}
					tempFile.Close()

					fullPrompt = fmt.Sprintf("%s\n\n来自 %s 的网页已保存到: %s\n\n使用 view 和 grep 工具分析此文件并提取请求的信息。", params.Prompt, params.URL, tempFilePath)
				} else {
					// 内容适中，直接包含在提示词中
					fullPrompt = fmt.Sprintf("%s\n\n网页 URL: %s\n\n<webpage_content>\n%s\n</webpage_content>", params.Prompt, params.URL, content)
				}
			} else {
				// 搜索模式: 让子代理根据需要进行搜索和获取
				fullPrompt = fmt.Sprintf("%s\n\n使用 web_search 工具查找相关信息。如果需要，将问题分解为更小、更集中的搜索。搜索后，使用 web_fetch 从最相关的结果中获取详细内容。", params.Prompt)
			}

			// 创建提示词选项
			promptOpts := []prompt.Option{
				prompt.WithWorkingDir(tmpDir),
			}

			// 构建提示词模板
			promptTemplate, err := prompt.NewPrompt("agentic_fetch", string(agenticFetchPromptTmpl), promptOpts...)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("创建提示词失败: %s", err)
			}

			// 构建代理模型
			_, small, err := c.buildAgentModels(ctx, true)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("构建模型失败: %s", err)
			}

			// 构建系统提示词
			systemPrompt, err := promptTemplate.Build(ctx, small.Model.Provider(), small.Model.Model(), *c.cfg)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("构建系统提示词失败: %s", err)
			}

			// 获取小型模型提供商配置
			smallProviderCfg, ok := c.cfg.Providers.Get(small.ModelCfg.Provider)
			if !ok {
				return fantasy.ToolResponse{}, errors.New("小型模型提供商未配置")
			}

			// 创建网络工具
			webFetchTool := tools.NewWebFetchTool(tmpDir, client)
			webSearchTool := tools.NewWebSearchTool(client)
			fetchTools := []fantasy.AgentTool{
				webFetchTool,
				webSearchTool,
				tools.NewGlobTool(tmpDir),
				tools.NewGrepTool(tmpDir),
				tools.NewSourcegraphTool(client),
				tools.NewViewTool(c.lspManager, c.permissions, c.filetracker, tmpDir),
			}

			// 创建会话代理
			agent := NewSessionAgent(SessionAgentOptions{
				LargeModel:           small, // 对两者都使用小型模型（获取不需要大型模型）
				SmallModel:           small,
				SystemPromptPrefix:   smallProviderCfg.SystemPromptPrefix,
				SystemPrompt:         systemPrompt,
				DisableAutoSummarize: c.cfg.Options.DisableAutoSummarize,
				IsYolo:               c.permissions.SkipRequests(),
				Sessions:             c.sessions,
				Messages:             c.messages,
				Tools:                fetchTools,
			})

			// 创建代理工具会话
			agentToolSessionID := c.sessions.CreateAgentToolSessionID(validationResult.AgentMessageID, call.ID)
			session, err := c.sessions.CreateTaskSession(ctx, agentToolSessionID, validationResult.SessionID, "Fetch Analysis")
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("创建会话失败: %s", err)
			}

			// 自动批准会话权限
			c.permissions.AutoApproveSession(session.ID)

			// 使用小型模型进行网络内容分析（更快更便宜）
			maxTokens := small.CatwalkCfg.DefaultMaxTokens
			if small.ModelCfg.MaxTokens != 0 {
				maxTokens = small.ModelCfg.MaxTokens
			}

			// 运行代理
			result, err := agent.Run(ctx, SessionAgentCall{
				SessionID:        session.ID,
				Prompt:           fullPrompt,
				MaxOutputTokens:  maxTokens,
				ProviderOptions:  getProviderOptions(small, smallProviderCfg),
				Temperature:      small.ModelCfg.Temperature,
				TopP:             small.ModelCfg.TopP,
				TopK:             small.ModelCfg.TopK,
				FrequencyPenalty: small.ModelCfg.FrequencyPenalty,
				PresencePenalty:  small.ModelCfg.PresencePenalty,
			})
			if err != nil {
				return fantasy.NewTextErrorResponse("生成响应失败"), nil
			}

			// 更新父会话的成本
			updatedSession, err := c.sessions.Get(ctx, session.ID)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("获取会话失败: %s", err)
			}
			parentSession, err := c.sessions.Get(ctx, validationResult.SessionID)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("获取父会话失败: %s", err)
			}

			parentSession.Cost += updatedSession.Cost

			// 保存更新后的父会话
			_, err = c.sessions.Save(ctx, parentSession)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("保存父会话失败: %s", err)
			}

			// 返回代理的响应
			return fantasy.NewTextResponse(result.Response.Content.Text()), nil
		}), nil
}
