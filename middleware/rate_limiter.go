package middleware

import (
	"net/http"
	"sync"
	"time"

	"finance-dashboard/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// TODO: Replace with Redis sliding window for horizontal scaling.
// Current implementation limits deployment to a single instance — behind a
// load balancer with N instances, a client effectively gets N × 100 req/min.
// Redis-based approach: ZADD rate_limit:{ip} {now} {requestID} /
// ZREMRANGEBYSCORE / ZCARD, sharing state across all instances.

// rateLimitEntry wraps a rate.Limiter with a last-seen timestamp so stale
// entries can be evicted, preventing unbounded memory growth from many unique IPs.
type rateLimitEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	limiters     sync.Map
	evictionOnce sync.Once
)

const (
	// evictionInterval controls how often the background cleanup runs.
	evictionInterval = 60 * time.Second
	// entryTTL is how long an IP's rate limiter is kept after its last request.
	entryTTL = 10 * time.Minute
)

// startEviction launches a background goroutine (once) that periodically
// removes rate limiter entries that haven't been seen within the TTL window.
func startEviction() {
	evictionOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(evictionInterval)
			defer ticker.Stop()

			for range ticker.C {
				now := time.Now()
				limiters.Range(func(key, value interface{}) bool {
					entry := value.(*rateLimitEntry)
					if now.Sub(entry.lastSeen) > entryTTL {
						limiters.Delete(key)
					}
					return true
				})
			}
		}()
	})
}

func getLimiter(ip string) *rate.Limiter {
	if v, ok := limiters.Load(ip); ok {
		entry := v.(*rateLimitEntry)
		entry.lastSeen = time.Now()
		return entry.limiter
	}
	// 100 requests per minute ≈ ~1.67 requests/second, burst of 100.
	limiter := rate.NewLimiter(rate.Limit(100.0/60.0), 100)
	entry := &rateLimitEntry{
		limiter:  limiter,
		lastSeen: time.Now(),
	}
	limiters.Store(ip, entry)
	return limiter
}

// RateLimiter restricts each client IP to 100 requests per minute.
// Rate limiter entries are evicted after 10 minutes of inactivity to prevent
// unbounded memory growth from many unique client IPs.
func RateLimiter() gin.HandlerFunc {
	startEviction()

	return func(c *gin.Context) {
		if !getLimiter(c.ClientIP()).Allow() {
			utils.Error(c, http.StatusTooManyRequests, "rate limit exceeded, please try again later")
			c.Abort()
			return
		}
		c.Next()
	}
}
