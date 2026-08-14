// Package config provides repogrep's minimal configuration: built-in
// defaults, overridable by an optional YAML file, overridable by env vars,
// overridable by CLI flags. The tool must work with zero config file
// present.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds repogrep's user-tunable defaults.
type Config struct {
	// DBPath is the path to the SQLite database file.
	DBPath string `yaml:"db_path"`

	// PruneInactiveMonths is the default threshold for "no activity in N
	// months" used by `prune` when --months isn't passed.
	PruneInactiveMonths int `yaml:"prune_inactive_months"`

	// DefaultLimit is the default row limit for `list`/`search` when
	// --limit isn't passed.
	DefaultLimit int `yaml:"default_limit"`

	// ImportConcurrency is the default worker pool size for per-record
	// fetches (e.g. READMEs) during import.
	ImportConcurrency int `yaml:"import_concurrency"`
}

// Default returns repogrep's built-in defaults.
func Default() Config {
	dir, _ := baseConfigDir()
	return Config{
		DBPath:              filepath.Join(dir, "repogrep.db"),
		PruneInactiveMonths: 12,
		DefaultLimit:        50,
		ImportConcurrency:   5,
	}
}

// baseConfigDir returns $XDG_CONFIG_HOME/repogrep if XDG_CONFIG_HOME is
// set, else ~/.config/repogrep, without creating it.
func baseConfigDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "repogrep"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "repogrep"), nil
}

// ConfigDir returns repogrep's config directory (see baseConfigDir),
// creating it (0700) if it doesn't exist.
func ConfigDir() (string, error) {
	dir, err := baseConfigDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	return dir, nil
}

// Load returns Config with defaults applied, then overridden by
// <config dir>/config.yaml if present, then by known env vars. Flags are
// applied on top of this by the CLI layer itself (highest precedence).
func Load() (Config, error) {
	cfg := Default()

	dir, err := ConfigDir()
	if err != nil {
		// Non-fatal: fall back to defaults (e.g. HOME unset in a test
		// environment). DB path will just be relative/empty; caller's
		// --db flag can still override it.
		dir = ""
	}

	if dir != "" {
		path := filepath.Join(dir, "config.yaml")
		if data, err := os.ReadFile(path); err == nil {
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return cfg, fmt.Errorf("parse %s: %w", path, err)
			}
		} else if !os.IsNotExist(err) {
			return cfg, fmt.Errorf("read %s: %w", path, err)
		}
	}

	if v := os.Getenv("REPOGREP_DB"); v != "" {
		cfg.DBPath = v
	}

	return cfg, nil
}
