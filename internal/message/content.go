// Package message 提供消息内容处理相关的类型定义和方法
// 本包定义了消息角色、内容部分、工具调用等核心数据结构
package message

import (
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"charm.land/catwalk/pkg/catwalk"
	"charm.land/fantasy"
	"charm.land/fantasy/providers/anthropic"
	"charm.land/fantasy/providers/google"
	"charm.land/fantasy/providers/openai"
)

// MessageRole 定义消息角色的类型
type MessageRole string

const (
	// Assistant 表示助手角色
	Assistant MessageRole = "assistant"
	// User 表示用户角色
	User MessageRole = "user"
	// System 表示系统角色
	System MessageRole = "system"
	// Tool 表示工具角色
	Tool MessageRole = "tool"
)

// FinishReason 定义消息结束原因的类型
type FinishReason string

const (
	// FinishReasonEndTurn 表示回合正常结束
	FinishReasonEndTurn FinishReason = "end_turn"
	// FinishReasonMaxTokens 表示达到最大令牌数限制
	FinishReasonMaxTokens FinishReason = "max_tokens"
	// FinishReasonToolUse 表示工具调用
	FinishReasonToolUse FinishReason = "tool_use"
	// FinishReasonCanceled 表示已取消
	FinishReasonCanceled FinishReason = "canceled"
	// FinishReasonError 表示发生错误
	FinishReasonError FinishReason = "error"
	// FinishReasonPermissionDenied 表示权限被拒绝
	FinishReasonPermissionDenied FinishReason = "permission_denied"

	// FinishReasonUnknown 表示未知结束原因（不应发生）
	FinishReasonUnknown FinishReason = "unknown"
)

// ContentPart 定义内容部分的接口
// 所有内容类型都必须实现此接口以作为消息的一部分
type ContentPart interface {
	isPart()
}

// ReasoningContent 表示推理内容，包含思考过程和签名信息
type ReasoningContent struct {
	// Thinking 包含思考过程的文本内容
	Thinking string `json:"thinking"`
	// Signature 包含推理签名（用于 Anthropic）
	Signature string `json:"signature"`
	// ThoughtSignature 包含思考签名（用于 Google）
	ThoughtSignature string `json:"thought_signature"`
	// ToolID 包含工具标识符（用于 OpenRouter Google 模型）
	ToolID string `json:"tool_id"`
	// ResponsesData 包含 OpenAI 响应推理元数据
	ResponsesData *openai.ResponsesReasoningMetadata `json:"responses_data"`
	// StartedAt 表示推理开始时间戳
	StartedAt int64 `json:"started_at,omitempty"`
	// FinishedAt 表示推理结束时间戳
	FinishedAt int64 `json:"finished_at,omitempty"`
}

// String 返回推理内容的文本表示
func (tc ReasoningContent) String() string {
	return tc.Thinking
}

// isPart 实现 ContentPart 接口
func (ReasoningContent) isPart() {}

// TextContent 表示文本内容
type TextContent struct {
	// Text 包含文本内容
	Text string `json:"text"`
}

// String 返回文本内容
func (tc TextContent) String() string {
	return tc.Text
}

// isPart 实现 ContentPart 接口
func (TextContent) isPart() {}

// ImageURLContent 表示图片 URL 内容
type ImageURLContent struct {
	// URL 包含图片的 URL 地址
	URL string `json:"url"`
	// Detail 包含图片细节级别设置
	Detail string `json:"detail,omitempty"`
}

// String 返回图片 URL
func (iuc ImageURLContent) String() string {
	return iuc.URL
}

// isPart 实现 ContentPart 接口
func (ImageURLContent) isPart() {}

// BinaryContent 表示二进制内容
type BinaryContent struct {
	// Path 包含文件路径
	Path string
	// MIMEType 包含 MIME 类型
	MIMEType string
	// Data 包含二进制数据
	Data []byte
}

// String 返回二进制内容的字符串表示
// 根据不同的推理提供者返回不同格式的编码字符串
func (bc BinaryContent) String(p catwalk.InferenceProvider) string {
	base64Encoded := base64.StdEncoding.EncodeToString(bc.Data)
	// OpenAI 提供者需要 data URI 格式
	if p == catwalk.InferenceProviderOpenAI {
		return "data:" + bc.MIMEType + ";base64," + base64Encoded
	}
	return base64Encoded
}

// isPart 实现 ContentPart 接口
func (BinaryContent) isPart() {}

