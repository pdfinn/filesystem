package security

import (
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func newValidator(t *testing.T) (*PathValidator, string) {
	t.Helper()
	base := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pv := NewPathValidator([]string{base}, logger)
	return pv, base
}

func TestValidatePathWithinAllowed(t *testing.T) {
	pv, base := newValidator(t)
	file := filepath.Join(base, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0644); err != nil {
		t.Fatalf("prep file: %v", err)
	}

	p, err := pv.ValidatePath(file)
	if err != nil {
		t.Fatalf("validate error: %v", err)
	}
	if p != file {
		t.Fatalf("unexpected path: %s", p)
	}
}

func TestValidatePathOutsideAllowed(t *testing.T) {
	pv, _ := newValidator(t)
	outside := filepath.Join(os.TempDir(), "outside.txt")
	if _, err := pv.ValidatePath(outside); err == nil {
		t.Fatalf("expected error for outside path")
	}
}

func TestValidateSymlinkTarget(t *testing.T) {
	pv, base := newValidator(t)
	target := filepath.Join(base, "target.txt")
	if err := os.WriteFile(target, []byte("x"), 0644); err != nil {
		t.Fatalf("prep target: %v", err)
	}
	link := filepath.Join(base, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	p, err := pv.ValidatePath(link)
	if err != nil {
		t.Fatalf("validate symlink: %v", err)
	}
	if p != target {
		t.Fatalf("expected real path %s got %s", target, p)
	}
}

func TestValidateSymlinkOutside(t *testing.T) {
	pv, base := newValidator(t)
	outsideDir := t.TempDir()
	target := filepath.Join(outsideDir, "target.txt")
	if err := os.WriteFile(target, []byte("x"), 0644); err != nil {
		t.Fatalf("prep target: %v", err)
	}
	link := filepath.Join(base, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	if _, err := pv.ValidatePath(link); err == nil {
		t.Fatalf("expected error for outside symlink")
	}
}

func TestExpandHomePath(t *testing.T) {
	home, _ := os.UserHomeDir()
	if got := ExpandHomePath("~"); got != home {
		t.Fatalf("expected %s got %s", home, got)
	}
	sub := ExpandHomePath("~/sub")
	if sub != filepath.Join(home, "sub") {
		t.Fatalf("expected joined path")
	}
}

func TestGetAllowedDirectories(t *testing.T) {
	pv, base := newValidator(t)
	dirs := pv.GetAllowedDirectories()
	if len(dirs) != 1 || dirs[0] != base {
		t.Fatalf("unexpected dirs: %v", dirs)
	}
	// ensure modification doesn't affect internal slice
	dirs[0] = "changed"
	if pv.allowedDirectories[0] != base {
		t.Fatalf("internal slice modified")
	}
}

func TestIsPathUnderDirectoryRelativeUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-specific test")
	}
	pv := &PathValidator{}
	if !pv.isPathUnderDirectory("a/b/c", "a/b") {
		t.Fatalf("expected true for relative unix path")
	}
	if pv.isPathUnderDirectory("../outside", "a/b") {
		t.Fatalf("expected false for path outside")
	}
}

func TestIsPathUnderDirectoryRelativeWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific test")
	}
	pv := &PathValidator{}
	if !pv.isPathUnderDirectory(`a\b\c`, `a\b`) {
		t.Fatalf("expected true for windows path")
	}
	if pv.isPathUnderDirectory(`..\outside`, `a\b`) {
		t.Fatalf("expected false for outside path")
	}
}
