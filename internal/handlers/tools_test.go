package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filesystem/pkg/filesystem"
	"filesystem/pkg/security"

	"github.com/mark3labs/mcp-go/mcp"
)

// helper to create handlers with a temporary directory
func newTestHandlers(t *testing.T) (*ToolHandlers, string) {
	t.Helper()
	base := t.TempDir()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	pv := security.NewPathValidator([]string{base}, logger)
	ops := filesystem.NewOperations(pv, logger)
	return NewToolHandlers(pv, ops, logger), base
}

// helper to build a call request
func newRequest(args map[string]interface{}) mcp.CallToolRequest {
	req := mcp.CallToolRequest{}
	req.Params.Name = "test_tool"
	req.Params.Arguments = args
	return req
}

func TestHandleWriteReadEditFile(t *testing.T) {
	th, base := newTestHandlers(t)
	ctx := context.Background()
	p := filepath.Join(base, "file.txt")

	// write file
	writeReq := newRequest(map[string]interface{}{"path": p, "content": "hello"})
	if _, err := th.handleWriteFile(ctx, writeReq); err != nil {
		t.Fatalf("write error: %v", err)
	}

	data, err := os.ReadFile(p)
	if err != nil || string(data) != "hello" {
		t.Fatalf("unexpected file content: %s %v", string(data), err)
	}

	// read file
	readReq := newRequest(map[string]interface{}{"path": p})
	if _, err := th.handleReadFile(ctx, readReq); err != nil {
		t.Fatalf("read error: %v", err)
	}

	// edit file
	edits := []interface{}{map[string]interface{}{"oldText": "hello", "newText": "bye"}}
	editArgs := map[string]interface{}{"path": p, "edits": edits, "dryRun": false}
	editReq := newRequest(editArgs)
	_, err = th.handleEditFile(ctx, editReq)
	if err != nil {
		t.Fatalf("edit error: %v", err)
	}

	edited, err := os.ReadFile(p)
	if err != nil || string(edited) != "bye" {
		t.Fatalf("edit failed, content: %s %v", string(edited), err)
	}
}

func TestHandleDirectoryOperations(t *testing.T) {
	th, base := newTestHandlers(t)
	ctx := context.Background()
	dir := filepath.Join(base, "sub")
	file := filepath.Join(dir, "a.txt")

	// create directory
	createReq := newRequest(map[string]interface{}{"path": dir})
	if _, err := th.handleCreateDirectory(ctx, createReq); err != nil {
		t.Fatalf("create dir error: %v", err)
	}

	if err := os.WriteFile(file, []byte("x"), 0644); err != nil {
		t.Fatalf("prep file: %v", err)
	}

	// list directory
	listReq := newRequest(map[string]interface{}{"path": dir})
	if _, err := th.handleListDirectory(ctx, listReq); err != nil {
		t.Fatalf("list error: %v", err)
	}

	// tree
	treeReq := newRequest(map[string]interface{}{"path": base})
	res, err := th.handleDirectoryTree(ctx, treeReq)
	if err != nil {
		t.Fatalf("tree error: %v", err)
	}
	if res == nil {
		t.Fatalf("nil tree result")
	}
}

func TestHandleMoveAndSearchFile(t *testing.T) {
	th, base := newTestHandlers(t)
	ctx := context.Background()
	src := filepath.Join(base, "src.txt")
	if err := os.WriteFile(src, []byte("y"), 0644); err != nil {
		t.Fatalf("prep src: %v", err)
	}
	dst := filepath.Join(base, "dst.txt")

	moveReq := newRequest(map[string]interface{}{"source": src, "destination": dst})
	if _, err := th.handleMoveFile(ctx, moveReq); err != nil {
		t.Fatalf("move error: %v", err)
	}

	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("dst not found: %v", err)
	}

	searchReq := newRequest(map[string]interface{}{"path": base, "pattern": "dst", "excludePatterns": []string{}})
	if _, err := th.handleSearchFiles(ctx, searchReq); err != nil {
		t.Fatalf("search error: %v", err)
	}
}

func TestHandleMoveFileDestinationExists(t *testing.T) {
	th, base := newTestHandlers(t)
	ctx := context.Background()
	src := filepath.Join(base, "src2.txt")
	if err := os.WriteFile(src, []byte("z"), 0644); err != nil {
		t.Fatalf("prep src: %v", err)
	}
	dst := filepath.Join(base, "dst2.txt")
	if err := os.WriteFile(dst, []byte("existing"), 0644); err != nil {
		t.Fatalf("prep dst: %v", err)
	}

	moveReq := newRequest(map[string]interface{}{"source": src, "destination": dst})
	result, err := th.handleMoveFile(ctx, moveReq)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected MCP error result for existing destination")
	}

	if _, err := os.Stat(src); err != nil {
		t.Fatalf("source missing after failed move: %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil || string(data) != "existing" {
		t.Fatalf("destination modified: %s %v", string(data), err)
	}
}

func TestHandleGetFileInfo(t *testing.T) {
	th, base := newTestHandlers(t)
	ctx := context.Background()
	p := filepath.Join(base, "info.txt")
	content := strings.Repeat("a", 10)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("prep: %v", err)
	}

	req := newRequest(map[string]interface{}{"path": p})
	res, err := th.handleGetFileInfo(ctx, req)
	if err != nil {
		t.Fatalf("info error: %v", err)
	}

	b, err := json.Marshal(res)
	if err != nil || len(b) == 0 {
		t.Fatalf("invalid result: %v", err)
	}
}

func TestHandleInvalidPath(t *testing.T) {
	th, _ := newTestHandlers(t)
	ctx := context.Background()

	// Create a temporary file outside the allowed directory
	tmpDir := os.TempDir()
	outside := filepath.Join(tmpDir, "outside.txt")
	if err := os.WriteFile(outside, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer os.Remove(outside)

	req := newRequest(map[string]interface{}{"path": outside})
	result, err := th.handleReadFile(ctx, req)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Logf("Read result: %+v", result)
		t.Fatalf("expected MCP error result for invalid path")
	}
}

func TestHandleReadMultipleFilesInvalid(t *testing.T) {
	th, base := newTestHandlers(t)
	ctx := context.Background()

	valid := filepath.Join(base, "a.txt")
	if err := os.WriteFile(valid, []byte("data"), 0644); err != nil {
		t.Fatalf("prep valid: %v", err)
	}

	invalid := filepath.Join(os.TempDir(), "outside.txt")

	req := newRequest(map[string]interface{}{"paths": []interface{}{valid, invalid}})
	res, err := th.handleReadMultipleFiles(ctx, req)
	if err != nil {
		t.Fatalf("mixed read error: %v", err)
	}
	b, _ := json.Marshal(res)
	if strings.Contains(string(b), invalid) {
		t.Fatalf("response should not include invalid path")
	}

	badReq := newRequest(map[string]interface{}{"paths": []interface{}{invalid}})
	badRes, err := th.handleReadMultipleFiles(ctx, badReq)
	if err != nil {
		t.Fatalf("invalid read error: %v", err)
	}
	bb, _ := json.Marshal(badRes)
	if !strings.Contains(string(bb), "No valid paths") {
		t.Fatalf("expected no valid paths error")
	}
}
