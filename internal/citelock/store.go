package citelock

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// validator decides whether a candidate document is publishable.
type validator func([]byte) error

var (
	errMalformed = errors.New("store is not valid")
	errSymlink   = errors.New("store is a symlink")
	errLockHeld  = errors.New("another process holds the store lock")
)

const (
	lockAttempts = 50
	lockInterval = 100 * time.Millisecond
)

// store is one JSON document on disk, guarded by a lock directory beside it.
//
// The lock is a directory rather than a file because os.Mkdir is atomic on
// every filesystem this runs on, so two processes cannot both believe they hold
// it.
type store struct {
	path     string
	validate validator
	initial  func() ([]byte, error)
}

func (s *store) lockPath() string { return s.path + ".lock" }

// lock takes the store's lock directory and returns the release function.
func (s *store) lock() (func(), error) {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return nil, fmt.Errorf("creating store directory: %w", err)
	}
	for attempt := 0; attempt < lockAttempts; attempt++ {
		err := os.Mkdir(s.lockPath(), 0o755)
		if err == nil {
			var released bool
			return func() {
				if released {
					return
				}
				released = true
				_ = os.Remove(s.lockPath())
			}, nil
		}
		if !os.IsExist(err) {
			return nil, fmt.Errorf("taking store lock: %w", err)
		}
		time.Sleep(lockInterval)
	}
	return nil, fmt.Errorf("%w: %s", errLockHeld, s.lockPath())
}

// read returns the current document, and initializes it when it is absent or
// zero bytes.
func (s *store) read() ([]byte, error) {
	data, err := os.ReadFile(s.path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading store: %w", err)
	}
	if len(data) == 0 {
		if s.initial == nil {
			return nil, errMalformed
		}
		return s.initial()
	}
	if err := s.validate(data); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", errMalformed, s.path, err)
	}
	return data, nil
}

// publish replaces the store with data, atomically, after validating it.
func (s *store) publish(data []byte) error {
	if err := s.validate(data); err != nil {
		return fmt.Errorf("refusing to publish a malformed store: %w", err)
	}
	if info, err := os.Lstat(s.path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		target, _ := os.Readlink(s.path)
		return fmt.Errorf("%w: %s -> %s", errSymlink, s.path, target)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("creating store directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), filepath.Base(s.path)+".*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	name := tmp.Name()
	defer func() { _ = os.Remove(name) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("syncing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}
	if err := os.Rename(name, s.path); err != nil {
		return fmt.Errorf("renaming temp file into place: %w", err)
	}
	return nil
}

// jsonValidator builds a validator that asserts the document parses into T and
// satisfies check.
func jsonValidator[T any](check func(T) error) validator {
	return func(data []byte) error {
		var doc T
		if err := json.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parsing store JSON: %w", err)
		}
		if check == nil {
			return nil
		}
		return check(doc)
	}
}
