package server

import (
    "bytes"
    "context"
    "io"
    "log/slog"
    "testing"

    "filesystem/pkg/config"
)

func TestNewNilParameters(t *testing.T) {
    logger := slog.New(slog.NewTextHandler(io.Discard, nil))
    cfg := config.Default()

    if _, err := New(nil, logger); err == nil {
        t.Fatalf("expected error for nil config")
    }
    if _, err := New(cfg, nil); err == nil {
        t.Fatalf("expected error for nil logger")
    }
}

func TestStartNilContext(t *testing.T) {
    srv := &Server{}
    if err := srv.Start(nil); err == nil {
        t.Fatalf("expected error for nil context")
    }
}

func TestShutdownLogsAndReturnsNil(t *testing.T) {
    var buf bytes.Buffer
    logger := slog.New(slog.NewTextHandler(&buf, nil))
    srv := &Server{logger: logger}

    ctx := context.Background()
    if err := srv.Shutdown(ctx); err != nil {
        t.Fatalf("shutdown error: %v", err)
    }
    logs := buf.String()
    if !bytes.Contains([]byte(logs), []byte("Shutting down MCP server")) || !bytes.Contains([]byte(logs), []byte("MCP server shutdown complete")) {
        t.Fatalf("expected shutdown messages in logs; got: %s", logs)
    }
}

