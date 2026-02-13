// Package logo 提供随机数缓存功能，用于logo渲染过程中的随机选择。
package logo

import (
	"math/rand/v2"
	"sync"
)

// randCaches 是随机数缓存映射，存储已生成的随机数以避免重复计算。
// 键为上限值n，值为对应的随机数结果。
var (
	randCaches   = make(map[int]int)
	randCachesMu sync.Mutex // randCachesMu 是保护 randCaches 并发访问的互斥锁
)

// cachedRandN 返回一个范围在 [0, n) 内的随机整数。
// 该函数使用缓存机制，对于相同的输入参数n，始终返回相同的随机数结果，
// 确保在多次渲染时保持一致性。
//
// 参数：
//   - n: 随机数的上限（不包含），必须为正整数
//
// 返回：
//   - 一个范围在 [0, n) 内的随机整数
func cachedRandN(n int) int {
	randCachesMu.Lock()
	defer randCachesMu.Unlock()

	// 检查缓存中是否已存在该n对应的随机数
	if n, ok := randCaches[n]; ok {
		return n
	}

	// 生成新的随机数并存入缓存
	r := rand.IntN(n)
	randCaches[n] = r
	return r
}
