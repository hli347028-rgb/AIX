package service

import (
	"sync"
	"time"
)

// tokenBucket 进程内令牌桶。
//
// 后端是 systemd 单进程单实例，因此不需要 Redis 之类的分布式限流；
// 若将来横向扩容，这里的配额会变成「每实例」而非「全局」，需要一并调整。
type tokenBucket struct {
	mu       sync.Mutex
	tokens   float64
	capacity float64
	rate     float64 // 每秒补充的令牌数
	last     time.Time
}

func newTokenBucket(ratePerSec int) *tokenBucket {
	if ratePerSec <= 0 {
		ratePerSec = 1
	}
	r := float64(ratePerSec)
	return &tokenBucket{
		tokens:   r,
		capacity: r,
		rate:     r,
		last:     time.Now(),
	}
}

// allow 消耗一个令牌，返回是否放行。
func (b *tokenBucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	b.tokens += now.Sub(b.last).Seconds() * b.rate
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	b.last = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// rateLimiter 按 key 维护多个令牌桶。
type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	lastGC  time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{buckets: make(map[string]*tokenBucket), lastGC: time.Now()}
}

// bucketIdleTTL 空闲桶的回收阈值。按 IP 限速时 key 的基数不受控，
// 不回收会让 map 随攻击流量无限增长。
const bucketIdleTTL = 10 * time.Minute

func (l *rateLimiter) allow(key string, ratePerSec int) bool {
	if key == "" {
		key = "(unknown)"
	}
	l.mu.Lock()
	b, ok := l.buckets[key]
	if !ok {
		b = newTokenBucket(ratePerSec)
		l.buckets[key] = b
	}
	if time.Since(l.lastGC) > bucketIdleTTL {
		l.gcLocked()
	}
	l.mu.Unlock()

	return b.allow()
}

func (l *rateLimiter) gcLocked() {
	cutoff := time.Now().Add(-bucketIdleTTL)
	for k, b := range l.buckets {
		b.mu.Lock()
		idle := b.last.Before(cutoff)
		b.mu.Unlock()
		if idle {
			delete(l.buckets, k)
		}
	}
	l.lastGC = time.Now()
}
