// Package message 提供消息管理服务，包括消息的创建、更新、查询和删除功能
package message

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/purpose168/crush-cn/internal/db"
	"github.com/purpose168/crush-cn/internal/pubsub"
)

// CreateMessageParams 创建消息的参数结构体
type CreateMessageParams struct {
	Role             MessageRole    // 消息角色（用户/助手）
	Parts            []ContentPart  // 消息内容部分
	Model            string         // 使用的模型名称
	Provider         string         // 提供商名称
	IsSummaryMessage bool           // 是否为摘要消息
}

// Service 消息服务接口，定义了消息管理的核心操作
type Service interface {
	pubsub.Subscriber[Message]
	// Create 创建新消息
	Create(ctx context.Context, sessionID string, params CreateMessageParams) (Message, error)
	// Update 更新消息内容
	Update(ctx context.Context, message Message) error
	// Get 根据ID获取消息
	Get(ctx context.Context, id string) (Message, error)
	// List 列出指定会话的所有消息
	List(ctx context.Context, sessionID string) ([]Message, error)
	// ListUserMessages 列出指定会话的用户消息
	ListUserMessages(ctx context.Context, sessionID string) ([]Message, error)
	// ListAllUserMessages 列出所有用户消息
	ListAllUserMessages(ctx context.Context) ([]Message, error)
	// Delete 删除指定消息
	Delete(ctx context.Context, id string) error
	// DeleteSessionMessages 删除指定会话的所有消息
	DeleteSessionMessages(ctx context.Context, sessionID string) error
}

// service 消息服务的具体实现
type service struct {
	*pubsub.Broker[Message]
	q db.Querier
}

// NewService 创建新的消息服务实例
func NewService(q db.Querier) Service {
	return &service{
		Broker: pubsub.NewBroker[Message](),
		q:      q,
	}
}

// Delete 删除指定消息
func (s *service) Delete(ctx context.Context, id string) error {
	message, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	err = s.q.DeleteMessage(ctx, message.ID)
	if err != nil {
		return err
	}
	// 在发布前克隆消息，以避免与 Parts 切片的并发修改产生竞态条件
	s.Publish(pubsub.DeletedEvent, message.Clone())
	return nil
}

// Create 创建新消息并保存到数据库
func (s *service) Create(ctx context.Context, sessionID string, params CreateMessageParams) (Message, error) {
	// 如果不是助手消息，添加完成标记
	if params.Role != Assistant {
		params.Parts = append(params.Parts, Finish{
			Reason: "stop",
		})
	}
	// 序列化消息内容部分
	partsJSON, err := marshalParts(params.Parts)
	if err != nil {
		return Message{}, err
	}
	// 转换布尔值为整数标志
	isSummary := int64(0)
	if params.IsSummaryMessage {
		isSummary = 1
	}
	// 在数据库中创建消息记录
	dbMessage, err := s.q.CreateMessage(ctx, db.CreateMessageParams{
		ID:               uuid.New().String(),
		SessionID:        sessionID,
		Role:             string(params.Role),
		Parts:            string(partsJSON),
		Model:            sql.NullString{String: string(params.Model), Valid: true},
		Provider:         sql.NullString{String: params.Provider, Valid: params.Provider != ""},
		IsSummaryMessage: isSummary,
	})
	if err != nil {
		return Message{}, err
	}
	// 将数据库记录转换为消息对象
	message, err := s.fromDBItem(dbMessage)
	if err != nil {
		return Message{}, err
	}
	// 在发布前克隆消息，以避免与 Parts 切片的并发修改产生竞态条件
	s.Publish(pubsub.CreatedEvent, message.Clone())
	return message, nil
}

