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
