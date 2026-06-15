// Package webbook 网络书籍流程编排
// 核心功能：多书源并发搜索、书籍信息/目录/正文获取的完整流程编排
package webbook

import (
	"sync"
	"time"
)

// =============================================================================
// 滑动窗口限流器（服务层：多书源并发搜索）
// =============================================================================

// SlidingWindowLimiter 滑动窗口限流器
// 用于控制同时发起的搜索请求数量，避免对书源造成过大压力
type SlidingWindowLimiter struct {
	mu         sync.Mutex
	cond       *sync.Cond      // 条件变量，替代忙等待
	semaphore  chan struct{}   // 信号量，控制并发数
	windowSize int             // 窗口大小（并发上限）
	window     []time.Time     // 时间窗口记录
	windowSizeMs int            // 窗口时长（毫秒）
}

// NewSlidingWindowLimiter 创建滑动窗口限流器
// maxConcurrent: 最大并发搜索数
// windowMs: 滑动窗口时长（毫秒）
func NewSlidingWindowLimiter(maxConcurrent int, windowMs int) *SlidingWindowLimiter {
	if maxConcurrent <= 0 {
		maxConcurrent = 1
	}
	if windowMs <= 0 {
		windowMs = 1000
	}
	l := &SlidingWindowLimiter{
		semaphore:    make(chan struct{}, maxConcurrent),
		windowSize:   maxConcurrent,
		window:       make([]time.Time, 0, maxConcurrent),
		windowSizeMs: windowMs,
	}
	l.cond = sync.NewCond(&l.mu)
	return l
}

// Allow 尝试获取许可，返回是否允许立即执行
func (l *SlidingWindowLimiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-time.Duration(l.windowSizeMs) * time.Millisecond)

	// 清理窗口外记录
	newWindow := make([]time.Time, 0, len(l.window))
	for _, t := range l.window {
		if t.After(windowStart) {
			newWindow = append(newWindow, t)
		}
	}
	l.window = newWindow

	// 检查当前窗口内请求数是否已达上限
	if len(l.window) >= l.windowSize {
		return false
	}

	l.window = append(l.window, now)
	return true
}

// cleanupExpired 清理过期的窗口记录并唤醒等待者
func (l *SlidingWindowLimiter) cleanupExpired() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-time.Duration(l.windowSizeMs) * time.Millisecond)

	newWindow := make([]time.Time, 0, len(l.window))
	for _, t := range l.window {
		if t.After(windowStart) {
			newWindow = append(newWindow, t)
		}
	}
	l.window = newWindow

	// 如果有空间，唤醒所有等待者
	if len(l.window) < l.windowSize {
		l.cond.Broadcast()
	}
}

// Acquire 获取许可，如果当前不允许则阻塞等待
func (l *SlidingWindowLimiter) Acquire() {
	l.semaphore <- struct{}{}
}

// Release 释放许可
func (l *SlidingWindowLimiter) Release() {
	<-l.semaphore
}

// Wait 等待直到允许执行（使用 sync.Cond 避免忙等待）
func (l *SlidingWindowLimiter) Wait() {
	l.mu.Lock()
	defer l.mu.Unlock()

	for {
		now := time.Now()
		windowStart := now.Add(-time.Duration(l.windowSizeMs) * time.Millisecond)

		// 清理窗口外记录
		newWindow := make([]time.Time, 0, len(l.window))
		for _, t := range l.window {
			if t.After(windowStart) {
				newWindow = append(newWindow, t)
			}
		}
		l.window = newWindow

		// 检查当前窗口内请求数是否已达上限
		if len(l.window) < l.windowSize {
			l.window = append(l.window, now)
			return
		}

		// 计算等待时间（到最早记录过期）
		if len(l.window) > 0 {
			waitTime := l.window[0].Add(time.Duration(l.windowSizeMs) * time.Millisecond).Sub(now)
			if waitTime > 0 {
				l.cond.Wait()
			}
		} else {
			l.cond.Wait()
		}
	}
}

// =============================================================================
// 书源频率限制器（书源层：单书源频率限制）
// =============================================================================

