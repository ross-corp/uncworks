package bff

import "net/http"

// HealthHandler returns an HTTP handler that responds with 200 "ok".
// Used for both /healthz (liveness) and /readyz (readiness) probes.
func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}
