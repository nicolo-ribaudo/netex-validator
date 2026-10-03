package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/theoremus-urban-solutions/netex-validator/types"
)

const (
	progressBarWidth = 20
	progressInterval = 80 * time.Millisecond
	clearLine        = "\r\033[K"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// progressBar draws a single, continuously updated progress line on a
// terminal. It also implements io.Writer so that log output can be written
// through it without garbling the line.
type progressBar struct {
	mu         sync.Mutex
	out        io.Writer
	p          painter
	start      time.Time
	lastDraw   time.Time
	drawn      bool
	finished   bool // set by Finish; later events are ignored
	frame      int
	filesTotal int
	filesDone  int
	fileFrac   map[string]float64 // completion of each file seen so far, 0..1
	current    string             // file shown in the status text
	status     string
	stop       chan struct{}
	stopped    sync.Once
}

// newProgressBar creates a progress bar and starts redrawing it periodically,
// so the spinner and elapsed time keep moving during long steps (e.g. parsing).
func newProgressBar(out io.Writer, color bool) *progressBar {
	b := &progressBar{
		out:      out,
		p:        painter{enabled: color},
		start:    time.Now(),
		fileFrac: make(map[string]float64),
		stop:     make(chan struct{}),
	}
	go b.tick()
	return b
}

func (b *progressBar) tick() {
	ticker := time.NewTicker(progressInterval)
	defer ticker.Stop()
	for {
		select {
		case <-b.stop:
			return
		case <-ticker.C:
			b.mu.Lock()
			if b.drawn && time.Since(b.lastDraw) >= progressInterval {
				b.draw()
			}
			b.mu.Unlock()
		}
	}
}

// Handle updates the bar from a validation progress event.
func (b *progressBar) Handle(ev types.ProgressEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.finished {
		return
	}

	force := false
	switch ev.Stage {
	case types.StageDataset:
		b.filesTotal = ev.FilesTotal
	case types.StageSchema:
		b.setFile(ev.File, 0.02, "checking schema")
	case types.StageParsing:
		b.setFile(ev.File, 0.1, "parsing")
	case types.StageRules:
		frac := 0.15
		if ev.RulesTotal > 0 {
			frac += 0.85 * float64(ev.RulesDone) / float64(ev.RulesTotal)
		}
		b.setFile(ev.File, frac, fmt.Sprintf("rules %d/%d", ev.RulesDone, ev.RulesTotal))
	case types.StageFileDone:
		b.fileFrac[ev.File] = 1
		b.filesDone++
		force = true
	case types.StageReferences:
		b.current, b.status = "", "checking references"
		force = true
	}

	if force || time.Since(b.lastDraw) >= progressInterval {
		b.draw()
	}
}

func (b *progressBar) setFile(file string, frac float64, status string) {
	b.fileFrac[file] = frac
	b.current, b.status = file, status
}

func (b *progressBar) fraction() float64 {
	total := b.filesTotal
	if total < len(b.fileFrac) {
		total = len(b.fileFrac)
	}
	if total == 0 {
		return 0
	}
	sum := 0.0
	for _, f := range b.fileFrac {
		sum += f
	}
	return sum / float64(total)
}

// draw renders the progress line. Callers must hold b.mu.
func (b *progressBar) draw() {
	frac := b.fraction()
	filled := int(frac * progressBarWidth)
	bar := b.p.paint(styleCyan, strings.Repeat("█", filled)) + b.p.paint(styleDim, strings.Repeat("░", progressBarWidth-filled))

	spinner := spinnerFrames[b.frame%len(spinnerFrames)]
	b.frame++

	// Most useful information first, since the end may be truncated
	info := b.status
	if b.filesTotal > 1 {
		info = fmt.Sprintf("[%d/%d] %s", b.filesDone, b.filesTotal, info)
	}
	if b.current != "" {
		info += " · " + filepath.Base(b.current)
	}
	elapsed := formatElapsed(time.Since(b.start))

	prefix := fmt.Sprintf("%s %s %3.0f%%  ", b.p.paint(styleCyan, spinner), bar, frac*100)
	suffix := "  " + b.p.paint(styleDim, elapsed)
	// Visible width: spinner + space + bar + "  100%  " + info + suffix
	room := terminalWidth() - (2 + progressBarWidth + 7 + 2 + len(elapsed) + 1)
	info = truncate(info, room)

	_, _ = io.WriteString(b.out, clearLine+prefix+info+suffix)
	b.drawn = true
	b.lastDraw = time.Now()
}

// Write writes log output above the progress line.
func (b *progressBar) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.drawn {
		_, _ = io.WriteString(b.out, clearLine)
	}
	n, err := b.out.Write(p)
	if b.drawn {
		b.draw()
	}
	return n, err
}

// Finish stops the bar and removes the progress line. It is safe to call more than once.
func (b *progressBar) Finish() {
	b.stopped.Do(func() { close(b.stop) })
	b.mu.Lock()
	defer b.mu.Unlock()
	b.finished = true
	if b.drawn {
		_, _ = io.WriteString(b.out, clearLine)
		b.drawn = false
	}
}

// terminalWidth returns the width of the terminal on stderr, falling back to
// $COLUMNS and then 80.
func terminalWidth() int {
	if n, ok := terminalWidthOf(os.Stderr); ok && n > 20 {
		return n
	}
	if n, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && n > 20 {
		return n
	}
	return 80
}

func truncate(s string, width int) string {
	if width <= 1 {
		return ""
	}
	if utf8.RuneCountInString(s) <= width {
		return s
	}
	r := []rune(s)
	return string(r[:width-1]) + "…"
}

// formatElapsed formats a duration with exactly one decimal of seconds:
// "0.4s", "1.0s", "1m05.2s".
func formatElapsed(d time.Duration) string {
	tenths := int64(d / (100 * time.Millisecond))
	minutes, tenths := tenths/600, tenths%600
	if minutes > 0 {
		return fmt.Sprintf("%dm%02d.%ds", minutes, tenths/10, tenths%10)
	}
	return fmt.Sprintf("%d.%ds", tenths/10, tenths%10)
}
