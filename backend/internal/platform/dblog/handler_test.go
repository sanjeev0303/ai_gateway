package dblog

import (
	"bytes"
	"errors"
	"log/slog"
	"testing"
)

func TestHandler_CopiesWarnAndErrorOnly(t *testing.T) {
	var out bytes.Buffer
	sink := &Sink{queue: make(chan entry, 10)}
	logger := sink.Wrap(slog.New(slog.NewJSONHandler(&out, &slog.HandlerOptions{Level: slog.LevelDebug})))

	logger.Info("routine")
	logger.With("request_id", "r1").WithGroup("db").Warn("slow query", "ms", 900)
	logger.Error("write failed", "error", errors.New("disk full"), "api_key", "sk-live-secret")

	if got := len(sink.queue); got != 2 {
		t.Fatalf("queued %d entries, want 2 (warn and error only)", got)
	}
	warn := <-sink.queue
	if warn.level != "WARN" || warn.attrs["request_id"] != "r1" || warn.attrs["db.ms"] == nil {
		t.Fatalf("unexpected warn entry: %+v", warn)
	}
	failed := <-sink.queue
	if failed.level != "ERROR" || failed.attrs["error"] != "disk full" {
		t.Fatalf("unexpected error entry: %+v", failed)
	}
	if failed.attrs["api_key"] != "<redacted>" {
		t.Fatalf("secret reached the sink: %v", failed.attrs["api_key"])
	}
	if !bytes.Contains(out.Bytes(), []byte("routine")) {
		t.Fatal("records no longer reach the wrapped handler")
	}
}

func TestSink_DropsWhenFullInsteadOfBlocking(t *testing.T) {
	sink := &Sink{queue: make(chan entry, 1)}
	logger := sink.Wrap(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	logger.Warn("one")
	logger.Warn("two") // must not block
	if sink.dropped.Load() != 1 {
		t.Fatalf("dropped = %d, want 1", sink.dropped.Load())
	}
}
