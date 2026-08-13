// Package citelock implements the offline gate over a change's citations.lock.
//
// An external-factual claim is one whose truth lives outside this repository
// and can change without a commit here: a price, a rate limit, an API's
// behavior, a version, a quoted upstream policy. A claim about this repository
// is not external, because a command decides that one.
//
// The gate is split in two, and the split is the point.
//
// Capture is the enforcement point. It fetches the cited URL live and refuses
// to write anything unless the author's verbatim quote is a literal substring
// of the fetched bytes. This is what rejects a real URL that does not state the
// claim.
//
// Verify is a pure function of the repository. It reads only the pinned
// snapshot, so it needs no network, returns the same verdict for the same
// inputs, and can gate a commit or a build. It reproduces capture's verdict
// rather than re-deriving it.
//
// The threat model excludes an author who bypasses capture and hand-writes both
// a snapshot and a matching provenance record. The offline gate cannot tell
// that forgery from a real capture without a network fetch.
package citelock

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	lockfileName = "citations.lock"
	snapDirName  = "citations"

	// fetchLimit caps a captured snapshot. A cited document that does not fit
	// is a sign the wrong URL was cited.
	fetchLimit = 8 << 20

	fetchTimeout = 30 * time.Second
	probeTimeout = 20 * time.Second
)

// UsageText documents the subcommands.
const UsageText = `Citation capture and the offline gate over citations.lock.

Usage:
  uncworks cite verify [<lockdir>]
  uncworks cite capture <url> --id <id> --quote <text> --class <class> [--doi <doi>] [--lockdir <dir>]
  uncworks cite recheck [<lockdir>]

Claim classes and their freshness windows:
  pricing, availability   30 days
  api                     90 days
  paper                   no expiry
  anything else          180 days

Set CITELOCK_OFFLINE=1 to skip every live check. Verify never touches the
network, with or without it.
`

// Record is one pinned citation.
type Record struct {
	ID         string `json:"id"`
	Source     string `json:"source"`
	Accessed   string `json:"accessed"`
	ClaimClass string `json:"claim_class"`
	Quote      string `json:"quote"`
	Snapshot   string `json:"snapshot"`
	SHA256     string `json:"sha256"`
	DOI        string `json:"doi,omitempty"`
}

// provenance is the sidecar capture writes beside every snapshot. Verify
// requires it, so a hand-written snapshot with no provenance fails.
type provenance struct {
	URL        string `json:"url"`
	HTTPStatus int    `json:"http_status"`
	FetchedAt  string `json:"fetched_at"`
	SHA256     string `json:"sha256"`
}

type lockfile struct {
	Records []json.RawMessage `json:"records"`
}

// The package's sentinel errors. Every returned error wraps one of these, so a
// caller can branch on the class of failure rather than on a message.
var (
	// ErrGateFailed reports that at least one record failed the offline gate.
	ErrGateFailed = errors.New("citation gate failed")
	// ErrUnsafeSource reports a source capture must never fetch.
	ErrUnsafeSource = errors.New("unsafe citation source")
	// ErrUnanchored reports that the quote is absent from the fetched bytes.
	ErrUnanchored = errors.New("quote does not anchor in the source")
	// ErrBadRecord reports a malformed record or an invalid flag.
	ErrBadRecord = errors.New("invalid citation record")
	// ErrFetch reports that the source could not be retrieved.
	ErrFetch = errors.New("could not fetch the source")
	// ErrRetracted reports a DOI Crossref marks as retracted.
	ErrRetracted = errors.New("cited DOI is retracted")
)

// Reporter receives one line per finding. The CLI writes them to stderr.
type Reporter func(format string, args ...any)

func discard(string, ...any) {}

func lockPath(dir string) string { return filepath.Join(dir, lockfileName) }

// freshnessDays is the age a claim of this class may reach before it MUST be
// captured again. Zero means the claim does not expire.
func freshnessDays(class string) int {
	switch class {
	case "pricing", "availability":
		return 30
	case "api":
		return 90
	case "paper":
		return 0
	default:
		return 180
	}
}

