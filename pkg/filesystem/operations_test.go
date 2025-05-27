package filesystem

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filesystem/pkg/security"
)

func newOps(t *testing.T) (*Operations, string) {
	t.Helper()
	base := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pv := security.NewPathValidator([]string{base}, logger)
	ops := NewOperations(pv, logger)
	return ops, base
}

type treeEntry struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Children []treeEntry `json:"children,omitempty"`
}

func TestDirectoryTreeSimple(t *testing.T) {
	ops, base := newOps(t)
	sub := filepath.Join(base, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	file := filepath.Join(sub, "a.txt")
	if err := os.WriteFile(file, []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	jsonStr, err := ops.DirectoryTree(base)
	if err != nil {
		t.Fatalf("tree error: %v", err)
	}
	var entries []treeEntry
	if err := json.Unmarshal([]byte(jsonStr), &entries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("no entries returned")
	}
}

func TestDirectoryTreeSymlinkLoop(t *testing.T) {
	ops, base := newOps(t)
	sub := filepath.Join(base, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// create symlink to parent to form loop
	link := filepath.Join(sub, "loop")
	if err := os.Symlink(base, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	if _, err := ops.DirectoryTree(base); err != nil {
		t.Fatalf("tree with symlink failed: %v", err)
	}
}

func TestDirectoryTreeInvalidPath(t *testing.T) {
	ops, _ := newOps(t)
	outside := filepath.Join(os.TempDir(), "outside")
	if _, err := ops.DirectoryTree(outside); err == nil {
		t.Fatalf("expected error for invalid path")
	}
}

func TestDirectoryTreeDepthLimit(t *testing.T) {
	ops, base := newOps(t)
	d := base
	for i := 0; i <= maxTreeDepth; i++ {
		d = filepath.Join(d, fmt.Sprintf("d%02d", i))
		if err := os.Mkdir(d, 0755); err != nil {
			t.Fatalf("mkdir depth %d: %v", i, err)
		}
	}
	if _, err := ops.DirectoryTree(base); err == nil {
		t.Fatalf("expected depth limit error")
	}
}

func TestReadFileWithinLimit(t *testing.T) {
	ops, base := newOps(t)
	p := filepath.Join(base, "small.txt")
	content := bytes.Repeat([]byte("a"), 100)
	if err := os.WriteFile(p, content, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := ops.ReadFile(p)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if got != string(content) {
		t.Fatalf("unexpected content: %q", got)
	}
}

func TestReadFileExceedsLimit(t *testing.T) {
	ops, base := newOps(t)
	p := filepath.Join(base, "big.txt")
	data := bytes.Repeat([]byte("b"), int(maxReadSize)+1)
	if err := os.WriteFile(p, data, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := ops.ReadFile(p); err == nil {
		t.Fatalf("expected error for oversized file")
	}
}

func TestSearchFilesExcludePatterns(t *testing.T) {
	ops, base := newOps(t)

	dir1 := filepath.Join(base, "dir1")
	dir2 := filepath.Join(base, "dir2")
	excl := filepath.Join(base, "exclude")

	if err := os.MkdirAll(dir1, 0755); err != nil {
		t.Fatalf("mkdir dir1: %v", err)
	}
	if err := os.MkdirAll(dir2, 0755); err != nil {
		t.Fatalf("mkdir dir2: %v", err)
	}
	if err := os.MkdirAll(excl, 0755); err != nil {
		t.Fatalf("mkdir exclude: %v", err)
	}

	files := []string{
		filepath.Join(base, "foo.txt"),
		filepath.Join(dir1, "foo.txt"),
		filepath.Join(dir2, "foo.txt"),
		filepath.Join(excl, "foo.txt"),
	}
	for _, f := range files {
		if err := os.WriteFile(f, []byte("x"), 0644); err != nil {
			t.Fatalf("write %s: %v", f, err)
		}
	}

	res, err := ops.SearchFiles(base, "foo", []string{"exclude"})
	if err != nil {
		t.Fatalf("search error: %v", err)
	}

	got := map[string]bool{}
	for _, p := range res {
		got[filepath.Clean(p)] = true
	}

	expect := []string{
		filepath.Join(base, "foo.txt"),
		filepath.Join(dir1, "foo.txt"),
		filepath.Join(dir2, "foo.txt"),
	}
	for _, p := range expect {
		if !got[filepath.Clean(p)] {
			t.Fatalf("missing expected path %s", p)
		}
	}
	excluded := filepath.Join(excl, "foo.txt")
	if got[filepath.Clean(excluded)] {
		t.Fatalf("excluded file returned: %s", excluded)
	}
}

func TestEditFileDryRun(t *testing.T) {
	ops, base := newOps(t)
	p := filepath.Join(base, "file.txt")
	original := "hello world"
	if err := os.WriteFile(p, []byte(original), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	edits := []EditOperation{{OldText: "hello", NewText: "hi"}}
	diff, err := ops.EditFile(p, edits, true)
	if err != nil {
		t.Fatalf("edit: %v", err)
	}
	if !strings.Contains(diff, "+hi") {
		t.Fatalf("diff does not contain change: %s", diff)
	}

	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(data) != original {
		t.Fatalf("file modified on dry run")
	}
}
