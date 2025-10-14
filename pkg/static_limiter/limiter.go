package limiter

import (
	"context"
	"fmt"
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
	// Configuration:
	// TB: Max burst of 10 requests. Refills 1 token every 6 seconds (10/min rate).
	// SWC: Limit of 100 requests per 60 minutes.
	return &Limiter{
		Client:         rdb,
		Ctx:            ctx,
		BucketCapacity: 10,
		RefillRate:     time.Second * 6, // 60s / 10 tokens = 6s per token
		SWCLimit:       100,
		SWCWindow:      time.Minute * 60, // 60 minutes
	}
}

// Key generates the unique key for the SWC window in Redis.
func (l *Limiter) Key(identifier string, windowTime time.Time, prefix string) string {
	// Truncate time to the nearest start of the window duration.
	windowStart := windowTime.Truncate(l.SWCWindow)
	return fmt.Sprintf("%s:%s:%d", prefix, identifier, windowStart.Unix())
}

// refillBucket is the O(1) Token Bucket logic.
// It calculates the new token count based on time elapsed since last check.
func (l *Limiter) refillBucket(bucketKey string, lastRefillKey string) (int64, error) {
	// 1. Get current token count and last refill time atomically
	pipe := l.Client.Pipeline()
	currentTokensCmd := pipe.Get(l.Ctx, bucketKey)
	lastRefillTimeCmd := pipe.Get(l.Ctx, lastRefillKey)
	_, err := pipe.Exec(l.Ctx)
	if err != nil && err != redis.Nil {
		return 0, err
	}

	now := time.Now()

	currentTokens, _ := currentTokensCmd.Int64()
	lastRefillTimeUnix, _ := lastRefillTimeCmd.Int64()

	// Handle Initial State (First Request)
	if currentTokensCmd.Err() == redis.Nil {
		currentTokens = l.BucketCapacity
	}
	if lastRefillTimeCmd.Err() == redis.Nil {
		lastRefillTimeUnix = now.UnixNano()
	}

	lastRefillTime := time.Unix(0, lastRefillTimeUnix)

	// 2. Calculate tokens to add
	timeElapsed := now.Sub(lastRefillTime)
	tokensToAdd := int64(timeElapsed.Nanoseconds() / l.RefillRate.Nanoseconds())

	newTokens := currentTokens + tokensToAdd

	// 3. Clamp newTokens at max capacity
	if newTokens > l.BucketCapacity {
		newTokens = l.BucketCapacity
	}

	// 4. Calculate the new 'Last Refill Time' (advancing it only by the time used for refilling)
	newLastRefillTime := lastRefillTime.Add(time.Duration(tokensToAdd) * l.RefillRate)

	// 5. Update Redis with the new state (Atomic Write & Expiration)
	pipe = l.Client.Pipeline()
	pipe.Set(l.Ctx, bucketKey, newTokens, 0)
	pipe.Set(l.Ctx, lastRefillKey, newLastRefillTime.UnixNano(), 0)

	// Set long expiration to clean up inactive users
	pipe.Expire(l.Ctx, bucketKey, time.Hour*2)
	pipe.Expire(l.Ctx, lastRefillKey, time.Hour*2)

	_, err = pipe.Exec(l.Ctx)
	if err != nil && err != redis.Nil {
		return 0, err
	}

	// Return the tokens BEFORE consumption
	return newTokens, nil
}

// Allow implements the Token Bucket + Sliding Window Counter hybrid logic.
func (l *Limiter) Allow(identifier string) bool {
	// --- CHECK 1: TOKEN BUCKET (Burst Defense) ---
	bucketKey := fmt.Sprintf("tb_tokens:%s", identifier)
	lastRefillKey := fmt.Sprintf("tb_refill:%s", identifier)

	currentTokens, err := l.refillBucket(bucketKey, lastRefillKey)
	if err != nil {
		fmt.Printf("[TOKEN BUCKET ERROR] Allowing request: %v\n", err)
		return true // Fail safe
	}

	if currentTokens < 1 {
		fmt.Printf("[DENIED - TB] ID: %s. No tokens available for burst limit.\n", identifier)
		return false // DENY due to burst limit exhaustion
	}

	// --- CHECK 2: SLIDING WINDOW COUNTER (Long-Term Rate Defense) ---
	now := time.Now()
	currentWindowKey := l.Key(identifier, now, "swc_count")
	previousWindowKey := l.Key(identifier, now.Add(-l.SWCWindow), "swc_count")

	// Calculate Overlap Percentage
	timeElapsedInCurrentWindow := now.Sub(now.Truncate(l.SWCWindow))
	overlap := 1.0 - (float64(timeElapsedInCurrentWindow) / float64(l.SWCWindow))

	// Fetch counts using Pipelining for O(1) efficiency
	pipe := l.Client.Pipeline()
	currentCountCmd := pipe.Get(l.Ctx, currentWindowKey)
	previousCountCmd := pipe.Get(l.Ctx, previousWindowKey)
	_, err = pipe.Exec(l.Ctx)
	if err != nil && err != redis.Nil {
		fmt.Printf("[SWC ERROR] Allowing request: %v\n", err)
		return true // Fail safe
	}

	currentCount, _ := currentCountCmd.Int64()
	previousCount, _ := previousCountCmd.Int64()

	// Calculate the Estimated Count
	estimatedCount := int64(float64(previousCount)*overlap) + currentCount

	// --- FINAL DECISION ---
	if estimatedCount < l.SWCLimit {
		// ALLOWED: Deduct 1 token and increment the SWC counter.

		// 1. Token Bucket consumption (Deduct 1 token)
		l.Client.Decr(l.Ctx, bucketKey)

		// 2. SWC increment
		l.Client.Incr(l.Ctx, currentWindowKey)

		// 3. Set expiration on the SWC key
		l.Client.Expire(l.Ctx, currentWindowKey, l.SWCWindow+time.Minute)

		fmt.Printf("[ALLOWED] ID: %s. Est. Count: %d/%d (Tokens Left: %d)\n",
			identifier, estimatedCount+1, l.SWCLimit, currentTokens-1)
		return true

	} else {
		// DENIED
		fmt.Printf("[DENIED - SWC] ID: %s. Exceeded long-term rate limit of %d (Est. %d).\n",
			identifier, l.SWCLimit, estimatedCount)
		return false
	}
}
