//go:build linux || darwin

package main

import (
	"os"
	"syscall"
	"unsafe"
)

type winsize struct{ Row, Col, Xpixel, Ypixel uint16 }

// getWinsize queries the terminal size of f; it fails if f is not a terminal.
func getWinsize(f *os.File) (winsize, bool) {
	var ws winsize
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws))) //nolint:gosec // G103: required for the ioctl
	return ws, errno == 0
}

// isTerminal reports whether f is a terminal that understands ANSI escapes.
func isTerminal(f *os.File) bool {
	_, ok := getWinsize(f)
	return ok
}

// terminalWidthOf returns the width of the terminal f is attached to, if known.
func terminalWidthOf(f *os.File) (int, bool) {
	ws, ok := getWinsize(f)
	if !ok || ws.Col == 0 {
		return 0, false
	}
	return int(ws.Col), true
}
