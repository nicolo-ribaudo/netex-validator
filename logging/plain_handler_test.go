package logging

import (
	"bytes"
	"testing"
)

func TestPlainFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LoggerConfig{Level: LevelInfo, Format: "plain", Output: &buf})

	logger.WithFile("a.xml").Debug("hidden")
	logger.WithFile("a.xml").Info("Starting validation", "codespace", "IT")
	logger.Warn("Something odd")
	logger.Error("It broke", "error", "boom")

	want := "Starting validation\nwarning: Something odd\nerror: It broke\n"
	if got := buf.String(); got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

func TestDefaultFormatIsPlain(t *testing.T) {
	var buf bytes.Buffer
	NewLogger(LoggerConfig{Output: &buf}).Info("hello", "key", "value")
	if got := buf.String(); got != "hello\n" {
		t.Errorf("output = %q, want %q", got, "hello\n")
	}
}