func sha256File(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading snapshot: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// anchored reports whether the verbatim quote appears in the snapshot. The
// match is literal, never fuzzy and never semantic, because a fuzzy match would
// accept the paraphrase this check exists to reject.
func anchored(path, quote string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("reading snapshot: %w", err)
	}
	return strings.Contains(string(data), quote), nil
}

func readLock(path string) (*lockfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading lockfile: %w", err)
	}
	var lock lockfile
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("parsing lockfile: %w", err)
	}
	return &lock, nil
}

// Verify runs the offline gate over lockdir. It performs no network I/O, so the
// same inputs always produce the same verdict.
//
// A missing lockfile is not a failure here. Requiring the file is the schema's
// job, because only the schema knows a change was supposed to carry one.
func Verify(lockdir string, report Reporter) error {
	if report == nil {
		report = discard
	}
	lock := lockPath(lockdir)
	if _, err := os.Stat(lock); err != nil {
		report("no %s in %s, nothing to verify", lockfileName, lockdir)
		return nil
	}

	parsed, err := readLock(lock)
	if err != nil {
		return fmt.Errorf("could not read %s: %w", lock, err)
	}

	failed := 0
	now := time.Now()
	for _, raw := range parsed.Records {
		if !verifyRecord(raw, lockdir, now, report) {
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("%w: %d of %d records failed in %s",
			ErrGateFailed, failed, len(parsed.Records), lock)
	}
	report("offline gate passed: %s (%d records)", lock, len(parsed.Records))
	return nil
}

func verifyRecord(raw json.RawMessage, lockdir string, now time.Time, report Reporter) bool {
	var rec Record
	if err := json.Unmarshal(raw, &rec); err != nil {
		report("[?] format: record is not an object")
		return false
	}
	id := rec.ID
	if id == "" {
		id = "?"
	}

	if rec.Source == "" || rec.Quote == "" || rec.Snapshot == "" ||
		rec.SHA256 == "" || rec.Accessed == "" || rec.ClaimClass == "" {
		report("[%s] format: missing a required field. Every record needs "+
			"id, source, accessed, claim_class, quote, snapshot, and sha256", id)
		return false
	}

	// The snapshot path comes out of a file the author edits, so confine it to
	// the lock directory. Without this, "../../etc/passwd" reads outside.
	snapFile, err := resolveInside(lockdir, rec.Snapshot)
	if err != nil {
		report("[%s] snapshot path escapes the lock directory: %s", id, rec.Snapshot)
		return false
	}
	prov := snapFile + ".prov.json"

	if _, err := os.Stat(snapFile); err != nil {
		report("[%s] snapshot missing: %s", id, snapFile)
		return false
	}
	if _, err := os.Stat(prov); err != nil {
		report("[%s] no capture provenance beside the snapshot. A hand-written "+
			"snapshot is rejected: re-run `uncworks cite capture`", id)
		return false
	}

	actual, err := sha256File(snapFile)
	if err != nil {
		report("[%s] integrity: could not hash %s", id, snapFile)
		return false
	}
	if actual != rec.SHA256 {
		report("[%s] integrity: snapshot sha256 mismatch. Recorded %s, actual %s",
			id, rec.SHA256, actual)
		return false
	}

	var sidecar provenance
	if data, err := os.ReadFile(prov); err == nil {
		_ = json.Unmarshal(data, &sidecar)
	}
	if sidecar.SHA256 != rec.SHA256 {
		report("[%s] integrity: provenance sha256 does not match the record", id)
		return false
	}

	ok, err := anchored(snapFile, rec.Quote)
	if err != nil {
		report("[%s] unanchored: could not read the snapshot: %v", id, err)
		return false
	}
	if !ok {
		report("[%s] unanchored: the verbatim quote is not in the snapshot. "+
			"Either the quote was paraphrased or the wrong page was cited", id)
		return false
	}

	maxDays := freshnessDays(rec.ClaimClass)
	if maxDays <= 0 {
		return true
	}
	accessed, err := time.ParseInLocation("2006-01-02", rec.Accessed, time.Local)
	if err != nil {
		report("[%s] freshness: unparseable accessed date %q", id, rec.Accessed)
		return false
	}
	ageDays := int(now.Sub(accessed).Hours() / 24)
	if ageDays > maxDays {
		report("[%s] stale: %s claim accessed %d days ago, limit is %d. "+
			"Run `uncworks cite recheck`", id, rec.ClaimClass, ageDays, maxDays)
		return false
	}
	return true
}

// resolveInside joins rel onto base and refuses a result outside base.
func resolveInside(base, rel string) (string, error) {
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("%w: absolute snapshot path: %s", ErrBadRecord, rel)
	}
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("resolving lock directory: %w", err)
	}
	joined := filepath.Join(absBase, rel)
	if joined != absBase && !strings.HasPrefix(joined, absBase+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: snapshot path escapes %s: %s", ErrBadRecord, base, rel)
	}
	return joined, nil
}

