package limiter

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// Limiter struct holds config for both Token Bucket and SWC.
type Limiter struct {
	Client *redis.Client
	Ctx    context.Context

	// Token Bucket Configuration (for instantaneous burst control)
	BucketCapacity int64         // Max burst size (N_burst)
	RefillRate     time.Duration // Time to refill one token (e.g., 6s for 10/min)

	// Sliding Window Counter Configuration (for long-term rate control)
	SWCLimit  int64         // Max requests in window (N_swc)
	SWCWindow time.Duration // Duration of the window (T_swc)
}

// NewLimiter is the constructor.
func NewLimiter(rdb *redis.Client, ctx context.Context) *Limiter {
	// Example configuration:
	// Token Bucket: Max burst of 10 requests. Refills 1 token every 6 seconds (10/min rate).
	// SWC: Limit of 100 requests per 60 minutes (3600 seconds).
	return &Limiter{
		Client:         rdb,
		Ctx:            ctx,
		BucketCapacity: 10,
		RefillRate:     time.Second * 6, // 60s / 10 tokens = 6s per token
		SWCLimit:       100,
		SWCWindow:      time.Minute * 60,
	}
}
