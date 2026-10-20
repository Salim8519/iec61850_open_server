//go:build windows

package desktop

import (
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var procLCMapStringEx = kernel32.NewProc("LCMapStringEx")

const lcmapLowercase = 0x00000100

// abspath is CPython 3.12 ntpath.abspath: GetFullPathNameW(normpath(path)), which also
// drops trailing dots and spaces of the last component ("C:\a\b.\" -> "C:\a\b").
func abspath(path string) string {
	cleaned := filepath.Clean(path)
	if full, err := syscall.FullPath(cleaned); err == nil {
		return full
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return cleaned
	}
	return absolute
}

// normcase is CPython 3.12 ntpath.normcase: "/" -> "\" then
// LCMapStringEx(LOCALE_NAME_INVARIANT, LCMAP_LOWERCASE). Unlike strings.ToLower it keeps
// characters such as U+0130 unchanged, as Python does.
func normcase(path string) string {
	path = strings.ReplaceAll(path, "/", `\`)
	if path == "" {
		return path
	}
	source := utf16.Encode([]rune(path))
	locale, _ := syscall.UTF16PtrFromString("")
	size, _, _ := procLCMapStringEx.Call(uintptr(unsafe.Pointer(locale)), lcmapLowercase,
		uintptr(unsafe.Pointer(&source[0])), uintptr(len(source)), 0, 0, 0, 0, 0)
	if size == 0 {
		return strings.ToLower(path)
	}
	target := make([]uint16, size)
	written, _, _ := procLCMapStringEx.Call(uintptr(unsafe.Pointer(locale)), lcmapLowercase,
		uintptr(unsafe.Pointer(&source[0])), uintptr(len(source)),
		uintptr(unsafe.Pointer(&target[0])), size, 0, 0, 0)
	if written == 0 {
		return strings.ToLower(path)
	}
	return string(utf16.Decode(target[:written]))
}
