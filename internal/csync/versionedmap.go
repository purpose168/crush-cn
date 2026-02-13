package csync

import (
	"iter"
	"sync/atomic"
)

// NewVersionedMap 创建一个新的带版本号的线程安全映射。
func NewVersionedMap[K comparable, V any]() *VersionedMap[K, V] {
	return &VersionedMap[K, V]{
		m: NewMap[K, V](),
	}
}

// VersionedMap 是一个跟踪其版本号的线程安全映射。
type VersionedMap[K comparable, V any] struct {
	m *Map[K, V]
	v atomic.Uint64
}

// Get 从映射中获取指定键的值。
func (m *VersionedMap[K, V]) Get(key K) (V, bool) {
	return m.m.Get(key)
}

// Set 在映射中设置指定键的值并递增版本号。
func (m *VersionedMap[K, V]) Set(key K, value V) {
	m.m.Set(key, value)
	m.v.Add(1)
}

// Del 从映射中删除指定键并递增版本号。
func (m *VersionedMap[K, V]) Del(key K) {
	m.m.Del(key)
	m.v.Add(1)
}

// Seq2 返回一个 iter.Seq2，用于生成映射中的键值对。
func (m *VersionedMap[K, V]) Seq2() iter.Seq2[K, V] {
	return m.m.Seq2()
}

// Copy 返回内部映射的副本。
func (m *VersionedMap[K, V]) Copy() map[K]V {
	return m.m.Copy()
}

// Len 返回映射中的元素数量。
func (m *VersionedMap[K, V]) Len() int {
	return m.m.Len()
}

// Version 返回映射的当前版本号。
func (m *VersionedMap[K, V]) Version() uint64 {
	return m.v.Load()
}
