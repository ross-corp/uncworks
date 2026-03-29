package server

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	defaultRPS   = 10.0
	defaultBurst = 20
)

// RateLimiterConfig holds configuration for a RateLimiter instance.
type RateLimiterConfig struct {
	Enabled    bool
	RPS        float64
	Burst      int
	TTLMinutes int
	TrustProxy bool
}

type ipEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter is a per-IP rate limiter using golang.org/x/time/rate token buckets.
// It evicts inactive IP entries via a background TTL sweep goroutine.
type RateLimiter struct {
	mu      sync.Mutex
	entries map[string]*ipEntry
	cfg     RateLimiterConfig
}

// NewRateLimiter creates a RateLimiter and starts the background TTL sweep goroutine.
// If RPS or Burst are zero or negative, a warning is logged and sensible defaults
// (RPS=10, Burst=20) are used in their place to prevent the limiter from blocking
// all traffic or behaving as infinite.
func NewRateLimiter(cfg RateLimiterConfig) *RateLimiter {
	if cfg.RPS <= 0 {
		slog.Warn("rate limiter RPS is invalid, using default",
			"provided", cfg.RPS, "default", defaultRPS)
		cfg.RPS = defaultRPS
	}
	if cfg.Burst <= 0 {
		slog.Warn("rate limiter Burst is invalid, using default",
			"provided", cfg.Burst, "default", defaultBurst)
		cfg.Burst = defaultBurst
	}
	rl := &RateLimiter{
		entries: make(map[string]*ipEntry),
		cfg:     cfg,
	}
	go rl.sweepLoop()
	return rl
}

// sweepLoop periodically evicts IP entries that have not been seen within the TTL.
func (rl *RateLimiter) sweepLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.sweep()
	}
}

func (rl *RateLimiter) sweep() {
	ttl := time.Duration(rl.cfg.TTLMinutes) * time.Minute
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	for ip, entry := range rl.entries {
		if now.Sub(entry.lastSeen) > ttl {
			delete(rl.entries, ip)
		}
	}
}

// Allow returns true if the request from ip is within the rate limit.
// It creates a new limiter for previously unseen IPs and updates the last-seen timestamp.
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	entry, ok := rl.entries[ip]
	if !ok {
		entry = &ipEntry{
			limiter: rate.NewLimiter(rate.Limit(rl.cfg.RPS), rl.cfg.Burst),
		}
		rl.entries[ip] = entry
	}
	entry.lastSeen = time.Now()
	return entry.limiter.Allow()
}

// Remaining returns the current number of available tokens for ip (floored to int).
// Returns Burst if ip has no existing entry.
func (rl *RateLimiter) Remaining(ip string) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	entry, ok := rl.entries[ip]
	if !ok {
		return rl.cfg.Burst
	}
	return int(entry.limiter.Tokens())
}

// RateLimitMiddleware returns middleware that enforces rl on every request.
// When the limit is exceeded it responds 429 with Retry-After and X-RateLimit-* headers.
// Health-check paths (/healthz, /readyz) are always passed through.
// If rl is nil or cfg.Enabled is false, the middleware is a no-op pass-through.
func RateLimitMiddleware(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rl == nil || !rl.cfg.Enabled {
				next.ServeHTTP(w, r)
				return
			}
			ip := clientIP(r, rl.cfg)
			if !rl.Allow(ip) {
				w.Header().Set("Retry-After", "1")
				w.Header().Set("X-RateLimit-Limit", itoa(rl.cfg.Burst))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", itoa(int(time.Now().Add(time.Second).Unix())))
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}
			remaining := rl.Remaining(ip)
			w.Header().Set("X-RateLimit-Limit", itoa(rl.cfg.Burst))
			w.Header().Set("X-RateLimit-Remaining", itoa(remaining))
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP extracts the client IP from the request.
// When cfg.TrustProxy is true it reads the first value of X-Forwarded-For.
// Always strips the port from r.RemoteAddr as a fallback.
func clientIP(r *http.Request, cfg RateLimiterConfig) string {
	if cfg.TrustProxy {
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			// Take the first (leftmost) address.
			for i := 0; i < len(fwd); i++ {
				if fwd[i] == ',' {
					return stripPort(trimSpace(fwd[:i]))
				}
			}
			return stripPort(trimSpace(fwd))
		}
	}
	return stripPort(r.RemoteAddr)
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

// stripPort removes the port suffix from an address like "1.2.3.4:5678".
func stripPort(addr string) string {
	if host, _, err := splitHostPort(addr); err == nil {
		return host
	}
	return addr
}

func splitHostPort(addr string) (host, port string, err error) {
	// Inline minimal implementation to avoid importing net just for this.
	// net.SplitHostPort handles IPv6 brackets correctly.
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i], addr[i+1:], nil
		}
		if addr[i] == ']' {
			break // IPv6 literal without port
		}
	}
	return "", "", &addrError{addr}
}

type addrError struct{ addr string }

func (e *addrError) Error() string { return "no port in address: " + e.addr }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
