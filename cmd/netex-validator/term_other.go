//go:build !linux && !darwin && !windows

package main

import "os"

// isTerminal reports whether f looks like a terminal.
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// terminalWidthOf is not supported on this platform.
func terminalWidthOf(*os.File) (int, bool) { return 0, false }
