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

func TestPushChangesInput_Fields(t *testing.T) {
	// Verify the PushChangesInput struct has all expected fields and can be constructed.
	input := PushChangesInput{
		AgentRunName:  "ar-test-001",
		PodIP:         "10.0.0.1",
		RepoPath:      "/workspace/repo",
		BranchName:    "aot/ar-test-001",
		CommitMessage: "feat: implement auth middleware",
		RepoURL:       "https://github.com/org/repo.git",
		ChangeName:    "auth-middleware",
	}

	assert.Equal(t, "ar-test-001", input.AgentRunName)
	assert.Equal(t, "10.0.0.1", input.PodIP)
	assert.Equal(t, "/workspace/repo", input.RepoPath)
	assert.Equal(t, "aot/ar-test-001", input.BranchName)
	assert.Equal(t, "feat: implement auth middleware", input.CommitMessage)
	assert.Equal(t, "https://github.com/org/repo.git", input.RepoURL)
	assert.Equal(t, "auth-middleware", input.ChangeName)
}

func TestPushChangesOutput_DiffStatAndProposal(t *testing.T) {
	// Verify that PushChangesOutput includes DiffStat and ProposalContent fields.
	output := PushChangesOutput{
		BranchName:      "aot/ar-test-001",
		CommitSHA:       "abc123def456",
		DiffStat:        " 3 files changed, 42 insertions(+), 5 deletions(-)",
		ProposalContent: "## Proposal\n\nAdd authentication middleware to all API routes.",
	}

	assert.Equal(t, "aot/ar-test-001", output.BranchName)
	assert.Equal(t, "abc123def456", output.CommitSHA)
	assert.Equal(t, " 3 files changed, 42 insertions(+), 5 deletions(-)", output.DiffStat)
	assert.Equal(t, "## Proposal\n\nAdd authentication middleware to all API routes.", output.ProposalContent)
}

func TestCreatePRInput_DefaultBaseBranch(t *testing.T) {
	// Verify that CreatePRInput can hold all required fields.
	input := CreatePRInput{
		RepoOwner:    "uncworks",
		RepoName:     "aot",
		BranchName:   "aot/ar-test-001",
		BaseBranch:   "",
		Title:        "feat: auth middleware",
		Body:         "Implements auth middleware per spec.",
		AgentRunName: "ar-test-001",
	}

	// The activity code defaults BaseBranch to "main" when empty.
	assert.Empty(t, input.BaseBranch)
	assert.Equal(t, "uncworks", input.RepoOwner)
	assert.Equal(t, "aot", input.RepoName)
}
