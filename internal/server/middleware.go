// Package server — auth/security middleware for the AOT API server.
//
// Scope: middleware helpers that can be composed in cmd/apiserver/main.go.
// Authorization (does caller own resource) is intentionally NOT done here
// because resource ownership requires a K8s lookup; those checks belong in
// each individual handler once user identity is propagated via context.
package server

import (
	"net/http"
	"strings"
)

// RequireJSONContentType returns middleware that enforces Content-Type:
// application/json on state-changing request methods (POST, PUT, PATCH).
//
// Rationale: without this check a caller can submit a form-encoded or
// multipart body and the JSON decoder will silently produce zero-values,
// masking the bad request.  WebSocket upgrade requests (GET) and the
// GitHub webhook (which uses application/x-www-form-urlencoded / raw bytes
// with its own HMAC guard) are exempt via their path prefix.
//
// The middleware is lenient about parameters in the media-type
// ("application/json; charset=utf-8" is accepted).
func RequireJSONContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			// Webhooks carry their own HMAC authentication and may arrive
			// without a JSON Content-Type from GitHub; exempt them.
			if strings.HasPrefix(r.URL.Path, "/api/v1/webhooks/") {
				next.ServeHTTP(w, r)
				return
			}
			ct := r.Header.Get("Content-Type")
			if ct == "" {
				http.Error(w, `{"error":"Content-Type header is required"}`, http.StatusUnsupportedMediaType)
				return
			}
			// Strip parameters (e.g. "; charset=utf-8") before comparison.
			mediaType := ct
			if i := strings.Index(ct, ";"); i != -1 {
				mediaType = strings.TrimSpace(ct[:i])
			}
			if !strings.EqualFold(mediaType, "application/json") {
				http.Error(w, `{"error":"Content-Type must be application/json"}`, http.StatusUnsupportedMediaType)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// SecurityHeaders sets conservative security-related response headers on
// every reply.  These are defence-in-depth headers; they do not replace
// authentication or input validation.
//
//   - X-Content-Type-Options: nosniff   — prevents MIME-type sniffing
//   - X-Frame-Options: DENY             — prevents clickjacking via iframes
//   - Referrer-Policy: strict-origin    — limits referrer leakage
//
// Cache-Control and Content-Security-Policy are intentionally left to the
// individual handlers because streaming SSE and WebSocket endpoints need
// different values.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin")
		next.ServeHTTP(w, r)
	})
}
