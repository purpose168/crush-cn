package event

// 这些测试验证 Error 函数能够正确处理各种场景。这些测试不会记录任何日志。

import (
	"testing"
)

func TestError(t *testing.T) {
	t.Run("当客户端为nil时提前返回", func(t *testing.T) {
		// 此测试验证当 PostHog 客户端未初始化时，Error 函数能够安全地提前返回，
		// 而不会尝试将任何事件加入队列。这在初始化期间或禁用指标时非常重要，
		// 因为我们不希望错误报告机制本身导致 panic。
		originalClient := client
		defer func() {
			client = originalClient
		}()

		client = nil
		Error("test error", "key", "value")
	})

	t.Run("处理nil客户端而不发生panic", func(t *testing.T) {
		// 此测试覆盖各种边界情况，其中错误值可能是 nil、字符串或 error 类型。
		originalClient := client
		defer func() {
			client = originalClient
		}()

		client = nil
		Error(nil)
		Error("some error")
		Error(newDefaultTestError("runtime error"), "key", "value")
	})

	t.Run("处理带属性的error", func(t *testing.T) {
		// 此测试验证 Error 函数能够处理提供错误上下文的额外键值属性。
		// 这些属性通常在从 panic 恢复时传递（例如：panic 名称、函数名称）。
		//
		// 即使有这些额外属性，函数也应该优雅地处理它们而不会发生 panic。
		originalClient := client
		defer func() {
			client = originalClient
		}()

		client = nil
		Error("test error",
			"type", "test",
			"severity", "high",
			"source", "unit-test",
		)
	})
}

// newDefaultTestError 创建一个模拟运行时 panic 错误的测试错误。
// 这有助于我们测试 Error 函数能够处理各种错误类型，
// 包括可能从 panic 恢复场景中传递的错误。
func newDefaultTestError(s string) error {
	return testError(s)
}

type testError string

func (e testError) Error() string {
	return string(e)
}
