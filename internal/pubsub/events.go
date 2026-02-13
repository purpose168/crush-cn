package pubsub

import "context"

// 事件类型常量定义
const (
	CreatedEvent EventType = "created"  // 创建事件
	UpdatedEvent EventType = "updated"  // 更新事件
	DeletedEvent EventType = "deleted"  // 删除事件
)

// Subscriber 订阅者接口
// 定义了订阅事件的标准接口，订阅者通过此接口接收事件通知
type Subscriber[T any] interface {
	Subscribe(context.Context) <-chan Event[T]
}

type (
	// EventType 事件类型标识符
	// 用于标识不同类型的事件
	EventType string

	// Event 表示资源生命周期中的一个事件
	// T 是事件载荷的类型
	Event[T any] struct {
		Type    EventType  // 事件类型
		Payload T          // 事件载荷数据
	}

	// Publisher 发布者接口
	// 定义了发布事件的标准接口
	Publisher[T any] interface {
		Publish(EventType, T)
	}
)
