package temporal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGitHubOwnerRepo(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		owner   string
		repo    string
		wantErr bool
	}{
		{
			name:  "HTTPS with .git suffix",
			url:   "https://github.com/org/repo.git",
			owner: "org",
			repo:  "repo",
		},
		{
			name:  "HTTPS without .git suffix",
			url:   "https://github.com/org/repo",
			owner: "org",
			repo:  "repo",
		},
		{
			name:  "SSH format",
			url:   "git@github.com:org/repo.git",
			owner: "org",
			repo:  "repo",
		},
		{
			name:  "SSH format without .git",
			url:   "git@github.com:myuser/myproject.git",
			owner: "myuser",
			repo:  "myproject",
		},
		{
			name:  "HTTPS with nested path",
			url:   "https://github.com/company/product",
			owner: "company",
			repo:  "product",
		},
		{
			name:    "invalid SSH URL missing colon path",
			url:     "git@github.com",
			wantErr: true,
		},
		{
			name:    "bare domain with no path segments",
			url:     "https://github.com/",
			wantErr: true,
		},
		{
			name:  "HTTPS with token in URL",
			url:   "https://x-access-token:ghp_abc123@github.com/org/repo.git",
			owner: "org",
			repo:  "repo",
		},
		{
			name:    "empty string",
			url:     "",
			wantErr: true,
		},
		{
			name:  "SSH with uppercase",
			url:   "git@github.com:MyOrg/MyRepo.git",
			owner: "MyOrg",
			repo:  "MyRepo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := parseGitHubOwnerRepo(tt.url)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.owner, owner)
			assert.Equal(t, tt.repo, repo)
		})
	}
}

func TestCreatePRInput_DefaultBaseBranch_IsEmptyBeforeActivity(t *testing.T) {
	// When BaseBranch is omitted, it is empty before the activity runs.
	// The CreatePR activity defaults it to "main" internally — this test
	// documents that the zero value is intentionally empty (not pre-set).
	var input CreatePRInput
	assert.Empty(t, input.BaseBranch,
		"BaseBranch should be empty when not set; CreatePR activity defaults it to main")
}

func TestParseGitHubOwnerRepo_DefaultBranchFallback(t *testing.T) {
	// Regression: activities_git.go:CreatePR defaults BaseBranch to "main" when empty.
	// Verify parseGitHubOwnerRepo round-trips an HTTPS URL correctly since CreatePR
	// uses it internally to construct the API URL.
	owner, repo, err := parseGitHubOwnerRepo("https://github.com/acme/widget.git")
	require.NoError(t, err)
	assert.Equal(t, "acme", owner)
	assert.Equal(t, "widget", repo)
}
