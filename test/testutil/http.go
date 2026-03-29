// test/testutil/http.go — Shared HTTP helpers for REST-API test packages.
// The regression package had two near-identical postJSON helpers (one for
// projects, one for chains).  A single generic PostJSON here replaces both.
package testutil

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// PostJSON marshals body to JSON and POSTs it to url.  The test fails
// immediately if marshalling or the HTTP call itself returns an error.
// The caller is responsible for closing resp.Body.
func PostJSON(t *testing.T, url string, body interface{}) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b)) //nolint:noctx
	require.NoError(t, err)
	return resp
}
