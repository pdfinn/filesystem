package filesystem

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
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
