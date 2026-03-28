//go:build darwin

// settings.go — App settings persistence for the desktop app.
// Stored at ~/.config/uncworks/app.yaml, separate from the CLI config.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// AppSettings holds all user-configurable values for the desktop app.
type AppSettings struct {
	GitHubToken    string            `json:"githubToken"    yaml:"githubToken,omitempty"`
	Namespace      string            `json:"namespace"      yaml:"namespace,omitempty"`
	KubeContext    string            `json:"kubeContext"    yaml:"kubeContext,omitempty"`
	PortRangeStart int               `json:"portRangeStart" yaml:"portRangeStart,omitempty"`
	PortRangeEnd   int               `json:"portRangeEnd"   yaml:"portRangeEnd,omitempty"`
	// EnvOverrides allows the user to set or override environment variables
	// (EDITOR, VISUAL, PAGER, XDG_CONFIG_HOME, XDG_DATA_HOME, etc.)
	// that are inherited by child processes spawned by the app.
	EnvOverrides         map[string]string `json:"envOverrides"         yaml:"envOverrides,omitempty"`
	// LiteLLMURL is the base URL for the LiteLLM proxy. Defaults to http://litellm:4000.
	LiteLLMURL           string            `json:"litellmURL"           yaml:"litellmURL,omitempty"`
	// GitHubAuthed indicates a GitHub OAuth token is stored in Keychain.
	GitHubAuthed         bool              `json:"githubAuthed"         yaml:"githubAuthed,omitempty"`
	// UpdateChannel is "stable" or "nightly". Empty defaults to "stable".
	UpdateChannel        string            `json:"updateChannel"        yaml:"updateChannel,omitempty"`
	// AutoUpdateEnabled opts in to automatic update checks at launch.
	AutoUpdateEnabled    bool              `json:"autoUpdateEnabled"    yaml:"autoUpdateEnabled,omitempty"`
	// DefaultManageModel is the default LiteLLM model for manage-phase agents.
	DefaultManageModel   string            `json:"defaultManageModel"   yaml:"defaultManageModel,omitempty"`
	// DefaultImplementModel is the default LiteLLM model for implement-phase agents.
	DefaultImplementModel string           `json:"defaultImplementModel" yaml:"defaultImplementModel,omitempty"`
	// WizardComplete tracks whether the setup wizard has been completed.
	WizardComplete       bool              `json:"wizardComplete"       yaml:"wizardComplete,omitempty"`
}

// EnvVarInfo describes a single environment variable — its current value
// (from the process environment) and the user's override (if any).
type EnvVarInfo struct {
	Key      string `json:"key"`
	System   string `json:"system"`   // value from os.Getenv
	Override string `json:"override"` // value from EnvOverrides (empty = not set)
	Desc     string `json:"desc"`
}

func defaultSettings() AppSettings {
	return AppSettings{
		Namespace:      "uncworks",
		PortRangeStart: 50100,
		PortRangeEnd:   50120,
		LiteLLMURL:     "http://litellm:4000",
		UpdateChannel:  "stable",
	}
}

// bootstrapConfig creates the config directory and writes default settings if
// no app.yaml exists yet. It never overwrites an existing config file.
func bootstrapConfig() error {
	dir, err := appConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	path := filepath.Join(dir, "app.yaml")
	if _, err := os.Stat(path); err == nil {
		// File already exists — do not overwrite.
		return nil
	}
	return saveAppSettings(defaultSettings())
}

func appConfigDir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("home dir: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "uncworks"), nil
}

func appSettingsPath() (string, error) {
	dir, err := appConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "app.yaml"), nil
}

func loadAppSettings() (AppSettings, error) {
	s := defaultSettings()
	path, err := appSettingsPath()
	if err != nil {
		return s, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return s, fmt.Errorf("read app settings: %w", err)
	}
	if err := yaml.Unmarshal(data, &s); err != nil {
		return s, fmt.Errorf("parse app settings: %w", err)
	}
	if s.Namespace == "" {
		s.Namespace = "uncworks"
	}
	if s.PortRangeStart == 0 {
		s.PortRangeStart = 50100
	}
	if s.PortRangeEnd == 0 {
		s.PortRangeEnd = 50120
	}
	return s, nil
}

func saveAppSettings(s AppSettings) error {
	dir, err := appConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	path := filepath.Join(dir, "app.yaml")
	return os.WriteFile(path, data, 0o600)
}
