package event

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"time"

	"github.com/posthog/posthog-go"
	"github.com/purpose168/crush-cn/internal/version"
)

const (
	endpoint = "https://data.charm.land"
	key      = "phc_4zt4VgDWLqbYnJYEwLRxFoaTL2noNrQij0C6E8k3I0V"

	nonInteractiveEventName = "NonInteractive"
)

var (
	client posthog.Client

	baseProps = posthog.NewProperties().
			Set("GOOS", runtime.GOOS).
			Set("GOARCH", runtime.GOARCH).
			Set("TERM", os.Getenv("TERM")).
			Set("SHELL", filepath.Base(os.Getenv("SHELL"))).
			Set("Version", version.Version).
			Set("GoVersion", runtime.Version()).
			Set(nonInteractiveEventName, false)
)

func SetNonInteractive(nonInteractive bool) {
	baseProps = baseProps.Set(nonInteractiveEventName, nonInteractive)
}

func Init() {
	c, err := posthog.NewWithConfig(key, posthog.Config{
		Endpoint:        endpoint,
		Logger:          logger{},
		ShutdownTimeout: 500 * time.Millisecond,
	})
	if err != nil {
		slog.Error("初始化 PostHog 客户端失败", "error", err)
	}
	client = c
	distinctId = getDistinctId()
}

func GetID() string { return distinctId }

func Alias(userID string) {
	if client == nil || distinctId == fallbackId || distinctId == "" || userID == "" {
		return
	}
	if err := client.Enqueue(posthog.Alias{
		DistinctId: distinctId,
		Alias:      userID,
	}); err != nil {
		slog.Error("将 PostHog 别名事件加入队列失败", "error", err)
		return
	}
	slog.Info("已在 PostHog 中设置别名", "machine_id", distinctId, "user_id", userID)
}

// send 使用给定的事件名称和属性向 PostHog 记录事件
func send(event string, props ...any) {
	if client == nil {
		return
	}
	err := client.Enqueue(posthog.Capture{
		DistinctId: distinctId,
		Event:      event,
		Properties: pairsToProps(props...).Merge(baseProps),
	})
	if err != nil {
		slog.Error("将 PostHog 事件加入队列失败", "event", event, "props", props, "error", err)
		return
	}
}

// Error 向 PostHog 记录错误事件，包含错误类型和消息
func Error(errToLog any, props ...any) {
	if client == nil {
		return
	}
	posthogErr := client.Enqueue(posthog.NewDefaultException(
		time.Now(),
		distinctId,
		reflect.TypeOf(errToLog).String(),
		fmt.Sprintf("%v", errToLog),
	))
	if posthogErr != nil {
		slog.Error("将 PostHog 错误加入队列失败", "err", errToLog, "props", props, "posthogErr", posthogErr)
		return
	}
}

func Flush() {
	if client == nil {
		return
	}
	if err := client.Close(); err != nil {
		slog.Error("刷新 PostHog 事件失败", "error", err)
	}
}

func pairsToProps(props ...any) posthog.Properties {
	p := posthog.NewProperties()

	if !isEven(len(props)) {
		slog.Error("事件属性必须以键值对的形式提供", "props", props)
		return p
	}

	for i := 0; i < len(props); i += 2 {
		key := props[i].(string)
		value := props[i+1]
		p = p.Set(key, value)
	}
	return p
}

func isEven(n int) bool {
	return n%2 == 0
}
