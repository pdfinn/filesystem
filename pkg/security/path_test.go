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
	if !filepath.IsAbs(p) {
		t.Fatalf("expected absolute path, got: %s", p)
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
	expectedBase := filepath.Base(target)
	actualBase := filepath.Base(p)
	if expectedBase != actualBase {
		t.Fatalf("expected target file %s got %s", expectedBase, actualBase)
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
	if len(dirs) == 0 {
		t.Fatalf("expected at least one directory")
	}

	found := false
	for _, dir := range dirs {
		if dir == base || filepath.Base(dir) == filepath.Base(base) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("base directory not found in allowed dirs: %v", dirs)
	}

	originalFirst := dirs[0]
	dirs[0] = "changed"
	newDirs := pv.GetAllowedDirectories()
	if newDirs[0] != originalFirst {
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
