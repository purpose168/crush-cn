package app

import (
	"context"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/purpose168/crush-cn/internal/pubsub"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

// 测试 setupSubscriber 的正常流程
func TestSetupSubscriber_NormalFlow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newSubscriberFixture(t, 10)

		time.Sleep(10 * time.Millisecond)
		synctest.Wait()

		f.broker.Publish(pubsub.CreatedEvent, "event1")
		f.broker.Publish(pubsub.CreatedEvent, "event2")

		for range 2 {
			select {
			case <-f.outputCh:
			case <-time.After(5 * time.Second):
				t.Fatal("等待消息超时")
			}
		}

		f.cancel()
		f.wg.Wait()
	})
}

// 测试 setupSubscriber 处理慢消费者的情况
func TestSetupSubscriber_SlowConsumer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newSubscriberFixture(t, 0)

		const numEvents = 5

		var pubWg sync.WaitGroup
		pubWg.Go(func() {
			for range numEvents {
				f.broker.Publish(pubsub.CreatedEvent, "event")
				time.Sleep(10 * time.Millisecond)
				synctest.Wait()
			}
		})

		time.Sleep(time.Duration(numEvents) * (subscriberSendTimeout + 20*time.Millisecond))
		synctest.Wait()

		received := 0
		for {
			select {
			case <-f.outputCh:
				received++
			default:
				pubWg.Wait()
				f.cancel()
				f.wg.Wait()
				require.Less(t, received, numEvents, "慢消费者应该丢弃一些消息")
				return
			}
		}
	})
}

// 测试 setupSubscriber 的上下文取消情况
func TestSetupSubscriber_ContextCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newSubscriberFixture(t, 10)

		f.broker.Publish(pubsub.CreatedEvent, "event1")
		time.Sleep(100 * time.Millisecond)
		synctest.Wait()

		f.cancel()
		f.wg.Wait()
	})
}

// 测试 setupSubscriber 在消息丢弃后是否能正确清理资源
func TestSetupSubscriber_DrainAfterDrop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newSubscriberFixture(t, 0)

		time.Sleep(10 * time.Millisecond)
		synctest.Wait()

		// 第一个事件：没人读取 outputCh，所以定时器触发（消息被丢弃）
		f.broker.Publish(pubsub.CreatedEvent, "event1")
		time.Sleep(subscriberSendTimeout + 25*time.Millisecond)
		synctest.Wait()

		// 第二个事件：触发 Stop()==false 路径；如果没有修复，这里会发生死锁
		f.broker.Publish(pubsub.CreatedEvent, "event2")

		// 如果定时器清理发生死锁，wg.Wait 永远不会返回
		done := make(chan struct{})
		go func() {
			f.cancel()
			f.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("setupSubscriber 协程挂起 — 可能是定时器清理死锁")
		}
	})
}

// 测试 setupSubscriber 是否存在定时器泄漏
func TestSetupSubscriber_NoTimerLeak(t *testing.T) {
	defer goleak.VerifyNone(t)
	synctest.Test(t, func(t *testing.T) {
		f := newSubscriberFixture(t, 100)

		for range 100 {
			f.broker.Publish(pubsub.CreatedEvent, "event")
			time.Sleep(5 * time.Millisecond)
			synctest.Wait()
		}

		f.cancel()
		f.wg.Wait()
	})
}

// subscriberFixture 是测试订阅者的辅助结构体
type subscriberFixture struct {
	broker   *pubsub.Broker[string]  // 消息代理
	wg       sync.WaitGroup          // 等待组，用于同步协程
	outputCh chan tea.Msg            // 输出通道，用于接收消息
	cancel   context.CancelFunc      // 取消函数，用于取消上下文
}

// newSubscriberFixture 创建一个新的订阅者测试夹具
// t: 测试对象
// bufSize: 输出通道的缓冲区大小
func newSubscriberFixture(t *testing.T, bufSize int) *subscriberFixture {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	f := &subscriberFixture{
		broker:   pubsub.NewBroker[string](),
		outputCh: make(chan tea.Msg, bufSize),
		cancel:   cancel,
	}
	t.Cleanup(f.broker.Shutdown)

	setupSubscriber(ctx, &f.wg, "test", func(ctx context.Context) <-chan pubsub.Event[string] {
		return f.broker.Subscribe(ctx)
	}, f.outputCh)

	return f
}
