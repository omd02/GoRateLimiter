package adaptive

import (
	"sync"

	"golang.org/x/time/rate"
)

// AdaptiveLimiter manages the dynamic rate limit based on a calculated factor.
type AdaptiveLimiter struct {
	mu                sync.RWMutex
	BaseLimit         float64
	underlyingLimiter *rate.Limiter
}

// NewAdaptiveLimiter creates a new limiter with a starting rate.
func NewAdaptiveLimiter(baseLimit float64) *AdaptiveLimiter {
	initialRate := rate.Limit(baseLimit)

	return &AdaptiveLimiter{
		BaseLimit:         baseLimit,
		underlyingLimiter: rate.NewLimiter(initialRate, int(baseLimit)),
	}
}

// Allow is the primary method called by the HTTP middleware.
func (l *AdaptiveLimiter) Allow() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.underlyingLimiter.Allow()
}

// UpdateFactor is the key method called by the Health Monitor to adjust the rate.
func (l *AdaptiveLimiter) UpdateFactor(factor float64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	newRate := l.BaseLimit * factor

	// Dynamically change the rate of the underlying limiter
	l.underlyingLimiter.SetLimit(rate.Limit(newRate))
}