// assertSafeURL refuses a source the capture path MUST never fetch.
//
// Every value here comes out of a file, so a crafted record could otherwise
// point capture at a cloud metadata endpoint or an internal service and use the
// snapshot as the exfiltration channel.
func assertSafeURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("could not parse URL %q: %w", raw, err)
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("%w: refusing non-https source: %s", ErrUnsafeSource, raw)
	}
	host := parsed.Hostname()
	switch {
	case host == "":
		return fmt.Errorf("%w: could not parse a host from %s", ErrUnsafeSource, raw)
	case host == "localhost" || strings.HasSuffix(host, ".localhost"):
		return fmt.Errorf("%w: refusing loopback host: %s", ErrUnsafeSource, host)
	case host == "metadata.google.internal" || strings.HasSuffix(host, ".internal"):
		return fmt.Errorf("%w: refusing internal metadata host: %s", ErrUnsafeSource, host)
	}
	if ip := net.ParseIP(host); ip != nil {
		// A literal IP is refused whatever its range. A public one still
		// defeats the point, because an IP is not a citable, stable source.
		return fmt.Errorf("%w: refusing literal-IP host, cite a DNS name: %s", ErrUnsafeSource, host)
	}
	return nil
}

// checkURL and guardAddr are package variables so a test can point capture at
// an httptest server, which listens on loopback and therefore fails both checks
// by design. Production never reassigns them.
var (
	checkURL  = assertSafeURL
	guardAddr = unsafeAddr
)

// unsafeAddr reports whether a resolved address is one capture must not reach.
// The hostname check above cannot decide this, because DNS can resolve a public
// name to a private address.
func unsafeAddr(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()
}

// safeClient is an HTTP client that refuses redirects and refuses to connect to
// a private address, whatever DNS says.
func safeClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, fmt.Errorf("splitting address: %w", err)
			}
			ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
			if err != nil {
				return nil, fmt.Errorf("resolving %s: %w", host, err)
			}
			for _, ip := range ips {
				if guardAddr(ip) {
					return nil, fmt.Errorf("%w: refusing to connect to %s, which resolves to %s",
						ErrUnsafeSource, host, ip)
				}
			}
			conn, err := dialer.DialContext(ctx, network, addr)
			if err != nil {
				return nil, fmt.Errorf("dialing %s: %w", addr, err)
			}
			return conn, nil
		},
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			// A redirect means the cited URL is not the URL that answered, so
			// the snapshot would not match what the author cited.
			return fmt.Errorf("%w: refusing redirect to %s, cite the final URL", ErrUnsafeSource, req.URL)
		},
	}
}

// fetch retrieves the URL and returns its body and status.
func fetch(ctx context.Context, rawURL string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("building request: %w", err)
	}
	// Some hosts serve a different document, or none at all, to a client with
	// no user agent.
	req.Header.Set("User-Agent", "uncworks-citelock/1")

	resp, err := safeClient(fetchTimeout).Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("fetching %s: %w", rawURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("%w: %s returned status %d", ErrFetch, rawURL, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, fetchLimit))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading body of %s: %w", rawURL, err)
	}
	return body, resp.StatusCode, nil
}

// offline reports whether live checks are disabled.
func offline() bool { return os.Getenv("CITELOCK_OFFLINE") == "1" }

// CaptureOptions carries the flags for one capture.
type CaptureOptions struct {
	URL        string
	ID         string
	Quote      string
	ClaimClass string
	DOI        string
	LockDir    string
}

