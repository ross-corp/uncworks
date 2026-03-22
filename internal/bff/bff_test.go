package bff

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// 8.1 SPA fallback
// ---------------------------------------------------------------------------

func TestSPAHandler_ServesStaticAsset(t *testing.T) {
	memFS := fstest.MapFS{
		"index.html":       {Data: []byte("<html>SPA</html>")},
		"assets/style.css": {Data: []byte("body{}")},
	}
	handler := SPAHandler(http.FS(memFS))

	req := httptest.NewRequest(http.MethodGet, "/assets/style.css", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "body{}")
}

func TestSPAHandler_FallbackToIndex(t *testing.T) {
	memFS := fstest.MapFS{
		"index.html":       {Data: []byte("<html>SPA</html>")},
		"assets/style.css": {Data: []byte("body{}")},
	}
	handler := SPAHandler(http.FS(memFS))

	req := httptest.NewRequest(http.MethodGet, "/run/ar-abc123", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "<html>SPA</html>")
}

func TestSPAHandler_APIPaths_NotFallback(t *testing.T) {
	memFS := fstest.MapFS{
		"index.html": {Data: []byte("<html>SPA</html>")},
	}
	handler := SPAHandler(http.FS(memFS))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---------------------------------------------------------------------------
// 8.2 API proxy
// ---------------------------------------------------------------------------

func TestProxy_ForwardsRequestToBackend(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Backend", "hello")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer backend.Close()

	proxy := NewProxy(backend.URL)
	mux := http.NewServeMux()
	proxy.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, `{"status":"ok"}`, rec.Body.String())
	assert.Equal(t, "hello", rec.Header().Get("X-Custom-Backend"))
}

func TestProxy_PreservesResponseHeaders(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Trace-Id", "trace-999")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, `{}`)
	}))
	defer backend.Close()

	proxy := NewProxy(backend.URL)
	mux := http.NewServeMux()
	proxy.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "trace-999", rec.Header().Get("X-Trace-Id"))
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}

// ---------------------------------------------------------------------------
// 8.3 Response shaping
// ---------------------------------------------------------------------------

func TestDisplaySpanName_Remapping(t *testing.T) {
	tests := []struct{ input, want string }{
		{"unc.thought", "manage.thought"},
		{"neph.write", "implement.write"},
		{"impl.bash", "implement.bash"},
		{"manage.thought", "manage.thought"},
		{"implement.read", "implement.read"},
		{"system.pipeline", "system.pipeline"},
	}
	for _, tt := range tests {
		got := displaySpanName(tt.input)
		if got != tt.want {
			t.Errorf("displaySpanName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEstimateCost_KnownModel(t *testing.T) {
	// deepseek-v3.1: $0.15/M input + $0.75/M output
	cost := EstimateCost("deepseek-v3.1", 10000, 1000)
	// (10000 * 0.15 + 1000 * 0.75) / 1_000_000 = (1500 + 750) / 1_000_000 = 0.00225
	if cost < 0.002 || cost > 0.003 {
		t.Errorf("EstimateCost = %f, want ~0.00225", cost)
	}
}

func TestEstimateCost_UnknownModel(t *testing.T) {
	cost := EstimateCost("unknown-model", 1000, 1000)
	if cost <= 0 {
		t.Error("unknown model should fall back to default pricing, got 0")
	}
}

func TestEstimateCost_ZeroTokens(t *testing.T) {
	cost := EstimateCost("deepseek-v3.1", 0, 0)
	if cost != 0 {
		t.Errorf("zero tokens should cost 0, got %f", cost)
	}
}

// ---------------------------------------------------------------------------
// 8.4 Cache
// ---------------------------------------------------------------------------

func TestCache_SetAndGet(t *testing.T) {
	c := NewCache()
	c.Set("key1", []byte("value1"), 5*time.Second)

	data, ok := c.Get("key1")
	require.True(t, ok)
	assert.Equal(t, []byte("value1"), data)
}

func TestCache_GetAfterTTLExpires(t *testing.T) {
	c := NewCache()
	c.Set("key1", []byte("value1"), 1*time.Millisecond)

	time.Sleep(5 * time.Millisecond)

	_, ok := c.Get("key1")
	assert.False(t, ok, "expected cache miss after TTL expiry")
}

func TestCache_InvalidateByPrefix(t *testing.T) {
	c := NewCache()
	c.Set("runs:1", []byte("a"), 5*time.Second)
	c.Set("runs:2", []byte("b"), 5*time.Second)
	c.Set("traces:1", []byte("c"), 5*time.Second)

	c.Invalidate("runs:")

	_, ok1 := c.Get("runs:1")
	_, ok2 := c.Get("runs:2")
	_, ok3 := c.Get("traces:1")

	assert.False(t, ok1, "runs:1 should be invalidated")
	assert.False(t, ok2, "runs:2 should be invalidated")
	assert.True(t, ok3, "traces:1 should still exist")
}

// ---------------------------------------------------------------------------
// 8.5 Rate limiter
// ---------------------------------------------------------------------------

func TestRateLimitMiddleware(t *testing.T) {
	handler := RateLimitMiddleware(100)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First 100 requests should succeed.
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	// 101st request should be rate-limited.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

// ---------------------------------------------------------------------------
// 8.6 Health check
// ---------------------------------------------------------------------------

func TestHealthz_Returns200OK(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", HealthHandler())

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	body, _ := io.ReadAll(rec.Body)
	assert.Equal(t, "ok", string(body))
}

// ---------------------------------------------------------------------------
// 8.7 Contract test — ConnectRPC path proxying
// ---------------------------------------------------------------------------

func TestProxy_ConnectRPCPathProxied(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/aot.api.v1.AOTService/") {
			w.Header().Set("Content-Type", "application/proto")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "connect-rpc-response")
			return
		}
		http.NotFound(w, r)
	}))
	defer backend.Close()

	proxy := NewProxy(backend.URL)
	mux := http.NewServeMux()
	proxy.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/aot.api.v1.AOTService/ListAgentRuns", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "connect-rpc-response", rec.Body.String())
	assert.Equal(t, "application/proto", rec.Header().Get("Content-Type"))
}