// ToolCall 表示工具调用
type ToolCall struct {
	// ID 包含工具调用的唯一标识符
	ID string `json:"id"`
	// Name 包含工具名称
	Name string `json:"name"`
	// Input 包含工具调用的输入参数（JSON 格式）
	Input string `json:"input"`
	// ProviderExecuted 表示是否由提供者执行
	ProviderExecuted bool `json:"provider_executed"`
	// Finished 表示工具调用是否已完成
	Finished bool `json:"finished"`
}

// isPart 实现 ContentPart 接口
func (ToolCall) isPart() {}

// ToolResult 表示工具执行结果
type ToolResult struct {
	// ToolCallID 包含对应工具调用的标识符
	ToolCallID string `json:"tool_call_id"`
	// Name 包含工具名称
	Name string `json:"name"`
	// Content 包含工具执行结果内容
	Content string `json:"content"`
	// Data 包含工具返回的数据
	Data string `json:"data"`
	// MIMEType 包含数据的 MIME 类型
	MIMEType string `json:"mime_type"`
	// Metadata 包含工具执行的元数据
	Metadata string `json:"metadata"`
	// IsError 表示结果是否为错误
	IsError bool `json:"is_error"`
}

// isPart 实现 ContentPart 接口
func (ToolResult) isPart() {}

// Finish 表示消息结束信息
type Finish struct {
	// Reason 包含结束原因
	Reason FinishReason `json:"reason"`
	// Time 包含结束时间戳
	Time int64 `json:"time"`
	// Message 包含结束消息
	Message string `json:"message,omitempty"`
	// Details 包含结束详情
	Details string `json:"details,omitempty"`
}

// isPart 实现 ContentPart 接口
func (Finish) isPart() {}

// Message 表示一条完整的消息
type Message struct {
	// ID 包含消息的唯一标识符
	ID string
	// Role 包含消息角色
	Role MessageRole
	// SessionID 包含会话标识符
	SessionID string
	// Parts 包含消息的所有内容部分
	Parts []ContentPart
	// Model 包含使用的模型名称
	Model string
	// Provider 包含提供者名称
	Provider string
	// CreatedAt 包含消息创建时间戳
	CreatedAt int64
	// UpdatedAt 包含消息更新时间戳
	UpdatedAt int64
	// IsSummaryMessage 表示是否为摘要消息
	IsSummaryMessage bool
}

// Content 返回消息中的文本内容
// 如果存在多个文本内容部分，返回第一个找到的
func (m *Message) Content() TextContent {
	for _, part := range m.Parts {
		if c, ok := part.(TextContent); ok {
			return c
		}
	}
	return TextContent{}
}

// ReasoningContent 返回消息中的推理内容
// 如果存在多个推理内容部分，返回第一个找到的
func (m *Message) ReasoningContent() ReasoningContent {
	for _, part := range m.Parts {
		if c, ok := part.(ReasoningContent); ok {
			return c
		}
	}
	return ReasoningContent{}
}

// ImageURLContent 返回消息中的所有图片 URL 内容
func (m *Message) ImageURLContent() []ImageURLContent {
	imageURLContents := make([]ImageURLContent, 0)
	for _, part := range m.Parts {
		if c, ok := part.(ImageURLContent); ok {
			imageURLContents = append(imageURLContents, c)
		}
	}
	return imageURLContents
}

// BinaryContent 返回消息中的所有二进制内容
func (m *Message) BinaryContent() []BinaryContent {
	binaryContents := make([]BinaryContent, 0)
	for _, part := range m.Parts {
		if c, ok := part.(BinaryContent); ok {
			binaryContents = append(binaryContents, c)
		}
	}
	return binaryContents
}

// ToolCalls 返回消息中的所有工具调用
func (m *Message) ToolCalls() []ToolCall {
	toolCalls := make([]ToolCall, 0)
	for _, part := range m.Parts {
		if c, ok := part.(ToolCall); ok {
			toolCalls = append(toolCalls, c)
		}
	}
	return toolCalls
}

// ToolResults 返回消息中的所有工具结果
func (m *Message) ToolResults() []ToolResult {
	toolResults := make([]ToolResult, 0)
	for _, part := range m.Parts {
		if c, ok := part.(ToolResult); ok {
			toolResults = append(toolResults, c)
		}
	}
	return toolResults
}

// IsFinished 检查消息是否已结束
func (m *Message) IsFinished() bool {
	for _, part := range m.Parts {
		if _, ok := part.(Finish); ok {
			return true
		}
	}
	return false
}

