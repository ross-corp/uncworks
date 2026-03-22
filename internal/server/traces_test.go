package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadSpansFile_Deduplicates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spans.jsonl")

	// Write a JSONL file with duplicate span IDs (open + close).
	// The second entry for "span-1" has an endTime — it should win.
	content := `{"id":"span-1","name":"stage.plan","type":"stage","startTime":"2025-01-01T00:00:00Z","endTime":"","status":"unset"}
{"id":"span-2","name":"tool.bash","type":"tool","startTime":"2025-01-01T00:00:01Z","endTime":"2025-01-01T00:00:02Z","status":"ok"}
{"id":"span-1","name":"stage.plan","type":"stage","startTime":"2025-01-01T00:00:00Z","endTime":"2025-01-01T00:00:05Z","status":"ok"}
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	spans, err := readSpansFile(path)
	require.NoError(t, err)

	// Should deduplicate: 2 unique span IDs
	require.Len(t, spans, 2, "expected 2 spans after deduplication")

	// The first span should be the later (closed) version of span-1
	assert.Equal(t, "span-1", spans[0].ID)
	assert.Equal(t, "2025-01-01T00:00:05Z", spans[0].EndTime, "later version should have endTime set")
	assert.Equal(t, "ok", spans[0].Status, "later version should have status=ok")

	// Second span should be span-2
	assert.Equal(t, "span-2", spans[1].ID)
}

func TestReadSpansFile_PreservesOrder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spans.jsonl")

	content := `{"id":"A","name":"first","type":"tool","startTime":"2025-01-01T00:00:00Z","endTime":"2025-01-01T00:00:01Z"}
{"id":"B","name":"second","type":"tool","startTime":"2025-01-01T00:00:01Z","endTime":"2025-01-01T00:00:02Z"}
{"id":"C","name":"third","type":"tool","startTime":"2025-01-01T00:00:02Z","endTime":"2025-01-01T00:00:03Z"}
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	spans, err := readSpansFile(path)
	require.NoError(t, err)
	require.Len(t, spans, 3)

	assert.Equal(t, "A", spans[0].ID)
	assert.Equal(t, "B", spans[1].ID)
	assert.Equal(t, "C", spans[2].ID)
}

func TestReadSpansFile_SkipsMalformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spans.jsonl")

	content := `{"id":"good-1","name":"first","type":"tool","startTime":"2025-01-01T00:00:00Z","endTime":"2025-01-01T00:00:01Z"}
this is not valid JSON
{"id":"good-2","name":"third","type":"tool","startTime":"2025-01-01T00:00:02Z","endTime":"2025-01-01T00:00:03Z"}
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	spans, err := readSpansFile(path)
	require.NoError(t, err)
	require.Len(t, spans, 2, "malformed line should be skipped")

	assert.Equal(t, "good-1", spans[0].ID)
	assert.Equal(t, "good-2", spans[1].ID)
}

func TestReadSpansFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spans.jsonl")

	require.NoError(t, os.WriteFile(path, []byte(""), 0o644))

	spans, err := readSpansFile(path)
	require.NoError(t, err)
	assert.Empty(t, spans)
}

func TestReadSpansFile_FileNotFound(t *testing.T) {
	_, err := readSpansFile("/nonexistent/path/spans.jsonl")
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestReadSpansFile_SkipsBlankLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spans.jsonl")

	content := `{"id":"span-1","name":"first","type":"tool","startTime":"2025-01-01T00:00:00Z","endTime":"2025-01-01T00:00:01Z"}

{"id":"span-2","name":"second","type":"tool","startTime":"2025-01-01T00:00:02Z","endTime":"2025-01-01T00:00:03Z"}
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	spans, err := readSpansFile(path)
	require.NoError(t, err)
	require.Len(t, spans, 2, "blank lines should be skipped")
}

func TestReadSpansFile_PreservesDiffData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spans.jsonl")

	content := `{"id":"span-1","name":"tool.write","type":"tool","startTime":"2025-01-01T00:00:00Z","endTime":"2025-01-01T00:00:01Z","hasDiff":true,"diff":{"files":[{"path":"main.go","patch":"+ fmt.Println()"}]}}
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	spans, err := readSpansFile(path)
	require.NoError(t, err)
	require.Len(t, spans, 1)

	assert.True(t, spans[0].HasDiff)
	require.NotNil(t, spans[0].Diff)
	require.Len(t, spans[0].Diff.Files, 1)
	assert.Equal(t, "main.go", spans[0].Diff.Files[0].Path)
}
