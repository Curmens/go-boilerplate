package middleware

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"example.com/utils"

	"github.com/gin-gonic/gin"
)

// ipWindow tracks request count within a fixed time window for a single IP.
type ipWindow struct {
	count     int
	windowEnd time.Time
}

// rateLimiterStore is a per-IP fixed-window rate limiter backed by an in-memory map.
// It is per-process: behind multiple replicas, move this to Redis.
type rateLimiterStore struct {
	mu      sync.Mutex
	clients map[string]*ipWindow
	limit   int
	window  time.Duration
}

func newRateLimiterStore(limit int, window time.Duration) *rateLimiterStore {
	s := &rateLimiterStore{
		clients: make(map[string]*ipWindow),
		limit:   limit,
		window:  window,
	}
	go s.cleanup()
	return s
}

// allow records a request and reports whether it is permitted, plus the time
// until the current window resets.
func (s *rateLimiterStore) allow(ip string) (bool, time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	w, ok := s.clients[ip]
	if !ok || now.After(w.windowEnd) {
		s.clients[ip] = &ipWindow{count: 1, windowEnd: now.Add(s.window)}
		return true, s.window
	}
	if w.count >= s.limit {
		return false, w.windowEnd.Sub(now)
	}
	w.count++
	return true, w.windowEnd.Sub(now)
}

// cleanup removes expired IP entries periodically to prevent unbounded memory growth.
func (s *rateLimiterStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for ip, w := range s.clients {
			if now.After(w.windowEnd) {
				delete(s.clients, ip)
			}
		}
		s.mu.Unlock()
	}
}

// RateLimitMiddleware limits each client IP to `limit` requests per `window`.
// Exceeding the limit returns 429 with a Retry-After header.
func RateLimitMiddleware(limit int, window time.Duration) gin.HandlerFunc {
	store := newRateLimiterStore(limit, window)
	return func(c *gin.Context) {
		ok, reset := store.allow(c.ClientIP())
		if !ok {
			c.Header("Retry-After", strconv.Itoa(int(math.Ceil(reset.Seconds()))))
			utils.AbortWithError(c, http.StatusTooManyRequests, utils.ErrCodeTooManyRequest, "Too many requests, please try again later")
			return
		}
		c.Next()
	}
}

// MaxBodySizeMiddleware rejects requests whose body exceeds maxBytes.
// Apply globally to prevent resource exhaustion from oversized payloads.
func MaxBodySizeMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