// FinishPart 返回消息的结束部分
// 如果不存在结束部分，返回 nil
func (m *Message) FinishPart() *Finish {
	for _, part := range m.Parts {
		if c, ok := part.(Finish); ok {
			return &c
		}
	}
	return nil
}

// FinishReason 返回消息的结束原因
// 如果消息未结束，返回空字符串
func (m *Message) FinishReason() FinishReason {
	for _, part := range m.Parts {
		if c, ok := part.(Finish); ok {
			return c.Reason
		}
	}
	return ""
}

// IsThinking 检查消息是否正在思考中
// 当存在推理内容但没有文本内容且未结束时返回 true
func (m *Message) IsThinking() bool {
	if m.ReasoningContent().Thinking != "" && m.Content().Text == "" && !m.IsFinished() {
		return true
	}
	return false
}

// AppendContent 向消息追加文本内容增量
// 如果已存在文本内容部分，则追加到该部分；否则创建新的文本内容部分
func (m *Message) AppendContent(delta string) {
	found := false
	for i, part := range m.Parts {
		if c, ok := part.(TextContent); ok {
			m.Parts[i] = TextContent{Text: c.Text + delta}
			found = true
		}
	}
	if !found {
		m.Parts = append(m.Parts, TextContent{Text: delta})
	}
}

// AppendReasoningContent 向消息追加推理内容增量
// 如果已存在推理内容部分，则追加到该部分；否则创建新的推理内容部分
func (m *Message) AppendReasoningContent(delta string) {
	found := false
	for i, part := range m.Parts {
		if c, ok := part.(ReasoningContent); ok {
			m.Parts[i] = ReasoningContent{
				Thinking:   c.Thinking + delta,
				Signature:  c.Signature,
				StartedAt:  c.StartedAt,
				FinishedAt: c.FinishedAt,
			}
			found = true
		}
	}
	if !found {
		m.Parts = append(m.Parts, ReasoningContent{
			Thinking:  delta,
			StartedAt: time.Now().Unix(),
		})
	}
}

// AppendThoughtSignature 向推理内容追加思考签名
// 用于 Google 提供者的推理验证
func (m *Message) AppendThoughtSignature(signature string, toolCallID string) {
	for i, part := range m.Parts {
		if c, ok := part.(ReasoningContent); ok {
			m.Parts[i] = ReasoningContent{
				Thinking:         c.Thinking,
				ThoughtSignature: c.ThoughtSignature + signature,
				ToolID:           toolCallID,
				Signature:        c.Signature,
				StartedAt:        c.StartedAt,
				FinishedAt:       c.FinishedAt,
			}
			return
		}
	}
	m.Parts = append(m.Parts, ReasoningContent{ThoughtSignature: signature})
}

// AppendReasoningSignature 向推理内容追加推理签名
// 用于 Anthropic 提供者的推理验证
func (m *Message) AppendReasoningSignature(signature string) {
	for i, part := range m.Parts {
		if c, ok := part.(ReasoningContent); ok {
			m.Parts[i] = ReasoningContent{
				Thinking:   c.Thinking,
				Signature:  c.Signature + signature,
				StartedAt:  c.StartedAt,
				FinishedAt: c.FinishedAt,
			}
			return
		}
	}
	m.Parts = append(m.Parts, ReasoningContent{Signature: signature})
}

// SetReasoningResponsesData 设置 OpenAI 响应推理元数据
func (m *Message) SetReasoningResponsesData(data *openai.ResponsesReasoningMetadata) {
	for i, part := range m.Parts {
		if c, ok := part.(ReasoningContent); ok {
			m.Parts[i] = ReasoningContent{
				Thinking:      c.Thinking,
				ResponsesData: data,
				StartedAt:     c.StartedAt,
				FinishedAt:    c.FinishedAt,
			}
			return
		}
	}
}

// FinishThinking 标记推理内容结束
// 设置推理结束时间戳
func (m *Message) FinishThinking() {
	for i, part := range m.Parts {
		if c, ok := part.(ReasoningContent); ok {
			if c.FinishedAt == 0 {
				m.Parts[i] = ReasoningContent{
					Thinking:   c.Thinking,
					Signature:  c.Signature,
					StartedAt:  c.StartedAt,
					FinishedAt: time.Now().Unix(),
				}
			}
			return
		}
	}
}