// Capture fetches the cited URL and pins it, and fails closed when the quote
// does not appear in the fetched bytes.
//
// Capture writes nothing when the quote does not anchor. A failure here means
// the claim is wrong, the quote was paraphrased, or the URL moved. Widening the
// quote until it passes defeats the check.
func Capture(ctx context.Context, opts CaptureOptions, report Reporter) error {
	if report == nil {
		report = discard
	}
	if opts.ID == "" || opts.Quote == "" || opts.ClaimClass == "" {
		return fmt.Errorf("%w: capture needs --id, --quote, and --class", ErrBadRecord)
	}
	if strings.ContainsAny(opts.ID, `/\`) || opts.ID == "." || opts.ID == ".." {
		return fmt.Errorf("%w: invalid claim id %q, because it becomes a filename", ErrBadRecord, opts.ID)
	}
	if err := checkURL(opts.URL); err != nil {
		return err
	}
	lockdir := opts.LockDir
	if lockdir == "" {
		lockdir = "."
	}

	body, status, err := fetch(ctx, opts.URL)
	if err != nil {
		return fmt.Errorf("capture: %w", err)
	}
	if !strings.Contains(string(body), opts.Quote) {
		return fmt.Errorf("%w: the quote for [%s] is not in the fetched bytes. "+
			"If the page renders its content with JavaScript, cite a stable or archived URL, "+
			"or the JSON API behind it", ErrUnanchored, opts.ID)
	}

	snapRel := filepath.Join(snapDirName, opts.ID+".snapshot")
	snapFile := filepath.Join(lockdir, snapRel)
	if err := os.MkdirAll(filepath.Dir(snapFile), 0o755); err != nil {
		return fmt.Errorf("creating snapshot directory: %w", err)
	}
	if err := os.WriteFile(snapFile, body, 0o644); err != nil {
		return fmt.Errorf("writing snapshot: %w", err)
	}
	sum, err := sha256File(snapFile)
	if err != nil {
		return err
	}

	sidecar, err := json.Marshal(provenance{
		URL:        opts.URL,
		HTTPStatus: status,
		FetchedAt:  time.Now().UTC().Format(time.RFC3339),
		SHA256:     sum,
	})
	if err != nil {
		return fmt.Errorf("encoding provenance: %w", err)
	}
	if err := os.WriteFile(snapFile+".prov.json", append(sidecar, '\n'), 0o644); err != nil {
		return fmt.Errorf("writing provenance: %w", err)
	}

	if err := liveChecks(ctx, opts.URL, opts.DOI, report); err != nil {
		return fmt.Errorf("capture: live check failed for [%s]: %w", opts.ID, err)
	}

	if err := upsert(lockdir, Record{
		ID:         opts.ID,
		Source:     opts.URL,
		Accessed:   time.Now().Format("2006-01-02"),
		ClaimClass: opts.ClaimClass,
		Quote:      opts.Quote,
		Snapshot:   snapRel,
		SHA256:     sum,
		DOI:        opts.DOI,
	}); err != nil {
		return err
	}
	report("captured [%s] into %s (sha256 %s)", opts.ID, snapRel, sum[:12])
	return nil
}

// liveChecks confirms the source still resolves and, for a DOI, is not
// retracted. These run at capture time only. They depend on live network state,
// so putting them in the offline gate would make a build's result a function of
// the weather.
func liveChecks(ctx context.Context, source, doi string, report Reporter) error {
	if offline() {
		report("CITELOCK_OFFLINE=1, skipping the live checks")
		return nil
	}
	if doi == "" {
		return nil
	}
	if strings.ContainsAny(doi, " \t\n\"'`;&|$") {
		return fmt.Errorf("%w: refusing DOI with unexpected characters: %q", ErrBadRecord, doi)
	}

	probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	body, _, err := fetch(probeCtx, "https://api.crossref.org/works/"+url.PathEscape(doi))
	if err != nil {
		// A transient Crossref failure MUST NOT block a capture whose quote
		// already anchored, so this degrades to a warning.
		report("live: could not reach Crossref for DOI %s, skipping the retraction check: %v", doi, err)
		return nil
	}
	var parsed struct {
		Message struct {
			UpdateTo []struct {
				Type string `json:"type"`
			} `json:"update-to"`
		} `json:"message"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		report("live: Crossref returned an unparseable body for DOI %s", doi)
		return nil
	}
	for _, update := range parsed.Message.UpdateTo {
		if update.Type == "retraction" {
			return fmt.Errorf("%w: %s", ErrRetracted, doi)
		}
	}
	return nil
}

// Recheck re-runs the live checks over every pinned record. It is an authoring
// action, never a gate: a source that dies after capture is found here, on the
// next run, and not by a build that used to pass.
func Recheck(ctx context.Context, lockdir string, report Reporter) error {
	if report == nil {
		report = discard
	}
	lock := lockPath(lockdir)
	if _, err := os.Stat(lock); err != nil {
		report("no %s in %s, nothing to recheck", lockfileName, lockdir)
		return nil
	}
	parsed, err := readLock(lock)
	if err != nil {
		return fmt.Errorf("could not read %s: %w", lock, err)
	}

	failed := 0
	for _, raw := range parsed.Records {
		var rec Record
		if err := json.Unmarshal(raw, &rec); err != nil {
			report("[?] recheck: record is not an object")
			failed++
			continue
		}
		if err := checkURL(rec.Source); err != nil {
			report("[%s] recheck: %v", rec.ID, err)
			failed++
			continue
		}
		if offline() {
			report("CITELOCK_OFFLINE=1, skipping the live checks")
			return nil
		}
		if _, _, err := fetch(ctx, rec.Source); err != nil {
			report("[%s] recheck: source no longer resolves: %v", rec.ID, err)
			failed++
			continue
		}
		if err := liveChecks(ctx, rec.Source, rec.DOI, report); err != nil {
			report("[%s] recheck: %v", rec.ID, err)
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("%w: %d records no longer check out in %s", ErrGateFailed, failed, lock)
	}
	report("recheck passed: %s", lock)
	return nil
}

// upsert replaces any record with the same id and appends the new one.
func upsert(lockdir string, rec Record) error {
	s := &store{
		path: lockPath(lockdir),
		validate: jsonValidator(func(doc struct {
			Records *[]json.RawMessage `json:"records"`
		}) error {
			if doc.Records == nil {
				return fmt.Errorf("%w: lockfile has no records array", ErrBadRecord)
			}
			return nil
		}),
		initial: func() ([]byte, error) {
			return []byte(`{"records":[]}` + "\n"), nil
		},
	}

	release, err := s.lock()
	if err != nil {
		return err
	}
	defer release()

	data, err := s.read()
	if err != nil {
		return err
	}
	var lock lockfile
	if err := json.Unmarshal(data, &lock); err != nil {
		return fmt.Errorf("parsing lockfile: %w", err)
	}

	kept := make([]json.RawMessage, 0, len(lock.Records)+1)
	for _, raw := range lock.Records {
		var cur struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(raw, &cur) == nil && cur.ID == rec.ID {
			continue
		}
		kept = append(kept, raw)
	}
	encoded, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("encoding record: %w", err)
	}
	kept = append(kept, encoded)
	lock.Records = kept

	out, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding lockfile: %w", err)
	}
	return s.publish(append(out, '\n'))
}

// CitedIDs returns every `[cite: <id>]` reference in the given text.
func CitedIDs(text string) []string {
	const open = "[cite: "
	var ids []string
	for i := 0; ; {
		start := strings.Index(text[i:], open)
		if start < 0 {
			return ids
		}
		start += i + len(open)
		end := strings.Index(text[start:], "]")
		if end < 0 {
			return ids
		}
		id := strings.TrimSpace(text[start : start+end])
		if id != "" {
			ids = append(ids, id)
		}
		i = start + end
	}
}

// RecordIDs returns the id of every record in lockdir's lockfile.
func RecordIDs(lockdir string) ([]string, error) {
	lock := lockPath(lockdir)
	if _, err := os.Stat(lock); err != nil {
		return nil, nil
	}
	parsed, err := readLock(lock)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(parsed.Records))
	for _, raw := range parsed.Records {
		var rec struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(raw, &rec) == nil && rec.ID != "" {
			ids = append(ids, rec.ID)
		}
	}
	return ids, nil
}
