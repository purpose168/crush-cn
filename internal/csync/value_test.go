package csync

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestValue_GetSet 测试Value类型的Get和Set方法的基本功能
func TestValue_GetSet(t *testing.T) {
	t.Parallel()

	// 创建一个初始值为42的Value实例
	v := NewValue(42)
	require.Equal(t, 42, v.Get())

	// 设置新值并验证
	v.Set(100)
	require.Equal(t, 100, v.Get())
}

// TestValue_ZeroValue 测试Value类型对零值的处理
func TestValue_ZeroValue(t *testing.T) {
	t.Parallel()

	// 创建一个空字符串零值的Value实例
	v := NewValue("")
	require.Equal(t, "", v.Get())

	// 设置新值并验证
	v.Set("hello")
	require.Equal(t, "hello", v.Get())
}

// TestValue_Struct 测试Value类型对结构体的处理
func TestValue_Struct(t *testing.T) {
	t.Parallel()

	// 定义测试用的配置结构体
	type config struct {
		Name  string
		Count int
	}

	// 创建一个包含结构体的Value实例
	v := NewValue(config{Name: "test", Count: 1})
	require.Equal(t, config{Name: "test", Count: 1}, v.Get())

	// 更新结构体值并验证
	v.Set(config{Name: "updated", Count: 2})
	require.Equal(t, config{Name: "updated", Count: 2}, v.Get())
}

// TestValue_PointerPanics 测试Value类型对指针类型的panic行为
func TestValue_PointerPanics(t *testing.T) {
	t.Parallel()

	// 验证创建指针类型的Value会触发panic
	require.Panics(t, func() {
		NewValue(&struct{}{})
	})
}

// TestValue_SlicePanics 测试Value类型对切片类型的panic行为
func TestValue_SlicePanics(t *testing.T) {
	t.Parallel()

	// 验证创建切片类型的Value会触发panic
	require.Panics(t, func() {
		NewValue([]string{"a", "b"})
	})
}

// TestValue_MapPanics 测试Value类型对map类型的panic行为
func TestValue_MapPanics(t *testing.T) {
	t.Parallel()

	// 验证创建map类型的Value会触发panic
	require.Panics(t, func() {
		NewValue(map[string]int{"a": 1})
	})
}

// TestValue_ConcurrentAccess 测试Value类型的并发访问安全性
func TestValue_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	// 创建一个初始值为0的Value实例
	v := NewValue(0)
	var wg sync.WaitGroup

	// 并发写入者：启动100个goroutine并发设置值
	for i := range 100 {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			v.Set(val)
		}(i)
	}

	// 并发读取者：启动100个goroutine并发读取值
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = v.Get()
		}()
	}

	// 等待所有goroutine完成
	wg.Wait()

	// 值应该是设置的值之一（0-99）
	got := v.Get()
	require.GreaterOrEqual(t, got, 0)
	require.Less(t, got, 100)
}
