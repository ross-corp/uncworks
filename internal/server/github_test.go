package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSplitRepo(t *testing.T) {
	tests := []struct {
		input     string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{"owner/repo", "owner", "repo", false},
		{"my-org/my-repo", "my-org", "my-repo", false},
		{"single", "", "", true},
		{"", "", "", true},
		{"/repo", "", "", true},
		{"owner/", "", "", true},
		{"a/b/c", "a", "b/c", false}, // SplitN(2) keeps extra slashes
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			owner, repo, err := splitRepo(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("splitRepo(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr {
				if owner != tt.wantOwner || repo != tt.wantRepo {
					t.Errorf("splitRepo(%q) = (%q, %q), want (%q, %q)", tt.input, owner, repo, tt.wantOwner, tt.wantRepo)
				}
			}
		})
	}
}

func TestCheckGitHubError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		headers    http.Header
		wantNil    bool
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "200 OK returns nil",
			statusCode: http.StatusOK,
			wantNil:    true,
		},
		{
			name:       "201 Created returns nil",
			statusCode: http.StatusCreated,
			wantNil:    true,
		},
		{
			name:       "404 returns not found",
			statusCode: http.StatusNotFound,
			wantStatus: http.StatusNotFound,
			wantMsg:    "repository or file not found",
		},
		{
			name:       "401 returns auth error",
			statusCode: http.StatusUnauthorized,
			wantStatus: http.StatusUnauthorized,
			wantMsg:    "GitHub authentication failed",
		},
		{
			name:       "403 with rate limit returns 429",
			statusCode: http.StatusForbidden,
			headers:    http.Header{"X-Ratelimit-Remaining": {"0"}},
			wantStatus: http.StatusTooManyRequests,
			wantMsg:    "rate limit exceeded",
		},
		{
			name:       "403 without rate limit returns forbidden",
			statusCode: http.StatusForbidden,
			wantStatus: http.StatusForbidden,
			wantMsg:    "GitHub access forbidden",
		},
		{
			name:       "500 returns generic error",
			statusCode: http.StatusInternalServerError,
			wantStatus: http.StatusInternalServerError,
			wantMsg:    "GitHub API error (500)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Header:     tt.headers,
				Body:       http.NoBody,
			}
			if resp.Header == nil {
				resp.Header = http.Header{}
			}

			err := checkGitHubError(resp)
			if tt.wantNil {
				if err != nil {
					t.Fatalf("checkGitHubError() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("checkGitHubError() = nil, want error")
			}
			if err.statusCode != tt.wantStatus {
				t.Errorf("statusCode = %d, want %d", err.statusCode, tt.wantStatus)
			}
			if !strings.Contains(err.message, tt.wantMsg) {
				t.Errorf("message = %q, want to contain %q", err.message, tt.wantMsg)
			}
		})
	}
}

func TestHandlePull(t *testing.T) {
	// Mock GitHub API that returns file content.
	ghServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		content := base64.StdEncoding.EncodeToString([]byte("# My Spec\nDo something"))
		_ = json.NewEncoder(w).Encode(ghContentsResponse{
			Content: content,
			SHA:     "abc123",
		})
	}))
	defer ghServer.Close()

	// Create a GitHubClient pointing at the mock.
	gc := &GitHubClient{
		token:      "test-token",
		httpClient: ghServer.Client(),
	}

	mux := http.NewServeMux()
	// We need to override the GitHub API URL, but the handler hardcodes it.
	// Instead, test the handler end-to-end.
	gc.RegisterHandlers(mux)

	t.Run("pull missing params", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/specs/pull", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("pull without token returns error", func(t *testing.T) {
		gc2 := &GitHubClient{token: "", httpClient: ghServer.Client()}
		mux2 := http.NewServeMux()
		gc2.RegisterHandlers(mux2)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/specs/pull?repo=owner/repo&path=spec.cs.md", nil)
		w := httptest.NewRecorder()
		mux2.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}

		var body errorResponse
		_ = json.NewDecoder(w.Body).Decode(&body)
		if !strings.Contains(body.Error, "GITHUB_TOKEN") {
			t.Errorf("error = %q, want to mention GITHUB_TOKEN", body.Error)
		}
	})
}

func TestHandlePush(t *testing.T) {
	gc := &GitHubClient{token: "", httpClient: &http.Client{}}
	mux := http.NewServeMux()
	gc.RegisterHandlers(mux)

	t.Run("push without token returns error", func(t *testing.T) {
		body := `{"repo":"owner/repo","path":"spec.cs.md","content":"hello","message":"add spec"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/specs/push", strings.NewReader(body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})

	t.Run("push with missing fields", func(t *testing.T) {
		gc2 := &GitHubClient{token: "test-token", httpClient: &http.Client{}}
		mux2 := http.NewServeMux()
		gc2.RegisterHandlers(mux2)

		body := `{"repo":"owner/repo"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/specs/push", strings.NewReader(body))
		w := httptest.NewRecorder()
		mux2.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("push with bad repo format", func(t *testing.T) {
		gc2 := &GitHubClient{token: "test-token", httpClient: &http.Client{}}
		mux2 := http.NewServeMux()
		gc2.RegisterHandlers(mux2)

		body := `{"repo":"invalid","path":"spec.cs.md","content":"hello","message":"add spec"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/specs/push", strings.NewReader(body))
		w := httptest.NewRecorder()
		mux2.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("push with invalid JSON", func(t *testing.T) {
		gc2 := &GitHubClient{token: "test-token", httpClient: &http.Client{}}
		mux2 := http.NewServeMux()
		gc2.RegisterHandlers(mux2)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/specs/push", strings.NewReader("not json"))
		w := httptest.NewRecorder()
		mux2.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})
}

func TestDeriveNameFromPrompt(t *testing.T) {
	tests := []struct {
		prompt string
		want   string
	}{
		{"Fix the auth bug", "fix-the-auth-bug"},
		{"Add new feature for the dashboard component", "add-new-feature-for-the"},
		{"", ""},
		{"   ", ""},
		{"A!B@C#D$E", "abcde"},
		{"hello---world", "hello-world"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%q", tt.prompt), func(t *testing.T) {
			got := deriveNameFromPrompt(tt.prompt)
			if got != tt.want {
				t.Errorf("deriveNameFromPrompt(%q) = %q, want %q", tt.prompt, got, tt.want)
			}
		})
	}
}

func TestDisplayNameRegex(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"fix-auth-bug", true},
		{"add-new-feature", true},
		{"a1b2c3", true},
		{"ab", false},             // too short (min 4 chars)
		{"-invalid", false},       // starts with hyphen
		{"invalid-", false},       // ends with hyphen
		{"UPPERCASE", false},      // uppercase not allowed
		{"has spaces", false},     // spaces not allowed
		{"has_underscore", false}, // underscore not allowed
		{"a-very-long-name-that-exceeds-the-fifty-character-limit-easily", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := displayNameRegex.MatchString(tt.name)
			if got != tt.valid {
				t.Errorf("displayNameRegex.MatchString(%q) = %v, want %v", tt.name, got, tt.valid)
			}
		})
	}
}
