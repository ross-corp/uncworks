// Package bff implements the Backend-for-Frontend HTTP server that proxies
// API requests to the apiserver and serves the single-page application.
// It provides session management, CSRF protection, CORS, rate limiting, and
// a transparent WebSocket proxy.
package bff

import (
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// Proxy forwards HTTP and WebSocket requests to the apiserver.
type Proxy struct {
	apiserverURL *url.URL
	reverseProxy *httputil.ReverseProxy
	Cache        *Cache
}

// NewProxy creates a new Proxy that connects to the apiserver at the given URL.
func NewProxy(apiserverURL string) *Proxy {
	target, err := url.Parse(apiserverURL)
	if err != nil {
		slog.Error("invalid apiserver URL", "url", apiserverURL, "err", err)
		panic("invalid apiserver URL: " + err.Error())
	}

	rp := httputil.NewSingleHostReverseProxy(target)

	// Customize director to preserve the original host header.
	origDirector := rp.Director
	rp.Director = func(req *http.Request) {
		origDirector(req)
		req.Host = target.Host
	}

	return &Proxy{
		apiserverURL: target,
		reverseProxy: rp,
		Cache:        NewCache(),
	}
}

// RegisterRoutes registers all API routes on the given mux.
func (p *Proxy) RegisterRoutes(mux *http.ServeMux) {
	// Health checks (handled locally, not proxied)
	mux.HandleFunc("GET /healthz", HealthHandler())
	mux.HandleFunc("GET /readyz", HealthHandler())

	// All API routes — proxy to apiserver
	mux.HandleFunc("/api/", p.proxyHandler())

	// ConnectRPC routes — proxy to apiserver
	mux.HandleFunc("/aot.api.v1.AOTService/", p.proxyHandler())
}

func (p *Proxy) proxyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// WebSocket requests need special handling —
		// httputil.ReverseProxy does not support Upgrade: websocket.
		if isWebSocketUpgrade(r) {
			slog.Debug("bff: websocket proxy start", "path", r.URL.Path)
			p.proxyWebSocket(w, r)
			slog.Debug("bff: websocket proxy done", "path", r.URL.Path, "duration_ms", time.Since(start).Milliseconds())
			return
		}
		slog.Debug("bff: proxy request", "method", r.Method, "path", r.URL.Path)
		p.reverseProxy.ServeHTTP(w, r)
		slog.Debug("bff: proxy response", "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(start).Milliseconds())
	}
}

// isWebSocketUpgrade returns true if the request carries an Upgrade: websocket header.
func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

// proxyWebSocket performs a raw TCP-level proxy for WebSocket upgrade requests.
func (p *Proxy) proxyWebSocket(w http.ResponseWriter, r *http.Request) {
	// Determine the backend TCP address.
	backendHost := p.apiserverURL.Host
	if !strings.Contains(backendHost, ":") {
		// Add default port based on scheme.
		if p.apiserverURL.Scheme == "https" {
			backendHost += ":443"
		} else {
			backendHost += ":80"
		}
	}

	// Connect to backend.
	backendConn, err := net.DialTimeout("tcp", backendHost, 10*time.Second)
	if err != nil {
		slog.Error("bff: websocket backend unreachable", "host", backendHost, "path", r.URL.Path, "error", err)
		http.Error(w, "websocket proxy: backend unreachable", http.StatusBadGateway)
		return
	}
	defer func() { _ = backendConn.Close() }()

	// Hijack the client connection.
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "websocket proxy: hijack not supported", http.StatusInternalServerError)
		return
	}

	// Rewrite the request URL to point at the backend path (keep original path/query).
	outReq := r.Clone(r.Context())
	outReq.URL.Scheme = p.apiserverURL.Scheme
	outReq.URL.Host = p.apiserverURL.Host
	outReq.Host = p.apiserverURL.Host
	outReq.RequestURI = r.URL.RequestURI()

	// Forward the original HTTP upgrade request to the backend.
	if err := outReq.Write(backendConn); err != nil {
		http.Error(w, "websocket proxy: failed to forward request", http.StatusBadGateway)
		return
	}

	// Now hijack the client side.
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		slog.Error("websocket proxy: hijack failed", "err", err, "path", r.URL.Path)
		return
	}
	defer func() { _ = clientConn.Close() }()

	// Bidirectional copy — when either direction finishes, we're done.
	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(backendConn, clientConn)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(clientConn, backendConn)
		done <- struct{}{}
	}()
	<-done
}