// DeleteSessionMessages 删除指定会话的所有消息
func (s *service) DeleteSessionMessages(ctx context.Context, sessionID string) error {
	messages, err := s.List(ctx, sessionID)
	if err != nil {
		return err
	}
	// 遍历并删除该会话的所有消息
	for _, message := range messages {
		if message.SessionID == sessionID {
			err = s.Delete(ctx, message.ID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Update 更新消息内容
func (s *service) Update(ctx context.Context, message Message) error {
	// 序列化消息内容部分
	parts, err := marshalParts(message.Parts)
	if err != nil {
		return err
	}
	// 设置完成时间
	finishedAt := sql.NullInt64{}
	if f := message.FinishPart(); f != nil {
		finishedAt.Int64 = f.Time
		finishedAt.Valid = true
	}
	// 更新数据库中的消息记录
	err = s.q.UpdateMessage(ctx, db.UpdateMessageParams{
		ID:         message.ID,
		Parts:      string(parts),
		FinishedAt: finishedAt,
	})
	if err != nil {
		return err
	}
	message.UpdatedAt = time.Now().Unix()
	// 在发布前克隆消息，以避免与 Parts 切片的并发修改产生竞态条件
	s.Publish(pubsub.UpdatedEvent, message.Clone())
	return nil
}

// Get 根据ID获取消息
func (s *service) Get(ctx context.Context, id string) (Message, error) {
	dbMessage, err := s.q.GetMessage(ctx, id)
	if err != nil {
		return Message{}, err
	}
	return s.fromDBItem(dbMessage)
}

// List 列出指定会话的所有消息
func (s *service) List(ctx context.Context, sessionID string) ([]Message, error) {
	dbMessages, err := s.q.ListMessagesBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	// 将数据库记录列表转换为消息对象列表
	messages := make([]Message, len(dbMessages))
	for i, dbMessage := range dbMessages {
		messages[i], err = s.fromDBItem(dbMessage)
		if err != nil {
			return nil, err
		}
	}
	return messages, nil
}

// ListUserMessages 列出指定会话的用户消息
func (s *service) ListUserMessages(ctx context.Context, sessionID string) ([]Message, error) {
	dbMessages, err := s.q.ListUserMessagesBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	// 将数据库记录列表转换为消息对象列表
	messages := make([]Message, len(dbMessages))
	for i, dbMessage := range dbMessages {
		messages[i], err = s.fromDBItem(dbMessage)
		if err != nil {
			return nil, err
		}
	}
	return messages, nil
}

// ListAllUserMessages 列出所有用户消息
func (s *service) ListAllUserMessages(ctx context.Context) ([]Message, error) {
	dbMessages, err := s.q.ListAllUserMessages(ctx)
	if err != nil {
		return nil, err
	}
	// 将数据库记录列表转换为消息对象列表
	messages := make([]Message, len(dbMessages))
	for i, dbMessage := range dbMessages {
		messages[i], err = s.fromDBItem(dbMessage)
		if err != nil {
			return nil, err
		}
	}
	return messages, nil
}

// fromDBItem 将数据库记录转换为消息对象
func (s *service) fromDBItem(item db.Message) (Message, error) {
	// 反序列化消息内容部分
	parts, err := unmarshalParts([]byte(item.Parts))
	if err != nil {
		return Message{}, err
	}
	return Message{
		ID:               item.ID,
		SessionID:        item.SessionID,
		Role:             MessageRole(item.Role),
		Parts:            parts,
		Model:            item.Model.String,
		Provider:         item.Provider.String,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
		IsSummaryMessage: item.IsSummaryMessage != 0,
	}, nil
}

// partType 内容部分的类型标识
type partType string

// 定义各种内容部分的类型常量
const (
	reasoningType  partType = "reasoning"   // 推理内容类型
	textType       partType = "text"        // 文本内容类型
	imageURLType   partType = "image_url"   // 图片URL类型
	binaryType     partType = "binary"      // 二进制内容类型
	toolCallType   partType = "tool_call"   // 工具调用类型
	toolResultType partType = "tool_result" // 工具结果类型
	finishType     partType = "finish"      // 完成标记类型
)

// partWrapper 用于JSON序列化的内容部分包装器
type partWrapper struct {
	Type partType    `json:"type"` // 内容类型
	Data ContentPart `json:"data"` // 内容数据
}

// marshalParts 将内容部分列表序列化为JSON字节切片
func marshalParts(parts []ContentPart) ([]byte, error) {
	wrappedParts := make([]partWrapper, len(parts))

	for i, part := range parts {
		var typ partType

		// 根据内容部分的实际类型确定类型标识
		switch part.(type) {
		case ReasoningContent:
			typ = reasoningType
		case TextContent:
			typ = textType
		case ImageURLContent:
			typ = imageURLType
		case BinaryContent:
			typ = binaryType
		case ToolCall:
			typ = toolCallType
		case ToolResult:
			typ = toolResultType
		case Finish:
			typ = finishType
		default:
			return nil, fmt.Errorf("未知的内容部分类型: %T", part)
		}

		wrappedParts[i] = partWrapper{
			Type: typ,
			Data: part,
		}
	}
	return json.Marshal(wrappedParts)
}

// unmarshalParts 将JSON字节切片反序列化为内容部分列表
func unmarshalParts(data []byte) ([]ContentPart, error) {
	temp := []json.RawMessage{}

	if err := json.Unmarshal(data, &temp); err != nil {
		return nil, err
	}

	parts := make([]ContentPart, 0)

	for _, rawPart := range temp {
		var wrapper struct {
			Type partType        `json:"type"` // 内容类型
			Data json.RawMessage `json:"data"` // 原始JSON数据
		}

		if err := json.Unmarshal(rawPart, &wrapper); err != nil {
			return nil, err
		}

		// 根据类型标识反序列化为对应的内容部分
		switch wrapper.Type {
		case reasoningType:
			part := ReasoningContent{}
			if err := json.Unmarshal(wrapper.Data, &part); err != nil {
				return nil, err
			}
			parts = append(parts, part)
		case textType:
			part := TextContent{}
			if err := json.Unmarshal(wrapper.Data, &part); err != nil {
				return nil, err
			}
			parts = append(parts, part)
		case imageURLType:
			part := ImageURLContent{}
			if err := json.Unmarshal(wrapper.Data, &part); err != nil {
				return nil, err
			}
			parts = append(parts, part)
		case binaryType:
			part := BinaryContent{}
			if err := json.Unmarshal(wrapper.Data, &part); err != nil {
				return nil, err
			}
			parts = append(parts, part)
		case toolCallType:
			part := ToolCall{}
			if err := json.Unmarshal(wrapper.Data, &part); err != nil {
				return nil, err
			}
			parts = append(parts, part)
		case toolResultType:
			part := ToolResult{}
			if err := json.Unmarshal(wrapper.Data, &part); err != nil {
				return nil, err
			}
			parts = append(parts, part)
		case finishType:
			part := Finish{}
			if err := json.Unmarshal(wrapper.Data, &part); err != nil {
				return nil, err
			}
			parts = append(parts, part)
		default:
			return nil, fmt.Errorf("未知的内容部分类型: %s", wrapper.Type)
		}
	}

	return parts, nil
}
