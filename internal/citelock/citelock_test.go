package citelock

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writePinned lays down a snapshot, its provenance, and a lockfile record that
// all agree, which is the only shape the offline gate accepts.
func writePinned(t *testing.T, dir string, rec Record, body string) Record {
	t.Helper()
	snapDir := filepath.Join(dir, snapDirName)
	if err := os.MkdirAll(snapDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	rec.Snapshot = filepath.Join(snapDirName, rec.ID+".snapshot")
	snapFile := filepath.Join(dir, rec.Snapshot)
	if err := os.WriteFile(snapFile, []byte(body), 0o644); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}
	sum := sha256.Sum256([]byte(body))
	rec.SHA256 = hex.EncodeToString(sum[:])

	prov, err := json.Marshal(provenance{URL: rec.Source, HTTPStatus: 200, SHA256: rec.SHA256})
	if err != nil {
		t.Fatalf("marshal provenance: %v", err)
	}
	if err := os.WriteFile(snapFile+".prov.json", prov, 0o644); err != nil {
		t.Fatalf("write provenance: %v", err)
	}
	writeLock(t, dir, rec)
	return rec
}

func writeLock(t *testing.T, dir string, recs ...Record) {
	t.Helper()
	raws := make([]json.RawMessage, 0, len(recs))
	for _, rec := range recs {
		encoded, err := json.Marshal(rec)
		if err != nil {
			t.Fatalf("marshal record: %v", err)
		}
		raws = append(raws, encoded)
	}
	out, err := json.MarshalIndent(lockfile{Records: raws}, "", "  ")
	if err != nil {
		t.Fatalf("marshal lockfile: %v", err)
	}
	if err := os.WriteFile(lockPath(dir), out, 0o644); err != nil {
		t.Fatalf("write lockfile: %v", err)
	}
}

func goodRecord() Record {
	return Record{
		ID:         "rfc2119-keywords",
		Source:     "https://example.com/spec",
		Accessed:   time.Now().Format("2006-01-02"),
		ClaimClass: "paper",
		Quote:      "the quote that carries the claim",
	}
}

func TestVerify_CompleteRecordPasses(t *testing.T) {
	dir := t.TempDir()
	writePinned(t, dir, goodRecord(), "preamble\nthe quote that carries the claim\ntail")

	if err := Verify(dir, nil); err != nil {
		t.Fatalf("expected the gate to pass, got %v", err)
	}
}

func TestVerify_MissingLockfileIsNotAFailure(t *testing.T) {
	if err := Verify(t.TempDir(), nil); err != nil {
		t.Fatalf("a missing lockfile is the schema's business, got %v", err)
	}
}

func TestVerify_MissingFieldNamesTheClaim(t *testing.T) {
	dir := t.TempDir()
	rec := writePinned(t, dir, goodRecord(), "the quote that carries the claim")
	rec.Quote = ""
	writeLock(t, dir, rec)

	var lines []string
	err := Verify(dir, func(f string, a ...any) { lines = append(lines, sprintf(f, a...)) })
	if !errors.Is(err, ErrGateFailed) {
		t.Fatalf("expected ErrGateFailed, got %v", err)
	}
	if !containsAll(lines, "rfc2119-keywords", "missing a required field") {
		t.Fatalf("report did not name the claim and the problem: %v", lines)
	}
}

func TestVerify_ParaphrasedQuoteIsUnanchored(t *testing.T) {
	dir := t.TempDir()
	writePinned(t, dir, goodRecord(), "the source says something else entirely")

	var lines []string
	err := Verify(dir, func(f string, a ...any) { lines = append(lines, sprintf(f, a...)) })
	if !errors.Is(err, ErrGateFailed) {
		t.Fatalf("expected ErrGateFailed, got %v", err)
	}
	if !containsAll(lines, "unanchored") {
		t.Fatalf("report did not call the record unanchored: %v", lines)
	}
}

