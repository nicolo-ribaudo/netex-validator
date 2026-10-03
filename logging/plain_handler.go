package logging

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"sync"
)

// plainHandler is a slog.Handler that writes only the log message, one per
// line. Warnings and errors are prefixed with their level and followed by
// their attributes (e.g. the file and error), which are needed to act on them.
type plainHandler struct {
	mu    *sync.Mutex
	out   io.Writer
	level slog.Leveler
	attrs []slog.Attr
}

func newPlainHandler(out io.Writer, level slog.Leveler) *plainHandler {
	return &plainHandler{mu: &sync.Mutex{}, out: out, level: level}
}

func (h *plainHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *plainHandler) Handle(_ context.Context, r slog.Record) error {
	line := r.Message
	if r.Level >= slog.LevelWarn {
		// Later attributes override earlier ones with the same key (loggers
		// often add e.g. "file" both via With() and in the call itself)
		var keys []string
		values := make(map[string]string)
		addAttr := func(a slog.Attr) bool {
			if a.Key == "component" || a.Key == "timestamp" {
				return true
			}
			if _, seen := values[a.Key]; !seen {
				keys = append(keys, a.Key)
			}
			values[a.Key] = a.Value.String()
			return true
		}
		for _, a := range h.attrs {
			addAttr(a)
		}
		r.Attrs(addAttr)
		details := make([]string, len(keys))
		for i, k := range keys {
			details[i] = k + "=" + values[k]
		}
		if len(details) > 0 {
			line += " (" + strings.Join(details, " ") + ")"
		}
		if r.Level >= slog.LevelError {
			line = "error: " + line
		} else {
			line = "warning: " + line
		}
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.out, line+"\n")
	return err
}

func (h *plainHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = append(append([]slog.Attr(nil), h.attrs...), attrs...)
	return &clone
}

func (h *plainHandler) WithGroup(_ string) slog.Handler { return h }
