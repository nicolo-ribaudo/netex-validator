package logging

import (
	"context"
	"io"
	"log/slog"
	"sync"
)

// plainHandler is a slog.Handler that writes only the log message, one per
// line. Warnings and errors are prefixed with their level; attributes are dropped.
type plainHandler struct {
	mu    *sync.Mutex
	out   io.Writer
	level slog.Leveler
}

func newPlainHandler(out io.Writer, level slog.Leveler) *plainHandler {
	return &plainHandler{mu: &sync.Mutex{}, out: out, level: level}
}

func (h *plainHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *plainHandler) Handle(_ context.Context, r slog.Record) error {
	line := r.Message
	switch {
	case r.Level >= slog.LevelError:
		line = "error: " + line
	case r.Level >= slog.LevelWarn:
		line = "warning: " + line
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.out, line+"\n")
	return err
}

func (h *plainHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }

func (h *plainHandler) WithGroup(_ string) slog.Handler { return h }