// ThinkingDuration 返回推理持续时间
// 如果推理未开始，返回 0；如果推理未结束，使用当前时间计算
func (m *Message) ThinkingDuration() time.Duration {
	reasoning := m.ReasoningContent()
	if reasoning.StartedAt == 0 {
		return 0
	}

	endTime := reasoning.FinishedAt
	if endTime == 0 {
		endTime = time.Now().Unix()
	}

	return time.Duration(endTime-reasoning.StartedAt) * time.Second
}

// FinishToolCall 标记指定的工具调用为已完成
func (m *Message) FinishToolCall(toolCallID string) {
	for i, part := range m.Parts {
		if c, ok := part.(ToolCall); ok {
			if c.ID == toolCallID {
				m.Parts[i] = ToolCall{
					ID:       c.ID,
					Name:     c.Name,
					Input:    c.Input,
					Finished: true,
				}
				return
			}
		}
	}
}

// AppendToolCallInput 向指定的工具调用追加输入参数增量
func (m *Message) AppendToolCallInput(toolCallID string, inputDelta string) {
	for i, part := range m.Parts {
		if c, ok := part.(ToolCall); ok {
			if c.ID == toolCallID {
				m.Parts[i] = ToolCall{
					ID:       c.ID,
					Name:     c.Name,
					Input:    c.Input + inputDelta,
					Finished: c.Finished,
				}
				return
			}
		}
	}
}

// AddToolCall 添加工具调用到消息
// 如果已存在相同 ID 的工具调用，则更新它
func (m *Message) AddToolCall(tc ToolCall) {
	for i, part := range m.Parts {
		if c, ok := part.(ToolCall); ok {
			if c.ID == tc.ID {
				m.Parts[i] = tc
				return
			}
		}
	}
	m.Parts = append(m.Parts, tc)
}

// SetToolCalls 设置消息的工具调用列表
// 移除所有现有的工具调用部分，然后添加新的工具调用
func (m *Message) SetToolCalls(tc []ToolCall) {
	// 移除所有现有的工具调用部分（可能有多个）
	parts := make([]ContentPart, 0)
	for _, part := range m.Parts {
		if _, ok := part.(ToolCall); ok {
			continue
		}
		parts = append(parts, part)
	}
	m.Parts = parts
	for _, toolCall := range tc {
		m.Parts = append(m.Parts, toolCall)
	}
}

// AddToolResult 添加工具执行结果到消息
func (m *Message) AddToolResult(tr ToolResult) {
	m.Parts = append(m.Parts, tr)
}

// SetToolResults 设置消息的工具结果列表
func (m *Message) SetToolResults(tr []ToolResult) {
	for _, toolResult := range tr {
		m.Parts = append(m.Parts, toolResult)
	}
}

// Clone 返回消息的深拷贝，包含独立的 Parts 切片
// 这可以防止在并发修改消息时发生竞态条件
func (m *Message) Clone() Message {
	clone := *m
	clone.Parts = make([]ContentPart, len(m.Parts))
	copy(clone.Parts, m.Parts)
	return clone
}

// AddFinish 添加消息结束信息
// 移除任何现有的结束部分，然后添加新的结束信息
func (m *Message) AddFinish(reason FinishReason, message, details string) {
	// 移除任何现有的结束部分
	for i, part := range m.Parts {
		if _, ok := part.(Finish); ok {
			m.Parts = slices.Delete(m.Parts, i, i+1)
			break
		}
	}
	m.Parts = append(m.Parts, Finish{Reason: reason, Time: time.Now().Unix(), Message: message, Details: details})
}

// AddImageURL 添加图片 URL 到消息
func (m *Message) AddImageURL(url, detail string) {
	m.Parts = append(m.Parts, ImageURLContent{URL: url, Detail: detail})
}

// AddBinary 添加二进制内容到消息
func (m *Message) AddBinary(mimeType string, data []byte) {
	m.Parts = append(m.Parts, BinaryContent{MIMEType: mimeType, Data: data})
}

// PromptWithTextAttachments 将提示词与文本附件组合成完整的提示字符串
// 文本附件会被包装在特定的 XML 标签中，并附带系统说明
func PromptWithTextAttachments(prompt string, attachments []Attachment) string {
	var sb strings.Builder
	sb.WriteString(prompt)
	addedAttachments := false
	for _, content := range attachments {
		if !content.IsText() {
			continue
		}
		if !addedAttachments {
			sb.WriteString("\n<system_info>以下文件已由用户附加，请在响应中考虑这些文件</system_info>\n")
			addedAttachments = true
		}
		if content.FilePath != "" {
			fmt.Fprintf(&sb, "<file path='%s'>\n", content.FilePath)
		} else {
			sb.WriteString("<file>\n")
		}
		sb.WriteString("\n")
		sb.Write(content.Content)
		sb.WriteString("\n</file>\n")
	}
	return sb.String()
}

