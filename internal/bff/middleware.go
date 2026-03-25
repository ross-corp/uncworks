package bff

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Middleware is a function that wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// Chain composes multiple middleware into a single middleware.
// Middleware are applied in the order given: the first argument is the outermost wrapper.
func Chain(mws ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(mws) - 1; i >= 0; i-- {
			next = mws[i](next)
		}
		return next
	}
}

// ---------------------------------------------------------------------------
// 4.5  Request ID
// ---------------------------------------------------------------------------

// RequestIDMiddleware adds an X-Request-ID header to every request/response.
// If the incoming request already carries one, it is preserved.
func RequestIDMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-ID")
			if id == "" {
				id = uuid.New().String()
			}
			r.Header.Set("X-Request-ID", id)
			w.Header().Set("X-Request-ID", id)
			next.ServeHTTP(w, r)
		})
	}
}

// ---------------------------------------------------------------------------
// 4.4  CORS
// ---------------------------------------------------------------------------

// CORSMiddleware sets Access-Control-Allow-Origin, Methods, and Headers.
// It handles OPTIONS preflight requests by returning 204 No Content.
func CORSMiddleware(allowedOrigin string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, X-CSRF-Token")
			w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID, X-CSRF-Token")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ---------------------------------------------------------------------------
// 4.3  Rate limiting — token bucket per IP
// ---------------------------------------------------------------------------

type ipBucket struct {
	mu        sync.Mutex
	tokens    float64
	lastCheck time.Time
}

// RateLimitMiddleware enforces a per-IP request rate limit using a token
// bucket algorithm. Each IP gets reqPerSec tokens per second with a burst
// equal to reqPerSec. Returns 429 Too Many Requests when exceeded.
func RateLimitMiddleware(reqPerSec int) Middleware {
	var buckets sync.Map // map[string]*ipBucket
	rate := float64(reqPerSec)

	// Background sweep: evict inactive IP buckets every 5 minutes.
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		const bucketTTL = 10 * time.Minute
		for range ticker.C {
			now := time.Now()
			buckets.Range(func(key, val any) bool {
				bucket := val.(*ipBucket)
				bucket.mu.Lock()
				inactive := now.Sub(bucket.lastCheck) > bucketTTL
				bucket.mu.Unlock()
				if inactive {
					buckets.Delete(key)
				}
				return true
			})
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractIP(r)

			val, _ := buckets.LoadOrStore(ip, &ipBucket{
				tokens:    rate,
				lastCheck: time.Now(),
			})
			bucket := val.(*ipBucket)

			bucket.mu.Lock()
			now := time.Now()
			elapsed := now.Sub(bucket.lastCheck).Seconds()
			bucket.lastCheck = now

			// Refill tokens based on elapsed time.
			bucket.tokens += elapsed * rate
			if bucket.tokens > rate {
				bucket.tokens = rate
			}

			if bucket.tokens < 1 {
				bucket.mu.Unlock()
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			bucket.tokens--
			bucket.mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}

// extractIP returns the client IP address, stripping the port if present.
func extractIP(r *http.Request) string {
	// Prefer X-Forwarded-For when behind a proxy.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// ---------------------------------------------------------------------------
// 4.1  Session middleware — cookie-based, in-memory store
// ---------------------------------------------------------------------------

type sessionEntry struct {
	createdAt time.Time
}

// SessionMiddleware creates or validates a session cookie on each request.
// Sessions are stored in an in-memory map keyed by a random session ID.
// The cookie is HttpOnly, Secure, SameSite=Strict.
func SessionMiddleware(secret string) Middleware {
	var mu sync.RWMutex
	sessions := make(map[string]sessionEntry)
	_ = secret // reserved for future HMAC signing of session IDs

	// Background sweep: evict sessions older than 24 hours every 5 minutes.
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		const sessionTTL = 24 * time.Hour
		for range ticker.C {
			now := time.Now()
			mu.Lock()
			for sid, entry := range sessions {
				if now.Sub(entry.createdAt) > sessionTTL {
					delete(sessions, sid)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			const cookieName = "aot_session"

			cookie, err := r.Cookie(cookieName)
			if err != nil || cookie.Value == "" {
				// No valid session cookie — create a new session.
				sid := generateSessionID()
				mu.Lock()
				sessions[sid] = sessionEntry{createdAt: time.Now()}
				mu.Unlock()

				http.SetCookie(w, &http.Cookie{
					Name:     cookieName,
					Value:    sid,
					Path:     "/",
					HttpOnly: true,
					Secure:   true,
					SameSite: http.SameSiteStrictMode,
				})
				next.ServeHTTP(w, r)
				return
			}

			// Validate existing session.
			mu.RLock()
			_, exists := sessions[cookie.Value]
			mu.RUnlock()

			if !exists {
				// Session expired or unknown — issue a new one.
				sid := generateSessionID()
				mu.Lock()
				sessions[sid] = sessionEntry{createdAt: time.Now()}
				mu.Unlock()

				http.SetCookie(w, &http.Cookie{
					Name:     cookieName,
					Value:    sid,
					Path:     "/",
					HttpOnly: true,
					Secure:   true,
					SameSite: http.SameSiteStrictMode,
				})
			}

			next.ServeHTTP(w, r)
		})
	}
}

// generateSessionID returns a cryptographically random 32-hex-char string.
func generateSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback — should never happen on modern OSes.
		return uuid.New().String()
	}
	return hex.EncodeToString(b)
}

// ---------------------------------------------------------------------------
// 4.2  CSRF middleware
// ---------------------------------------------------------------------------

// csrfEntry holds a CSRF token and the time it was created.
type csrfEntry struct {
	token     string
	createdAt time.Time
}

// CSRFMiddleware generates a CSRF token per session and sets it in the
// X-CSRF-Token response header. State-changing methods (POST, PUT, DELETE)
// must include the matching token in the X-CSRF-Token request header.
// Requests that carry an Authorization header (API-key auth) skip CSRF
// verification since they are not susceptible to CSRF attacks.
func CSRFMiddleware() Middleware {
	var mu sync.RWMutex
	// Map of session cookie value → CSRF entry.
	tokens := make(map[string]csrfEntry)

	// Background sweep: evict CSRF entries older than 24 hours every 5 minutes.
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		const csrfTTL = 24 * time.Hour
		for range ticker.C {
			now := time.Now()
			mu.Lock()
			for sid, entry := range tokens {
				if now.Sub(entry.createdAt) > csrfTTL {
					delete(tokens, sid)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip CSRF for API-key authenticated requests.
			if r.Header.Get("Authorization") != "" {
				next.ServeHTTP(w, r)
				return
			}

			const cookieName = "aot_session"
			cookie, _ := r.Cookie(cookieName)
			sessionID := ""
			if cookie != nil {
				sessionID = cookie.Value
			}

			// Ensure a CSRF token exists for this session.
			mu.RLock()
			entry, exists := tokens[sessionID]
			mu.RUnlock()

			var token string
			if !exists || sessionID == "" {
				token = generateCSRFToken()
				if sessionID != "" {
					mu.Lock()
					tokens[sessionID] = csrfEntry{token: token, createdAt: time.Now()}
					mu.Unlock()
				}
			} else {
				token = entry.token
			}

			// Always expose the current CSRF token so the client can read it.
			w.Header().Set("X-CSRF-Token", token)

			// For state-changing methods, verify the token.
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete {
				reqToken := r.Header.Get("X-CSRF-Token")
				if reqToken == "" || reqToken != token {
					http.Error(w, "invalid or missing CSRF token", http.StatusForbidden)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// generateCSRFToken returns a cryptographically random 32-hex-char string.
func generateCSRFToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return uuid.New().String()
	}
	return hex.EncodeToString(b)
}