// RateLimiter 令牌桶频率限制器
// 用于控制单个书源的请求频率，避免触发反爬
type RateLimiter struct {
	mu           sync.Mutex
	rate         float64       // 每秒请求数
	burst        int           // 桶容量（突发上限）
	tokens       float64       // 当前令牌数
	lastRefill   time.Time     // 上次 refill 时间
}

// NewRateLimiter 创建频率限制器
// rate: 每秒请求数
// burst: 突发上限（桶容量）
func NewRateLimiter(rate float64, burst int) *RateLimiter {
	if rate <= 0 {
		rate = 1
	}
	if burst <= 0 {
		burst = 1
	}
	return &RateLimiter{
		rate:       rate,
		burst:      burst,
		tokens:     float64(burst),
		lastRefill: time.Now(),
	}
}

// Wait 等待直到有可用令牌
func (rl *RateLimiter) Wait() {
	for {
		if rl.Allow() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// Allow 尝试获取令牌，返回是否允许立即执行
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()

	//  refill 令牌
	rl.tokens += elapsed * rl.rate
	if rl.tokens > float64(rl.burst) {
		rl.tokens = float64(rl.burst)
	}
	rl.lastRefill = now

	if rl.tokens >= 1 {
		rl.tokens -= 1
		return true
	}
	return false
}

// =============================================================================
// 并发控制记录（两层并发控制）
// =============================================================================

// ConcurrentRecord 并发控制记录
// 管理服务层和书源层两层并发控制
type ConcurrentRecord struct {
	// 服务层：多书源并发搜索限流
	serviceLimiter *SlidingWindowLimiter

	// 书源层：按书源ID的频率限制器
	sourceLimiters map[string]*RateLimiter
	sourceLimiterMu sync.RWMutex

	// 全局并发控制配置
	maxConcurrent int       // 最大并发搜索数
	windowMs      int       // 滑动窗口时长
	defaultRate   float64   // 默认书源请求速率
	defaultBurst  int       // 默认书源突发上限
}

// NewConcurrentRecord 创建并发控制器
func NewConcurrentRecord(maxConcurrent int, windowMs int, defaultRate float64, defaultBurst int) *ConcurrentRecord {
	return &ConcurrentRecord{
		serviceLimiter: NewSlidingWindowLimiter(maxConcurrent, windowMs),
		sourceLimiters: make(map[string]*RateLimiter),
		maxConcurrent:  maxConcurrent,
		windowMs:       windowMs,
		defaultRate:    defaultRate,
		defaultBurst:   defaultBurst,
	}
}

// GetSourceLimiter 获取或创建书源频率限制器
func (cr *ConcurrentRecord) GetSourceLimiter(sourceID string) *RateLimiter {
	cr.sourceLimiterMu.RLock()
	rl, ok := cr.sourceLimiters[sourceID]
	cr.sourceLimiterMu.RUnlock()
	if ok {
		return rl
	}

	cr.sourceLimiterMu.Lock()
	defer cr.sourceLimiterMu.Unlock()

	// 双重检查
	if rl, ok = cr.sourceLimiters[sourceID]; ok {
		return rl
	}

	rl = NewRateLimiter(cr.defaultRate, cr.defaultBurst)
	cr.sourceLimiters[sourceID] = rl
	return rl
}

// SetSourceRate 设置指定书源的频率限制
func (cr *ConcurrentRecord) SetSourceRate(sourceID string, rate float64, burst int) {
	cr.sourceLimiterMu.Lock()
	defer cr.sourceLimiterMu.Unlock()

	cr.sourceLimiters[sourceID] = NewRateLimiter(rate, burst)
}

// ServiceWait 服务层等待（滑动窗口）
func (cr *ConcurrentRecord) ServiceWait() {
	cr.serviceLimiter.Wait()
}

// ServiceRelease 服务层释放
func (cr *ConcurrentRecord) ServiceRelease() {
	cr.serviceLimiter.Release()
}

// SourceWait 书源层等待（频率限制）
func (cr *ConcurrentRecord) SourceWait(sourceID string) {
	rl := cr.GetSourceLimiter(sourceID)
	rl.Wait()
}

// SourceAllow 书源层尝试允许（非阻塞）
func (cr *ConcurrentRecord) SourceAllow(sourceID string) bool {
	rl := cr.GetSourceLimiter(sourceID)
	return rl.Allow()
}
