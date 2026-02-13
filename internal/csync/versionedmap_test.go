package csync

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionedMap_Set(t *testing.T) {
	t.Parallel()

	vm := NewVersionedMap[string, int]()
	require.Equal(t, uint64(0), vm.Version())

	vm.Set("key1", 42)
	require.Equal(t, uint64(1), vm.Version())

	value, ok := vm.Get("key1")
	require.True(t, ok)
	require.Equal(t, 42, value)
}

func TestVersionedMap_Del(t *testing.T) {
	t.Parallel()

	vm := NewVersionedMap[string, int]()
	vm.Set("key1", 42)
	initialVersion := vm.Version()

	vm.Del("key1")
	require.Equal(t, initialVersion+1, vm.Version())

	_, ok := vm.Get("key1")
	require.False(t, ok)
}

func TestVersionedMap_VersionIncrement(t *testing.T) {
	t.Parallel()

	vm := NewVersionedMap[string, int]()
	initialVersion := vm.Version()

	// 设置值应该增加版本号
	vm.Set("key1", 42)
	require.Equal(t, initialVersion+1, vm.Version())

	// 删除值应该增加版本号
	vm.Del("key1")
	require.Equal(t, initialVersion+2, vm.Version())

	// 删除不存在的键仍然会增加版本号
	vm.Del("nonexistent")
	require.Equal(t, initialVersion+3, vm.Version())
}

func TestVersionedMap_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	vm := NewVersionedMap[int, int]()
	const numGoroutines = 100
	const numOperations = 100

	// 初始版本号
	initialVersion := vm.Version()

	// 执行并发的 Set 和 Del 操作
	for i := range numGoroutines {
		go func(id int) {
			for j := range numOperations {
				key := id*numOperations + j
				vm.Set(key, key*2)
				vm.Del(key)
			}
		}(i)
	}

	// 通过检查版本号等待操作完成
	// 这是一个简化的检查 - 在实际测试中你可能想使用 sync.WaitGroup
	expectedMinVersion := initialVersion + uint64(numGoroutines*numOperations*2)

	// 留出一些时间让操作完成
	for vm.Version() < expectedMinVersion {
		// 忙等待 - 在实际测试中你会使用适当的同步机制
	}

	// 最终版本号应该至少是预期的最小值
	require.GreaterOrEqual(t, vm.Version(), expectedMinVersion)
	require.Equal(t, 0, vm.Len())
}
