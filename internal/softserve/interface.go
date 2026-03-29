package softserve

// RepoManager defines the interface for managing git repositories on a soft-serve server.
// Implementations: Client (production), mock in tests.
type RepoManager interface {
	// CreateRepo creates a new repository with the given name.
	CreateRepo(name string) error
	// DeleteRepo removes the repository with the given name.
	DeleteRepo(name string) error
	// RepoExists reports whether a repository with the given name exists.
	RepoExists(name string) (bool, error)
	// CloneURL returns the SSH clone URL for the named repository.
	CloneURL(name string) string
	// ScaffoldAndPush creates initial scaffold files in a new repository and pushes them.
	ScaffoldAndPush(scaffold ScaffoldProject) error
	// ReadFile reads the contents of filePath from the named repository at HEAD.
	ReadFile(repoName, filePath string) (string, error)
	// WriteFile writes content to filePath in the named repository, commits, and pushes.
	WriteFile(repoName, filePath, content, commitMsg string) error
	// ListFiles returns all file paths tracked in the named repository at HEAD.
	ListFiles(repoName string) ([]string, error)
}

// Verify Client implements RepoManager at compile time.
var _ RepoManager = (*Client)(nil)
