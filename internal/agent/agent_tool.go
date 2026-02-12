package agent

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"charm.land/fantasy"

	"github.com/purpose168/crush-cn/internal/agent/prompt"
	"github.com/purpose168/crush-cn/internal/agent/tools"
	"github.com/purpose168/crush-cn/internal/config"
)

//go:embed templates/agent_tool.md
var agentToolDescription []byte

// AgentParams 代理工具参数
// Prompt: 代理需要执行的任务

type AgentParams struct {
	Prompt string `json:"prompt" description:"The task for the agent to perform"`
}

// AgentToolName 代理工具名称
const (
	AgentToolName = "agent"
)

// agentTool 创建代理工具
func (c *coordinator) agentTool(ctx context.Context) (fantasy.AgentTool, error) {
	agentCfg, ok := c.cfg.Agents[config.AgentTask]
	if !ok {
		return nil, errors.New("任务代理未配置")
	}
	prompt, err := taskPrompt(prompt.WithWorkingDir(c.cfg.WorkingDir()))
	if err != nil {
		return nil, err
	}

	agent, err := c.buildAgent(ctx, prompt, agentCfg, true)
	if err != nil {
		return nil, err
	}
	return fantasy.NewParallelAgentTool(
		AgentToolName,
		string(agentToolDescription),
		func(ctx context.Context, params AgentParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
			if params.Prompt == "" {
				return fantasy.NewTextErrorResponse("提示词是必需的"), nil
			}

			sessionID := tools.GetSessionFromContext(ctx)
			if sessionID == "" {
				return fantasy.ToolResponse{}, errors.New("上下文缺少会话ID")
			}

			agentMessageID := tools.GetMessageFromContext(ctx)
			if agentMessageID == "" {
				return fantasy.ToolResponse{}, errors.New("上下文缺少代理消息ID")
			}

			agentToolSessionID := c.sessions.CreateAgentToolSessionID(agentMessageID, call.ID)
			session, err := c.sessions.CreateTaskSession(ctx, agentToolSessionID, sessionID, "New Agent Session")
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("创建会话错误: %s", err)
			}
			model := agent.Model()
			maxTokens := model.CatwalkCfg.DefaultMaxTokens
			if model.ModelCfg.MaxTokens != 0 {
				maxTokens = model.ModelCfg.MaxTokens
			}

			providerCfg, ok := c.cfg.Providers.Get(model.ModelCfg.Provider)
			if !ok {
				return fantasy.ToolResponse{}, errors.New("模型提供商未配置")
			}
			result, err := agent.Run(ctx, SessionAgentCall{
				SessionID:        session.ID,
				Prompt:           params.Prompt,
				MaxOutputTokens:  maxTokens,
				ProviderOptions:  getProviderOptions(model, providerCfg),
				Temperature:      model.ModelCfg.Temperature,
				TopP:             model.ModelCfg.TopP,
				TopK:             model.ModelCfg.TopK,
				FrequencyPenalty: model.ModelCfg.FrequencyPenalty,
				PresencePenalty:  model.ModelCfg.PresencePenalty,
			})
			if err != nil {
				return fantasy.NewTextErrorResponse("生成响应错误"), nil
			}
			updatedSession, err := c.sessions.Get(ctx, session.ID)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("获取会话错误: %s", err)
			}
			parentSession, err := c.sessions.Get(ctx, sessionID)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("获取父会话错误: %s", err)
			}

			parentSession.Cost += updatedSession.Cost

			_, err = c.sessions.Save(ctx, parentSession)
			if err != nil {
				return fantasy.ToolResponse{}, fmt.Errorf("保存父会话错误: %s", err)
			}
			return fantasy.NewTextResponse(result.Response.Content.Text()), nil
		}), nil
}
