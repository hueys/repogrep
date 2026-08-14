package config

import (
	"os"
	"path/filepath"
	"testing"
)

// clearXDG ensures XDG_CONFIG_HOME doesn't leak in from the real
// environment and skew baseConfigDir's precedence logic.
func clearXDG(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", "")
}

func TestDefault_NoXDG(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	clearXDG(t)

	cfg := Default()

	wantDB := filepath.Join(home, ".config", "repogrep", "repogrep.db")
	if cfg.DBPath != wantDB {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, wantDB)
	}
	if cfg.PruneInactiveMonths != 12 {
		t.Errorf("PruneInactiveMonths = %d, want 12", cfg.PruneInactiveMonths)
	}
	if cfg.DefaultLimit != 50 {
		t.Errorf("DefaultLimit = %d, want 50", cfg.DefaultLimit)
	}
	if cfg.ImportConcurrency != 5 {
		t.Errorf("ImportConcurrency = %d, want 5", cfg.ImportConcurrency)
	}
}

func TestDefault_XDGConfigHomeTakesPrecedence(t *testing.T) {
	home := t.TempDir()
	xdg := filepath.Join(t.TempDir(), "xdgcfg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	cfg := Default()

	wantDB := filepath.Join(xdg, "repogrep", "repogrep.db")
	if cfg.DBPath != wantDB {
		t.Errorf("DBPath = %q, want %q (XDG_CONFIG_HOME should win over HOME)", cfg.DBPath, wantDB)
	}
}

func TestConfigDir_CreatesDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	clearXDG(t)

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}
	want := filepath.Join(home, ".config", "repogrep")
	if dir != want {
		t.Errorf("ConfigDir = %q, want %q", dir, want)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("expected ConfigDir to create %s, stat failed: %v", dir, err)
	}
	if !info.IsDir() {
		t.Errorf("%s exists but is not a directory", dir)
	}
}

func TestLoad_DefaultsWhenNoConfigFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	clearXDG(t)
	t.Setenv("REPOGREP_DB", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := Default()
	if cfg != want {
		t.Errorf("Load() with no config file = %+v, want defaults %+v", cfg, want)
	}
}

func TestLoad_YAMLOverridesMergeWithDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	clearXDG(t)
	t.Setenv("REPOGREP_DB", "")

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}
	yaml := "prune_inactive_months: 6\ndefault_limit: 25\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.PruneInactiveMonths != 6 {
		t.Errorf("PruneInactiveMonths = %d, want 6 (from config.yaml)", cfg.PruneInactiveMonths)
	}
	if cfg.DefaultLimit != 25 {
		t.Errorf("DefaultLimit = %d, want 25 (from config.yaml)", cfg.DefaultLimit)
	}
	// Fields omitted from the YAML should keep their built-in defaults,
	// not get zeroed out.
	if cfg.ImportConcurrency != 5 {
		t.Errorf("ImportConcurrency = %d, want 5 (default preserved for omitted key)", cfg.ImportConcurrency)
	}
	wantDB := filepath.Join(home, ".config", "repogrep", "repogrep.db")
	if cfg.DBPath != wantDB {
		t.Errorf("DBPath = %q, want %q (default preserved for omitted key)", cfg.DBPath, wantDB)
	}
}

func TestLoad_EnvVarOverridesYAMLDBPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	clearXDG(t)

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}
	yaml := "db_path: /from/yaml.db\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}

	// Without the env var, the YAML value should win over the built-in default.
	t.Setenv("REPOGREP_DB", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DBPath != "/from/yaml.db" {
		t.Errorf("DBPath = %q, want %q (from config.yaml)", cfg.DBPath, "/from/yaml.db")
	}

	// With the env var set, it should win over both the YAML and the default.
	t.Setenv("REPOGREP_DB", "/from/env.db")
	cfg, err = Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DBPath != "/from/env.db" {
		t.Errorf("DBPath = %q, want %q (REPOGREP_DB should win over config.yaml)", cfg.DBPath, "/from/env.db")
	}
}

func TestLoad_InvalidYAMLReturnsError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	clearXDG(t)
	t.Setenv("REPOGREP_DB", "")

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("not: [valid: yaml"), 0o600); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}

	if _, err := Load(); err == nil {
		t.Fatalf("expected Load() to error on malformed config.yaml, got nil")
	}
}
