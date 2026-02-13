package csync

import (
	"iter"
	"sync"
)

// LazySlice 是一个线程安全的延迟加载切片。
type LazySlice[K any] struct {
	inner []K
	wg    sync.WaitGroup
}

// NewLazySlice 创建一个新切片，并在 goroutine 中运行 [load] 函数来填充它。
func NewLazySlice[K any](load func() []K) *LazySlice[K] {
	s := &LazySlice[K]{}
	s.wg.Go(func() {
		s.inner = load()
	})
	return s
}

// Seq 返回一个迭代器，用于从切片中产出元素。
func (s *LazySlice[K]) Seq() iter.Seq[K] {
	s.wg.Wait()
	return func(yield func(K) bool) {
		for _, v := range s.inner {
			if !yield(v) {
				return
			}
		}
	}
}

// Slice 是一个线程安全的切片实现，提供并发访问能力。
type Slice[T any] struct {
	inner []T
	mu    sync.RWMutex
}

// NewSlice 创建一个新的线程安全切片。
func NewSlice[T any]() *Slice[T] {
	return &Slice[T]{
		inner: make([]T, 0),
	}
}

// NewSliceFrom 从现有切片创建一个新的线程安全切片。
func NewSliceFrom[T any](s []T) *Slice[T] {
	inner := make([]T, len(s))
	copy(inner, s)
	return &Slice[T]{
		inner: inner,
	}
}

// Append 在切片末尾添加一个或多个元素。
func (s *Slice[T]) Append(items ...T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inner = append(s.inner, items...)
}

// Get 返回指定索引位置的元素。
func (s *Slice[T]) Get(index int) (T, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var zero T
	if index < 0 || index >= len(s.inner) {
		return zero, false
	}
	return s.inner[index], true
}

// Len 返回切片中元素的数量。
func (s *Slice[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.inner)
}

// SetSlice 用新的切片替换整个切片。
func (s *Slice[T]) SetSlice(items []T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inner = make([]T, len(items))
	copy(s.inner, items)
}

// Seq 返回一个迭代器，用于从切片中产出元素。
func (s *Slice[T]) Seq() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, v := range s.Seq2() {
			if !yield(v) {
				return
			}
		}
	}
}

// Seq2 返回一个迭代器，用于从切片中产出索引-值对。
func (s *Slice[T]) Seq2() iter.Seq2[int, T] {
	items := s.Copy()
	return func(yield func(int, T) bool) {
		for i, v := range items {
			if !yield(i, v) {
				return
			}
		}
	}
}

// Copy 返回内部切片的副本。
func (s *Slice[T]) Copy() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]T, len(s.inner))
	copy(items, s.inner)
	return items
}
