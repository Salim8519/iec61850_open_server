package main

import (
	"errors"
	"syscall"
	"unsafe"
)

var (
	procOpenClipboard  = user32.NewProc("OpenClipboard")
	procEmptyClipboard = user32.NewProc("EmptyClipboard")
	procSetClipboard   = user32.NewProc("SetClipboardData")
	procCloseClipboard = user32.NewProc("CloseClipboard")
	procGlobalAlloc    = kernel32.NewProc("GlobalAlloc")
	procGlobalLock     = kernel32.NewProc("GlobalLock")
	procGlobalUnlock   = kernel32.NewProc("GlobalUnlock")
	procGlobalFree     = kernel32.NewProc("GlobalFree")
	procMoveMemory     = kernel32.NewProc("RtlMoveMemory")
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002
)

// setClipboardText puts text on the Windows clipboard, ready for Ctrl+V.
func setClipboardText(owner uintptr, text string) error {
	chars, err := syscall.UTF16FromString(text)
	if err != nil {
		return err
	}
	if ok, _, _ := procOpenClipboard.Call(owner); ok == 0 {
		return errors.New("the clipboard is busy; try again")
	}
	defer procCloseClipboard.Call()
	procEmptyClipboard.Call()
	size := uintptr(len(chars) * 2)
	memory, _, _ := procGlobalAlloc.Call(gmemMoveable, size)
	if memory == 0 {
		return errors.New("not enough memory for the clipboard")
	}
	target, _, _ := procGlobalLock.Call(memory)
	if target == 0 {
		procGlobalFree.Call(memory)
		return errors.New("cannot use the clipboard")
	}
	procMoveMemory.Call(target, uintptr(unsafe.Pointer(&chars[0])), size)
	procGlobalUnlock.Call(memory)
	if done, _, _ := procSetClipboard.Call(cfUnicodeText, memory); done == 0 {
		procGlobalFree.Call(memory)
		return errors.New("cannot put the text on the clipboard")
	}
	return nil // the clipboard now owns the memory
}