// ToAIMessage 将消息转换为 fantasy.Message 格式
// 根据消息角色（用户、助手、工具）转换为对应的 fantasy 消息格式
func (m *Message) ToAIMessage() []fantasy.Message {
	var messages []fantasy.Message
	switch m.Role {
	case User:
		var parts []fantasy.MessagePart
		text := strings.TrimSpace(m.Content().Text)
		var textAttachments []Attachment
		// 收集文本类型的二进制内容作为附件
		for _, content := range m.BinaryContent() {
			if !strings.HasPrefix(content.MIMEType, "text/") {
				continue
			}
			textAttachments = append(textAttachments, Attachment{
				FilePath: content.Path,
				MimeType: content.MIMEType,
				Content:  content.Data,
			})
		}
		text = PromptWithTextAttachments(text, textAttachments)
		if text != "" {
			parts = append(parts, fantasy.TextPart{Text: text})
		}
		// 处理非文本类型的二进制内容
		for _, content := range m.BinaryContent() {
			// 跳过文本附件（已处理）
			if strings.HasPrefix(content.MIMEType, "text/") {
				continue
			}
			parts = append(parts, fantasy.FilePart{
				Filename:  content.Path,
				Data:      content.Data,
				MediaType: content.MIMEType,
			})
		}
		messages = append(messages, fantasy.Message{
			Role:    fantasy.MessageRoleUser,
			Content: parts,
		})
	case Assistant:
		var parts []fantasy.MessagePart
		text := strings.TrimSpace(m.Content().Text)
		if text != "" {
			parts = append(parts, fantasy.TextPart{Text: text})
		}
		// 处理推理内容
		reasoning := m.ReasoningContent()
		if reasoning.Thinking != "" {
			reasoningPart := fantasy.ReasoningPart{Text: reasoning.Thinking, ProviderOptions: fantasy.ProviderOptions{}}
			// 添加 Anthropic 提供者的推理签名选项
			if reasoning.Signature != "" {
				reasoningPart.ProviderOptions[anthropic.Name] = &anthropic.ReasoningOptionMetadata{
					Signature: reasoning.Signature,
				}
			}
			// 添加 OpenAI 提供者的响应数据
			if reasoning.ResponsesData != nil {
				reasoningPart.ProviderOptions[openai.Name] = reasoning.ResponsesData
			}
			// 添加 Google 提供者的推理元数据
			if reasoning.ThoughtSignature != "" {
				reasoningPart.ProviderOptions[google.Name] = &google.ReasoningMetadata{
					Signature: reasoning.ThoughtSignature,
					ToolID:    reasoning.ToolID,
				}
			}
			parts = append(parts, reasoningPart)
		}
		// 添加工具调用
		for _, call := range m.ToolCalls() {
			parts = append(parts, fantasy.ToolCallPart{
				ToolCallID:       call.ID,
				ToolName:         call.Name,
				Input:            call.Input,
				ProviderExecuted: call.ProviderExecuted,
			})
		}
		messages = append(messages, fantasy.Message{
			Role:    fantasy.MessageRoleAssistant,
			Content: parts,
		})
	case Tool:
		var parts []fantasy.MessagePart
		// 处理工具执行结果
		for _, result := range m.ToolResults() {
			var content fantasy.ToolResultOutputContent
			if result.IsError {
				// 错误结果
				content = fantasy.ToolResultOutputContentError{
					Error: errors.New(result.Content),
				}
			} else if result.Data != "" {
				// 媒体内容结果
				content = fantasy.ToolResultOutputContentMedia{
					Data:      result.Data,
					MediaType: result.MIMEType,
				}
			} else {
				// 文本内容结果
				content = fantasy.ToolResultOutputContentText{
					Text: result.Content,
				}
			}
			parts = append(parts, fantasy.ToolResultPart{
				ToolCallID: result.ToolCallID,
				Output:     content,
			})
		}
		messages = append(messages, fantasy.Message{
			Role:    fantasy.MessageRoleTool,
			Content: parts,
		})
	}
	return messages
}