func TestVerify_EditedSnapshotFailsIntegrity(t *testing.T) {
	dir := t.TempDir()
	rec := writePinned(t, dir, goodRecord(), "the quote that carries the claim")
	// Edit the snapshot after capture, keeping the recorded hash.
	if err := os.WriteFile(filepath.Join(dir, rec.Snapshot),
		[]byte("the quote that carries the claim, plus an edit"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	var lines []string
	err := Verify(dir, func(f string, a ...any) { lines = append(lines, sprintf(f, a...)) })
	if !errors.Is(err, ErrGateFailed) {
		t.Fatalf("expected ErrGateFailed, got %v", err)
	}
	if !containsAll(lines, "integrity") {
		t.Fatalf("report did not call out integrity: %v", lines)
	}
}

func TestVerify_HandWrittenSnapshotLacksProvenance(t *testing.T) {
	dir := t.TempDir()
	rec := writePinned(t, dir, goodRecord(), "the quote that carries the claim")
	if err := os.Remove(filepath.Join(dir, rec.Snapshot) + ".prov.json"); err != nil {
		t.Fatalf("remove provenance: %v", err)
	}

	var lines []string
	err := Verify(dir, func(f string, a ...any) { lines = append(lines, sprintf(f, a...)) })
	if !errors.Is(err, ErrGateFailed) {
		t.Fatalf("expected ErrGateFailed, got %v", err)
	}
	if !containsAll(lines, "provenance") {
		t.Fatalf("report did not mention provenance: %v", lines)
	}
}

func TestVerify_StalePricingClaimFails(t *testing.T) {
	dir := t.TempDir()
	rec := goodRecord()
	rec.ClaimClass = "pricing"
	rec.Accessed = time.Now().AddDate(0, 0, -45).Format("2006-01-02")
	writePinned(t, dir, rec, "the quote that carries the claim")

	var lines []string
	err := Verify(dir, func(f string, a ...any) { lines = append(lines, sprintf(f, a...)) })
	if !errors.Is(err, ErrGateFailed) {
		t.Fatalf("expected ErrGateFailed, got %v", err)
	}
	if !containsAll(lines, "stale") {
		t.Fatalf("report did not call the claim stale: %v", lines)
	}
}

func TestVerify_PaperClaimNeverExpires(t *testing.T) {
	dir := t.TempDir()
	rec := goodRecord()
	rec.Accessed = "1998-01-01"
	writePinned(t, dir, rec, "the quote that carries the claim")

	if err := Verify(dir, nil); err != nil {
		t.Fatalf("a paper claim does not expire, got %v", err)
	}
}

func TestVerify_SnapshotPathCannotEscapeTheLockDir(t *testing.T) {
	dir := t.TempDir()
	rec := goodRecord()
	rec.Snapshot = "../../etc/passwd"
	rec.SHA256 = strings.Repeat("a", 64)
	writeLock(t, dir, rec)

	var lines []string
	err := Verify(dir, func(f string, a ...any) { lines = append(lines, sprintf(f, a...)) })
	if !errors.Is(err, ErrGateFailed) {
		t.Fatalf("expected ErrGateFailed, got %v", err)
	}
	if !containsAll(lines, "escapes") {
		t.Fatalf("report did not reject the traversal: %v", lines)
	}
}

func TestAssertSafeURL_RejectsUnsafeTargets(t *testing.T) {
	unsafe := []string{
		"file:///etc/passwd",
		"http://example.com/page",
		"https://169.254.169.254/latest/meta-data/",
		"https://127.0.0.1/secret",
		"https://localhost/secret",
		"https://10.0.0.5/internal",
		"https://metadata.google.internal/computeMetadata/v1/",
		"https://service.internal/config",
	}
	for _, raw := range unsafe {
		if err := assertSafeURL(raw); err == nil {
			t.Errorf("expected %s to be refused", raw)
		}
	}
	if err := assertSafeURL("https://example.com/page"); err != nil {
		t.Errorf("expected a public https URL to pass, got %v", err)
	}
}

func TestCapture_FailsClosedWhenTheQuoteDoesNotAnchor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("the page says something else"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	err := captureAgainstTestServer(t, srv.URL, CaptureOptions{
		ID: "claim", Quote: "a quote that is not present", ClaimClass: "api", LockDir: dir,
	})
	if err == nil {
		t.Fatal("expected capture to fail closed")
	}
	if _, statErr := os.Stat(lockPath(dir)); statErr == nil {
		t.Fatal("capture wrote a lockfile despite failing")
	}
	if entries, _ := os.ReadDir(filepath.Join(dir, snapDirName)); len(entries) != 0 {
		t.Fatal("capture wrote a snapshot despite failing")
	}
}

func TestCapture_WritesAGateablePinOnSuccess(t *testing.T) {
	const page = "prelude\nCallers MUST retry on 429.\nepilogue"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(page))
	}))
	defer srv.Close()

	dir := t.TempDir()
	if err := captureAgainstTestServer(t, srv.URL, CaptureOptions{
		ID: "retry-on-429", Quote: "Callers MUST retry on 429.", ClaimClass: "api", LockDir: dir,
	}); err != nil {
		t.Fatalf("capture: %v", err)
	}
	if err := Verify(dir, nil); err != nil {
		t.Fatalf("a fresh capture must pass the gate it just wrote, got %v", err)
	}
}