// ---------------------------------------------------------------------------
// 8.8 WebSocket proxy
// ---------------------------------------------------------------------------

func TestProxyWebSocket_Returns101(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		// Echo loop
		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			if err := conn.WriteMessage(mt, msg); err != nil {
				break
			}
		}
	}))
	defer backend.Close()

	proxy := NewProxy(backend.URL)
	frontend := httptest.NewServer(proxy.proxyHandler())
	defer frontend.Close()

	// Connect through the proxy
	wsURL := "ws" + strings.TrimPrefix(frontend.URL, "http") + "/api/v1/runs/test/exec"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	require.Equal(t, 101, resp.StatusCode)
	defer func() { _ = conn.Close() }()

	// Echo test
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte("hello")))
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, "hello", string(msg))
}

// ---------------------------------------------------------------------------
// 8.9 Middleware tests
// ---------------------------------------------------------------------------

func TestCORSMiddleware_SetsHeaders(t *testing.T) {
	handler := CORSMiddleware("http://localhost:3000")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.Equal(t, "http://localhost:3000", rr.Header().Get("Access-Control-Allow-Origin"))
	require.Contains(t, rr.Header().Get("Access-Control-Allow-Methods"), "GET")
	require.Equal(t, "true", rr.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORSMiddleware_PreflightReturns204(t *testing.T) {
	handler := CORSMiddleware("*")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // should NOT be reached
	}))
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/runs", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusNoContent, rr.Code)
}

func TestSessionMiddleware_SetsCookie(t *testing.T) {
	handler := SessionMiddleware("test-secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	cookies := rr.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "aot_session" {
			found = true
			require.True(t, c.HttpOnly, "session cookie must be HttpOnly")
			require.True(t, c.Secure, "session cookie must be Secure")
		}
	}
	require.True(t, found, "should set aot_session cookie")
}

func TestSessionMiddleware_ReusesExistingSession(t *testing.T) {
	handler := SessionMiddleware("test-secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request — get the session cookie
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	var sessionCookie *http.Cookie
	for _, c := range rr1.Result().Cookies() {
		if c.Name == "aot_session" {
			sessionCookie = c
		}
	}
	require.NotNil(t, sessionCookie, "first request should set cookie")

	// Second request — send the cookie back
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.AddCookie(sessionCookie)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	// Should not set a new cookie (session is valid)
	var newCookie *http.Cookie
	for _, c := range rr2.Result().Cookies() {
		if c.Name == "aot_session" {
			newCookie = c
		}
	}
	// No new cookie should be set since the session is known
	assert.Nil(t, newCookie, "valid session should not set a new cookie")
}

func TestRequestIDMiddleware_AddsHeader(t *testing.T) {
	handler := RequestIDMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.NotEmpty(t, rr.Header().Get("X-Request-ID"), "response should have X-Request-ID")
}

func TestRequestIDMiddleware_PreservesExisting(t *testing.T) {
	handler := RequestIDMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "existing-id-123")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	require.Equal(t, "existing-id-123", rr.Header().Get("X-Request-ID"))
}

func TestChainMiddleware_AppliesInOrder(t *testing.T) {
	// Chain CORS + RequestID, verify both take effect
	combined := Chain(
		CORSMiddleware("http://example.com"),
		RequestIDMiddleware(),
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	combined.ServeHTTP(rr, req)

	require.Equal(t, "http://example.com", rr.Header().Get("Access-Control-Allow-Origin"))
	require.NotEmpty(t, rr.Header().Get("X-Request-ID"))
}
