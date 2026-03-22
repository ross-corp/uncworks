package bff

import (
	"net/http"
	"strings"
)

// SPAHandler serves static files from the given filesystem.
// For paths that don't match a file, it serves index.html (SPA client-side routing).
func SPAHandler(staticFS http.FileSystem) http.Handler {
	fileServer := http.FileServer(staticFS)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Try to open the file
		if f, err := staticFS.Open(path); err == nil {
			stat, statErr := f.Stat()
			_ = f.Close()
			if statErr == nil {
				if stat.IsDir() {
					// Only serve directory if it has an index.html
					if idx, idxErr := staticFS.Open(path + "/index.html"); idxErr != nil {
						// No index.html in directory — fall through to SPA
						serveIndex(w, r, staticFS)
						return
					} else {
						_ = idx.Close()
					}
				}
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// File not found — serve index.html for SPA routing
		// But not for API paths or ConnectRPC paths
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/aot.api.v1.") {
			http.NotFound(w, r)
			return
		}
		serveIndex(w, r, staticFS)
	})
}

func serveIndex(w http.ResponseWriter, r *http.Request, staticFS http.FileSystem) {
	// Serve the root path so FileServer picks up index.html without redirecting.
	r.URL.Path = "/"
	http.FileServer(staticFS).ServeHTTP(w, r)
}
