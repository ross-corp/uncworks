// config.go — XDG-aware config file management for uncworks CLI.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the user-level config stored at ~/.config/uncworks/config.yaml.
type Config struct {
	Server struct {
		// Address is the gRPC server address (e.g. "grpc.example.com:50055").
		// Empty means use local port-forward.
		Address string `yaml:"address,omitempty"`
	} `yaml:"server,omitempty"`
	// WebURL is the base URL of the UNCWORKS web dashboard (e.g. "http://192.168.1.10:30080").
	// Used by 'uncworks runs ui' to open run detail pages.
	WebURL string `yaml:"web_url,omitempty"`
	// DefaultModelTier is the LLM model tier used when --model-tier is not specified.
	DefaultModelTier string `yaml:"default_model_tier,omitempty"`
}

// configDir returns the XDG config directory for uncworks.
// Uses $XDG_CONFIG_HOME if set, otherwise ~/.config.
func configDir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("find home directory: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "uncworks"), nil
}

// configPath returns the full path to config.yaml.
func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// loadConfig reads the config file, returning an empty Config if it doesn't exist.
func loadConfig() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

// saveConfig writes the config to disk, creating the directory if needed.
func saveConfig(cfg Config) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// pidFilePath returns the path to the port-forward PID file.
func pidFilePath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "port-forward.pid"), nil
}
