package csync

import (
	"encoding/json"
	"iter"
	"maps"
	"sync"
)

// Map 是一个并发映射实现，提供线程安全的访问。
type Map[K comparable, V any] struct {
	inner map[K]V
	mu    sync.RWMutex
}

// NewMap 创建一个新的线程安全映射，具有指定的键和值类型。
func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		inner: make(map[K]V),
	}
}

// NewMapFrom 从现有映射创建一个新的线程安全映射。
func NewMapFrom[K comparable, V any](m map[K]V) *Map[K, V] {
	return &Map[K, V]{
		inner: m,
	}
}

// NewLazyMap 创建一个新的延迟加载映射。提供的加载函数在
// 单独的 goroutine 中执行以填充映射。
func NewLazyMap[K comparable, V any](load func() map[K]V) *Map[K, V] {
	m := &Map[K, V]{}
	m.mu.Lock()
	go func() {
		defer m.mu.Unlock()
		m.inner = load()
	}()
	return m
}

// Reset 用新的映射替换内部映射。
func (m *Map[K, V]) Reset(input map[K]V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inner = input
}

// Set 在映射中为指定的键设置值。
func (m *Map[K, V]) Set(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inner[key] = value
}

// Del 从映射中删除指定的键。
func (m *Map[K, V]) Del(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.inner, key)
}

// Get 从映射中获取指定键的值。
func (m *Map[K, V]) Get(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.inner[key]
	return v, ok
}

// Len 返回映射中的项目数量。
func (m *Map[K, V]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.inner)
}

// GetOrSet 如果键存在则获取并返回该键的值，否则执行给定的函数，
// 将其返回值设置为给定键的值，并返回该值。
func (m *Map[K, V]) GetOrSet(key K, fn func() V) V {
	got, ok := m.Get(key)
	if ok {
		return got
	}
	value := fn()
	m.Set(key, value)
	return value
}

// Take 获取一个项目然后删除它。
func (m *Map[K, V]) Take(key K) (V, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.inner[key]
	delete(m.inner, key)
	return v, ok
}

// Copy 返回内部映射的副本。
func (m *Map[K, V]) Copy() map[K]V {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return maps.Clone(m.inner)
}

// Seq2 返回一个 iter.Seq2，从映射中产生键值对。
func (m *Map[K, V]) Seq2() iter.Seq2[K, V] {
	dst := m.Copy()
	return func(yield func(K, V) bool) {
		for k, v := range dst {
			if !yield(k, v) {
				return
			}
		}
	}
}

// Seq 返回一个 iter.Seq，从映射中产生值。
func (m *Map[K, V]) Seq() iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, v := range m.Seq2() {
			if !yield(v) {
				return
			}
		}
	}
}

var (
	_ json.Unmarshaler = &Map[string, any]{}
	_ json.Marshaler   = &Map[string, any]{}
)

func (Map[K, V]) JSONSchemaAlias() any { //nolint
	m := map[K]V{}
	return m
}

// UnmarshalJSON 实现 json.Unmarshaler 接口。
func (m *Map[K, V]) UnmarshalJSON(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inner = make(map[K]V)
	return json.Unmarshal(data, &m.inner)
}

// MarshalJSON 实现 json.Marshaler 接口。
func (m *Map[K, V]) MarshalJSON() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return json.Marshal(m.inner)
}
