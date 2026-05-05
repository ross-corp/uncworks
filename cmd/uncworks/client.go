// client.go — shared API client construction for uncworks CLI commands.
package main

import (
	"fmt"
	"net/http"
	"strings"

	apiv1connect "github.com/uncworks/aot/gen/go/api/v1/apiv1connect"
)

const defaultServerPort = 50055

// serverBaseURL resolves the server base URL from the flag override or stored config.
func serverBaseURL(serverAddr string) string {
	addr := serverAddr
	if addr == "" {
		cfg, err := loadConfig()
		if err == nil && cfg.Server.Address != "" {
			addr = cfg.Server.Address
		} else {
			addr = fmt.Sprintf("http://localhost:%d", defaultServerPort)
		}
	}
	if !strings.HasPrefix(addr, "http") {
		addr = "http://" + addr
	}
	return addr
}

// newClient returns an AOTServiceClient connected to the configured server.
// If serverAddr is non-empty it overrides the stored config.
func newClient(serverAddr string) (apiv1connect.AOTServiceClient, error) {
	return apiv1connect.NewAOTServiceClient(http.DefaultClient, serverBaseURL(serverAddr)), nil
}
