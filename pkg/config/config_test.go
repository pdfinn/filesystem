package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeConfig(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestLoadSuccess(t *testing.T) {
	tmp := t.TempDir()
	cfgStr := fmt.Sprintf(`log_level: warn
allowed_directories:
  - %q
server:
  name: "srv"
  version: "v1"
  transport: "stdio"
`, tmp)
	path := writeConfig(t, tmp, cfgStr)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.LogLevel != "warn" {
		t.Fatalf("log level expected warn got %s", cfg.LogLevel)
	}
	if len(cfg.AllowedDirectories) != 1 || cfg.AllowedDirectories[0] != filepath.Clean(tmp) {
		t.Fatalf("dirs not normalized: %v", cfg.AllowedDirectories)
	}
	if cfg.Server.Name != "srv" || cfg.Server.Version != "v1" || cfg.Server.Transport != "stdio" {
		t.Fatalf("server config mismatch: %+v", cfg.Server)
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load("no_such_file.yaml"); err == nil {
		t.Fatalf("expected error for missing file")
	}
}

func TestLoadInvalidLogLevel(t *testing.T) {
	dir := t.TempDir()
	cfgStr := fmt.Sprintf(`log_level: bad
allowed_directories:
  - %q
`, dir)
	path := writeConfig(t, dir, cfgStr)
	if _, err := Load(path); err == nil {
		t.Fatalf("expected error for invalid log level")
	}
}

func TestLoadEmptyAllowed(t *testing.T) {
	dir := t.TempDir()
	cfgStr := `log_level: info
allowed_directories: []
`
	path := writeConfig(t, dir, cfgStr)
	if _, err := Load(path); err == nil {
		t.Fatalf("expected error for empty directories")
	}
}

func TestHomeExpansionAndNormalization(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("cwd: %v", err)
	}
	relDir := t.TempDir()
	relPath, err := filepath.Rel(wd, relDir)
	if err != nil {
		t.Fatalf("rel: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("home: %v", err)
	}
	homeSub, err := os.MkdirTemp(home, "cfg")
	if err != nil {
		t.Fatalf("homesub: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(homeSub) })

	cfgStr := fmt.Sprintf(`allowed_directories:
  - "~/%s"
  - %q
log_level: info
`, filepath.Base(homeSub), relPath)
	confPath := writeConfig(t, t.TempDir(), cfgStr)

	cfg, err := Load(confPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	expect := []string{filepath.Clean(homeSub), filepath.Clean(relDir)}
	if !reflect.DeepEqual(cfg.AllowedDirectories, expect) {
		t.Fatalf("expected %v got %v", expect, cfg.AllowedDirectories)
	}
}
