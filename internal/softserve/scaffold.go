package softserve

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed all:scaffold
var scaffoldFS embed.FS

// writeScaffoldFiles writes the embedded scaffold files (e.g. .pi/ openspec skills)
// into the given directory, preserving relative paths from the embed root.
func writeScaffoldFiles(dir string) error {
	return fs.WalkDir(scaffoldFS, "scaffold", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Strip the "scaffold/" prefix to get the relative path within the project.
		rel, err := filepath.Rel("scaffold", path)
		if err != nil {
			return err
		}
		dest := filepath.Join(dir, rel)
		if d.IsDir() {
			return os.MkdirAll(dest, 0755)
		}
		data, err := scaffoldFS.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		return os.WriteFile(dest, data, 0644)
	})
}
