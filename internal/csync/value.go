package csync

import (
	"reflect"
	"sync"
)

// Value 是一个泛型线程安全包装器，用于任意值类型。
//
// 对于切片，请使用 [Slice]。对于映射，请使用 [Map]。不支持指针类型。
type Value[T any] struct {
	v  T
	mu sync.RWMutex
}

// NewValue 使用给定的初始值创建一个新的 Value。
//
// 如果 t 是指针、切片或映射类型，则会触发 panic。请使用对应的专用类型。
func NewValue[T any](t T) *Value[T] {
	v := reflect.ValueOf(t)
	switch v.Kind() {
	case reflect.Pointer:
		panic("csync.Value 不支持指针类型")
	case reflect.Slice:
		panic("csync.Value 不支持切片类型；请使用 csync.Slice")
	case reflect.Map:
		panic("csync.Value 不支持映射类型；请使用 csync.Map")
	}
	return &Value[T]{v: t}
}

// Get 返回当前值。
func (v *Value[T]) Get() T {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.v
}

// Set 更新值。
func (v *Value[T]) Set(t T) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.v = t
}
