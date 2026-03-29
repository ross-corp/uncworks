// test/bdd/auth_ratelimit_spec_test.go — BDD scenarios for auth boundaries and rate limiting.
//
// Scenarios covered:
//   - No token → 401
//   - Wrong token → 401
//   - Valid token → 200
//   - Health/webhook endpoints bypass Bearer auth
//   - Rate limit burst exhaustion → 429
//   - Per-IP isolation: exhausting one IP does not affect another
//   - Disabled rate limiter passes all requests through
package bdd_test

import (
	"crypto/subtle"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	"github.com/uncworks/aot/internal/server"
)

// withBDDAuth replicates the withAuth middleware from cmd/apiserver/main.go.
// When apiKey is empty all requests pass through. Otherwise Bearer auth is
// required except for health/reflection/webhook paths.
func withBDDAuth(h http.Handler, apiKey string) http.Handler {
	if apiKey == "" {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if p == "/healthz" || p == "/readyz" ||
			strings.HasPrefix(p, "/grpc.health.") ||
			strings.HasPrefix(p, "/grpc.reflection.") ||
			p == "/api/v1/webhooks/github" {
			h.ServeHTTP(w, r)
			return
		}
		auth := r.Header.Get("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == auth {
			token = ""
		}
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(apiKey)) != 1 {
			http.Error(w, `{"error":"invalid or missing API key"}`, http.StatusUnauthorized)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// newAuthHandler creates an HTTP mux with the ProjectHandler wired up.
func newAuthHandler() http.Handler {
	k8s := fake.NewClientBuilder().WithScheme(bddScheme).Build()
	mux := http.NewServeMux()
	ph := &server.ProjectHandler{K8sClient: k8s, Namespace: "default"}
	ph.RegisterProjectHandlers(mux)
	return mux
}

// doRequest fires a single GET against handler from the given IP.
func doRequest(handler http.Handler, path, ip, token string) int {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.RemoteAddr = ip + ":1234"
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec.Code
}

// doRequests fires n sequential requests and returns the status codes.
func doRequests(handler http.Handler, n int, ip string) []int {
	codes := make([]int, n)
	for i := range codes {
		codes[i] = doRequest(handler, "/api/v1/counts", ip, "")
	}
	return codes
}

var _ = Describe("Auth Boundary", func() {
	var (
		handler    http.Handler
		baseInner  http.Handler
		apiKey     = "test-secret-key"
	)

	BeforeEach(func() {
		baseInner = newAuthHandler()
		handler = withBDDAuth(baseInner, apiKey)
	})

	Describe("Protected endpoints", func() {
		Context("When no Authorization header is provided", func() {
			It("returns 401 Unauthorized", func() {
				code := doRequest(handler, "/api/v1/projects", "10.0.0.1", "")
				Expect(code).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("When an incorrect Bearer token is provided", func() {
			It("returns 401 Unauthorized", func() {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
				req.Header.Set("Authorization", "Bearer wrong-key")
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				Expect(rec.Code).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("When the correct Bearer token is provided", func() {
			It("returns 200 OK (passes through to the underlying handler)", func() {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
				req.Header.Set("Authorization", "Bearer "+apiKey)
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				Expect(rec.Code).To(Equal(http.StatusOK))
			})
		})

		Context("When the token is passed via the ?token= query parameter", func() {
			It("returns 200 OK (WebSocket clients that cannot set headers)", func() {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/projects?token="+apiKey, nil)
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				Expect(rec.Code).To(Equal(http.StatusOK))
			})
		})
	})

	Describe("Exempt paths bypass Bearer auth", func() {
		It("allows /healthz without a token", func() {
			mux := http.NewServeMux()
			mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			protected := withBDDAuth(mux, apiKey)
			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			rec := httptest.NewRecorder()
			protected.ServeHTTP(rec, req)
			Expect(rec.Code).To(Equal(http.StatusOK))
		})

		It("allows /readyz without a token", func() {
			mux := http.NewServeMux()
			mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			protected := withBDDAuth(mux, apiKey)
			req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
			rec := httptest.NewRecorder()
			protected.ServeHTTP(rec, req)
			Expect(rec.Code).To(Equal(http.StatusOK))
		})

		It("does not intercept the webhook endpoint with the Bearer auth layer", func() {
			mux := http.NewServeMux()
			mux.HandleFunc("/api/v1/webhooks/github", func(w http.ResponseWriter, _ *http.Request) {
				// HMAC layer rejects it — this is intentional; we verify Bearer auth is not the rejector.
				http.Error(w, "hmac required", http.StatusUnauthorized)
			})
			protected := withBDDAuth(mux, apiKey)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github", nil)
			rec := httptest.NewRecorder()
			protected.ServeHTTP(rec, req)
			Expect(rec.Body.String()).NotTo(ContainSubstring("invalid or missing API key"))
		})
	})

	Describe("No API key configured", func() {
		Context("When the server is started without an API key", func() {
			It("passes all requests through without auth checks", func() {
				open := withBDDAuth(baseInner, "")
				req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
				rec := httptest.NewRecorder()
				open.ServeHTTP(rec, req)
				Expect(rec.Code).To(Equal(http.StatusOK))
			})
		})
	})
})

var _ = Describe("Rate Limiting", func() {
	var okHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	Describe("Burst exhaustion", func() {
		Context("Given a rate limiter with burst=2 and near-zero RPS", func() {
			var handler http.Handler

			BeforeEach(func() {
				cfg := server.RateLimiterConfig{
					Enabled:    true,
					RPS:        0.0001,
					Burst:      2,
					TTLMinutes: 10,
				}
				rl := server.NewRateLimiter(cfg)
				handler = server.RateLimitMiddleware(rl)(okHandler)
			})

			When("10 requests are sent from the same IP", func() {
				It("allows exactly burst=2 and throttles the rest with 429", func() {
					codes := doRequests(handler, 10, "10.1.0.1")
					got200, got429 := 0, 0
					for _, c := range codes {
						switch c {
						case http.StatusOK:
							got200++
						case http.StatusTooManyRequests:
							got429++
						}
					}
					Expect(got200).To(Equal(2), "exactly burst=2 requests should succeed")
					Expect(got429).To(Equal(8), "remaining 8 should be rate-limited")
				})
			})
		})
	})

	Describe("Per-IP isolation", func() {
		Context("Given a rate limiter with burst=2", func() {
			var handler http.Handler

			BeforeEach(func() {
				cfg := server.RateLimiterConfig{
					Enabled:    true,
					RPS:        0.0001,
					Burst:      2,
					TTLMinutes: 10,
				}
				rl := server.NewRateLimiter(cfg)
				handler = server.RateLimitMiddleware(rl)(okHandler)
			})

			When("IP-A exhausts its burst", func() {
				BeforeEach(func() {
					// Drain IP-A's tokens.
					doRequests(handler, 5, "10.2.0.1")
				})

				It("does not affect IP-B which still has a fresh bucket", func() {
					codes := doRequests(handler, 2, "10.2.0.2")
					for _, c := range codes {
						Expect(c).To(Equal(http.StatusOK), "IP-B should have its own full burst")
					}
				})
			})
		})
	})

	Describe("Disabled rate limiter", func() {
		Context("Given a rate limiter that is disabled", func() {
			var handler http.Handler

			BeforeEach(func() {
				cfg := server.RateLimiterConfig{
					Enabled:    false,
					RPS:        0.0001,
					Burst:      1,
					TTLMinutes: 10,
				}
				rl := server.NewRateLimiter(cfg)
				handler = server.RateLimitMiddleware(rl)(okHandler)
			})

			When("many requests are sent", func() {
				It("passes all of them through with 200", func() {
					codes := doRequests(handler, 20, "10.3.0.1")
					for _, c := range codes {
						Expect(c).To(Equal(http.StatusOK))
					}
				})
			})
		})
	})

	Describe("429 response headers", func() {
		Context("Given a rate limiter with burst=1", func() {
			var handler http.Handler

			BeforeEach(func() {
				cfg := server.RateLimiterConfig{
					Enabled:    true,
					RPS:        0.0001,
					Burst:      1,
					TTLMinutes: 10,
				}
				rl := server.NewRateLimiter(cfg)
				handler = server.RateLimitMiddleware(rl)(okHandler)
			})

			When("the single burst token is exhausted", func() {
				It("includes Retry-After and X-RateLimit-* headers on the 429 response", func() {
					// Exhaust the burst.
					req := httptest.NewRequest(http.MethodGet, "/api/v1/counts", nil)
					req.RemoteAddr = "10.4.0.1:9999"
					rec := httptest.NewRecorder()
					handler.ServeHTTP(rec, req)
					Expect(rec.Code).To(Equal(http.StatusOK))

					// Next request is throttled.
					req = httptest.NewRequest(http.MethodGet, "/api/v1/counts", nil)
					req.RemoteAddr = "10.4.0.1:9999"
					rec = httptest.NewRecorder()
					handler.ServeHTTP(rec, req)

					Expect(rec.Code).To(Equal(http.StatusTooManyRequests))
					Expect(rec.Header().Get("Retry-After")).NotTo(BeEmpty())
					Expect(rec.Header().Get("X-RateLimit-Limit")).NotTo(BeEmpty())
					Expect(rec.Header().Get("X-RateLimit-Remaining")).To(Equal("0"))
				})
			})
		})
	})
})

// Ensure aotv1alpha1 import is used (scheme registration via bddScheme in suite_test.go).
var _ = aotv1alpha1.AgentRunPhase("")
