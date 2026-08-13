package gate

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// GitRepo reads a real repository through the git binary.
type GitRepo struct {
	// Root is the working tree to run git in.
	Root string
}

// ChangedPaths returns the paths that differ between two refs.
//
// It uses the merge base rather than a two-dot diff, so a base branch that has
// moved on does not make every commit since the fork look like part of this
// pull request.
func (g GitRepo) ChangedPaths(ctx context.Context, base, head string) ([]string, error) {
	out, err := g.run(ctx, "diff", "--name-only", base+"..."+head)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, line := range strings.Split(out, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			paths = append(paths, line)
		}
	}
	return paths, nil
}

// Exists reports whether a path is present at a ref.
func (g GitRepo) Exists(ctx context.Context, ref, path string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "cat-file", "-e", ref+":"+path)
	cmd.Dir = g.Root
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if ok := asExitError(err, &exitErr); ok {
			// git exits non-zero for an absent path, which is an answer rather
			// than a failure.
			return false, nil
		}
		return false, fmt.Errorf("running git cat-file: %w", err)
	}
	return true, nil
}

func (g GitRepo) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.Root
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("running git %s: %w: %s",
			strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func asExitError(err error, target **exec.ExitError) bool {
	exitErr, ok := err.(*exec.ExitError)
	if ok {
		*target = exitErr
	}
	return ok
}
