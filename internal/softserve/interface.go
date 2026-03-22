package softserve

// RepoManager defines the interface for managing git repositories on a soft-serve server.
// Implementations: Client (production), mock in tests.
type RepoManager interface {
	CreateRepo(name string) error
	DeleteRepo(name string) error
	RepoExists(name string) (bool, error)
	CloneURL(name string) string
	ScaffoldAndPush(scaffold ScaffoldProject) error
	ReadFile(repoName, filePath string) (string, error)
	WriteFile(repoName, filePath, content, commitMsg string) error
	ListFiles(repoName string) ([]string, error)
}

// Verify Client implements RepoManager at compile time.
var _ RepoManager = (*Client)(nil)
