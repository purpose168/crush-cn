package tools

import (
	"context"
)

type (
	sessionIDContextKey string
	messageIDContextKey string
	supportsImagesKey   string
	modelNameKey        string
)

const (
	// SessionIDContextKey 是上下文中会话 ID 的键。
	SessionIDContextKey sessionIDContextKey = "session_id"
	// MessageIDContextKey 是上下文中消息 ID 的键。
	MessageIDContextKey messageIDContextKey = "message_id"
	// SupportsImagesContextKey 是上下文中模型图像支持能力的键。
	SupportsImagesContextKey supportsImagesKey = "supports_images"
	// ModelNameContextKey 是上下文中模型名称的键。
	ModelNameContextKey modelNameKey = "model_name"
)

// GetSessionFromContext 从上下文中检索会话 ID。
func GetSessionFromContext(ctx context.Context) string {
	sessionID := ctx.Value(SessionIDContextKey)
	if sessionID == nil {
		return ""
	}
	s, ok := sessionID.(string)
	if !ok {
		return ""
	}
	return s
}

// GetMessageFromContext 从上下文中检索消息 ID。
func GetMessageFromContext(ctx context.Context) string {
	messageID := ctx.Value(MessageIDContextKey)
	if messageID == nil {
		return ""
	}
	s, ok := messageID.(string)
	if !ok {
		return ""
	}
	return s
}

// GetSupportsImagesFromContext 从上下文中检索模型是否支持图像。
func GetSupportsImagesFromContext(ctx context.Context) bool {
	supportsImages := ctx.Value(SupportsImagesContextKey)
	if supportsImages == nil {
		return false
	}
	if supports, ok := supportsImages.(bool); ok {
		return supports
	}
	return false
}

// GetModelNameFromContext 从上下文中检索模型名称。
func GetModelNameFromContext(ctx context.Context) string {
	modelName := ctx.Value(ModelNameContextKey)
	if modelName == nil {
		return ""
	}
	s, ok := modelName.(string)
	if !ok {
		return ""
	}
	return s
}
