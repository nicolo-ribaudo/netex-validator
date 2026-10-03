package main

import (
	"bytes"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/theoremus-urban-solutions/netex-validator/types"
)

func TestProgressBarFraction(t *testing.T) {
	var buf bytes.Buffer
	b := newProgressBar(&buf, false)
	defer b.Finish()

	b.Handle(types.ProgressEvent{Stage: types.StageDataset, FilesTotal: 2})
	b.Handle(types.ProgressEvent{Stage: types.StageFileDone, File: "a.xml"})
	b.Handle(types.ProgressEvent{Stage: types.StageRules, File: "b.xml", RulesDone: 10, RulesTotal: 20})

	b.mu.Lock()
	got := b.fraction()
	b.mu.Unlock()
	want := (1 + 0.15 + 0.85*0.5) / 2
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("fraction = %v, want %v", got, want)
	}
}

func TestProgressBarDrawAndFinish(t *testing.T) {
	var buf bytes.Buffer
	b := newProgressBar(&buf, false)

	b.Handle(types.ProgressEvent{Stage: types.StageFileDone, File: "data.xml"})
	out := buf.String()
	if !strings.Contains(out, "100%") || !strings.HasPrefix(out, clearLine) {
		t.Errorf("unexpected progress line: %q", out)
	}

	buf.Reset()
	if _, err := b.Write([]byte("warning: something\n")); err != nil {
		t.Fatal(err)
	}
	// Log output clears the line, is written, and the bar is redrawn below it
	out = buf.String()
	if !strings.HasPrefix(out, clearLine+"warning: something\n"+clearLine) {
		t.Errorf("log output not written above the bar: %q", out)
	}

	buf.Reset()
	b.Finish()
	b.Finish() // idempotent
	if buf.String() != clearLine {
		t.Errorf("Finish wrote %q, want a single clear-line", buf.String())
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("rules 10/73 · file.xml", 11); got != "rules 10/7…" {
		t.Errorf("truncate = %q", got)
	}
	if got := truncate("short", 10); got != "short" {
		t.Errorf("truncate = %q", got)
	}
}

func TestFormatElapsed(t *testing.T) {
	for d, want := range map[time.Duration]string{
		0:                                     "0.0s",
		40 * time.Millisecond:                 "0.0s",
		400 * time.Millisecond:                "0.4s",
		time.Second:                           "1.0s",
		12*time.Second + 340*time.Millisecond: "12.3s",
		65*time.Second + 200*time.Millisecond: "1m05.2s",
		10*time.Minute + 3*time.Second:        "10m03.0s",
	} {
		if got := formatElapsed(d); got != want {
			t.Errorf("formatElapsed(%v) = %q, want %q", d, got, want)
		}
	}
}

func TestProgressBarIgnoresEventsAfterFinish(t *testing.T) {
	var buf bytes.Buffer
	b := newProgressBar(&buf, false)
	b.Finish()
	b.Handle(types.ProgressEvent{Stage: types.StageFileDone, File: "late.xml"})
	if _, err := b.Write([]byte("log line\n")); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "log line\n" {
		t.Errorf("output after Finish = %q, want only the log line", got)
	}
}
