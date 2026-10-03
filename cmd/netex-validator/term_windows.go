//go:build windows

package main

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procSetConsoleMode             = kernel32.NewProc("SetConsoleMode")
	procGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
)

const enableVirtualTerminalProcessing = 0x0004

// isTerminal reports whether f is a console that understands ANSI escapes,
// enabling ANSI processing if needed. Legacy consoles that can't do it are
// treated as non-terminals, so no escape codes are printed to them.
func isTerminal(f *os.File) bool {
	handle := syscall.Handle(f.Fd())
	var mode uint32
	if err := syscall.GetConsoleMode(handle, &mode); err != nil {
		return false
	}
	if mode&enableVirtualTerminalProcessing != 0 {
		return true
	}
	ok, _, _ := procSetConsoleMode.Call(uintptr(handle), uintptr(mode|enableVirtualTerminalProcessing))
	return ok != 0
}

// terminalWidthOf returns the width of the console window f is attached to.
func terminalWidthOf(f *os.File) (int, bool) {
	var info struct {
		Size, CursorPosition     struct{ X, Y int16 }
		Attributes               uint16
		Left, Top, Right, Bottom int16
		MaximumWindowSize        struct{ X, Y int16 }
	}
	ok, _, _ := procGetConsoleScreenBufferInfo.Call(f.Fd(), uintptr(unsafe.Pointer(&info))) //nolint:gosec // G103: required for the API call
	if ok == 0 {
		return 0, false
	}
	return int(info.Right-info.Left) + 1, true
}
