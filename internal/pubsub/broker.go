package pubsub

import (
	"context"
	"sync"
)

// bufferSize 订阅者通道的默认缓冲区大小
const bufferSize = 64

// Broker 事件代理，实现发布-订阅模式
// T 是事件载荷的类型
type Broker[T any] struct {
	subs      map[chan Event[T]]struct{}  // 订阅者通道映射
	mu        sync.RWMutex                // 读写互斥锁，保护并发访问
	done      chan struct{}               // 关闭信号通道
	subCount  int                         // 当前订阅者数量
	maxEvents int                         // 最大事件数量限制
}

// NewBroker 创建新的事件代理
// 使用默认配置创建代理实例
func NewBroker[T any]() *Broker[T] {
	return NewBrokerWithOptions[T](bufferSize, 1000)
}

// NewBrokerWithOptions 使用自定义配置创建事件代理
// 参数:
//   - channelBufferSize: 订阅者通道的缓冲区大小
//   - maxEvents: 最大事件数量限制
func NewBrokerWithOptions[T any](channelBufferSize, maxEvents int) *Broker[T] {
	return &Broker[T]{
		subs:      make(map[chan Event[T]]struct{}),
		done:      make(chan struct{}),
		maxEvents: maxEvents,
	}
}

// Shutdown 关闭事件代理
// 关闭所有订阅者通道并清理资源
func (b *Broker[T]) Shutdown() {
	select {
	case <-b.done:  // 已经关闭
		return
	default:
		close(b.done)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// 关闭所有订阅者通道
	for ch := range b.subs {
		delete(b.subs, ch)
		close(ch)
	}

	b.subCount = 0
}

// Subscribe 订阅事件
// 返回一个事件通道，订阅者通过此通道接收事件
// 当上下文取消时，自动取消订阅
func (b *Broker[T]) Subscribe(ctx context.Context) <-chan Event[T] {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 检查代理是否已关闭
	select {
	case <-b.done:
		ch := make(chan Event[T])
		close(ch)
		return ch
	default:
	}

	// 创建新的订阅者通道
	sub := make(chan Event[T], bufferSize)
	b.subs[sub] = struct{}{}
	b.subCount++

	// 启动goroutine监听上下文取消
	go func() {
		<-ctx.Done()

		b.mu.Lock()
		defer b.mu.Unlock()

		// 检查代理是否已关闭
		select {
		case <-b.done:
			return
		default:
		}

		// 移除订阅者
		delete(b.subs, sub)
		close(sub)
		b.subCount--
	}()

	return sub
}

// GetSubscriberCount 获取当前订阅者数量
func (b *Broker[T]) GetSubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.subCount
}

// Publish 发布事件
// 将事件发送给所有订阅者
// 如果订阅者通道已满，则跳过该订阅者（非阻塞）
func (b *Broker[T]) Publish(t EventType, payload T) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// 检查代理是否已关闭
	select {
	case <-b.done:
		return
	default:
	}

	// 构建事件对象
	event := Event[T]{Type: t, Payload: payload}

	// 向所有订阅者发送事件
	for sub := range b.subs {
		select {
		case sub <- event:
			// 事件发送成功
		default:
			// 通道已满，订阅者处理较慢 - 跳过此事件
			// 这可以防止阻塞发布者
		}
	}
}
