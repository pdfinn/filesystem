package main

import (
	"os"
	"path/filepath"
	"testing"

	"filesystem/pkg/config"
)

func TestValidateCommandLineDirectoriesDot(t *testing.T) {
	tmp := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("cwd: %v", err)
	}
	// change to temporary directory for the test
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(wd) })

	cfg := config.Default()
	cfg.AllowedDirectories = []string{"."}

	if err := validateCommandLineDirectories(cfg); err != nil {
		t.Fatalf("validate: %v", err)
	}

	expect, _ := filepath.Abs(".")
	if len(cfg.AllowedDirectories) != 1 || cfg.AllowedDirectories[0] != expect {
		t.Fatalf("expected %s got %v", expect, cfg.AllowedDirectories)
	}
}

func TestValidateCommandLineDirectoriesNonexistent(t *testing.T) {
	base := t.TempDir()
	missing := filepath.Join(base, "no_such")

	cfg := config.Default()
	cfg.AllowedDirectories = []string{missing}

	if err := validateCommandLineDirectories(cfg); err == nil {
		t.Fatalf("expected error for nonexistent directory")
	}
}

func TestValidateCommandLineDirectoriesFile(t *testing.T) {
	base := t.TempDir()
	file := filepath.Join(base, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg := config.Default()
	cfg.AllowedDirectories = []string{file}

	if err := validateCommandLineDirectories(cfg); err == nil {
		t.Fatalf("expected error for non-directory path")
	}
}

func TestValidateCommandLineDirectoriesEmpty(t *testing.T) {
	cfg := config.Default()
	cfg.AllowedDirectories = []string{}

	if err := validateCommandLineDirectories(cfg); err == nil {
		t.Fatalf("expected error for empty slice")
	}
}