func TestCapture_RefusesARedirect(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("the quote"))
	}))
	defer final.Close()
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))
	defer redirector.Close()

	dir := t.TempDir()
	err := captureAgainstTestServer(t, redirector.URL, CaptureOptions{
		ID: "claim", Quote: "the quote", ClaimClass: "api", LockDir: dir,
	})
	if err == nil {
		t.Fatal("expected capture to refuse the redirect")
	}
}

func TestCapture_RejectsAnIDThatWouldEscapeTheSnapshotDir(t *testing.T) {
	err := Capture(context.Background(), CaptureOptions{
		URL: "https://example.com/x", ID: "../escape", Quote: "q", ClaimClass: "api",
		LockDir: t.TempDir(),
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "invalid claim id") {
		t.Fatalf("expected the id to be rejected, got %v", err)
	}
}

func TestUpsert_ReplacesARecordWithTheSameID(t *testing.T) {
	dir := t.TempDir()
	for _, quote := range []string{"first", "second"} {
		if err := upsert(dir, Record{ID: "same", Quote: quote, Source: "https://example.com"}); err != nil {
			t.Fatalf("upsert: %v", err)
		}
	}
	parsed, err := readLock(lockPath(dir))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(parsed.Records) != 1 {
		t.Fatalf("expected one record after two upserts, got %d", len(parsed.Records))
	}
	var rec Record
	if err := json.Unmarshal(parsed.Records[0], &rec); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rec.Quote != "second" {
		t.Fatalf("expected the newer quote to win, got %q", rec.Quote)
	}
}

func TestCitedIDs(t *testing.T) {
	text := "Pricing is 0.15 [cite: openrouter-pricing] and the cap is 30 [cite: rate-limit]."
	got := CitedIDs(text)
	want := []string{"openrouter-pricing", "rate-limit"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestRecordIDs(t *testing.T) {
	dir := t.TempDir()
	writeLock(t, dir, Record{ID: "a"}, Record{ID: "b"})
	ids, err := RecordIDs(dir)
	if err != nil {
		t.Fatalf("RecordIDs: %v", err)
	}
	if len(ids) != 2 || ids[0] != "a" || ids[1] != "b" {
		t.Fatalf("got %v", ids)
	}
}

// captureAgainstTestServer runs Capture against an httptest server. That server
// listens on loopback over plain HTTP, which the production checks refuse by
// design, so the test relaxes both for its duration.
func captureAgainstTestServer(t *testing.T, rawURL string, opts CaptureOptions) error {
	t.Helper()
	t.Setenv("CITELOCK_OFFLINE", "1")

	savedURL, savedAddr := checkURL, guardAddr
	checkURL = func(string) error { return nil }
	guardAddr = func(net.IP) bool { return false }
	t.Cleanup(func() { checkURL, guardAddr = savedURL, savedAddr })

	opts.URL = rawURL
	return Capture(context.Background(), opts, nil)
}

func containsAll(lines []string, wants ...string) bool {
	joined := strings.Join(lines, "\n")
	for _, want := range wants {
		if !strings.Contains(joined, want) {
			return false
		}
	}
	return true
}

func sprintf(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}
