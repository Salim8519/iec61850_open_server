package main

import (
	"errors"
	"runtime"
	"syscall"
	"unsafe"
)

// A Win32 window and its message loop must stay on one OS thread.
func init() { runtime.LockOSThread() }

const windowClass = "ScadaBenchLicenseHelper"

// control describes one child control in 96-DPI units; they are scaled at creation.
type control struct {
	id         int
	class      string
	text       string
	style      uint32
	exStyle    uint32
	x, y, w, h int
}

// window is the helper's single top-level window (one per process).
type window struct {
	hwnd     uintptr
	controls map[int]uintptr
	clicks   map[int]func(*window)
}

var current *window

func (w *window) item(id int) uintptr { return w.controls[id] }

func windowProc(hwnd, message, wParam, lParam uintptr) uintptr {
	switch message {
	case wmCommand:
		if wParam>>16&0xFFFF == bnClicked && current != nil {
			if click := current.clicks[int(wParam&0xFFFF)]; click != nil {
				click(current)
				return 0
			}
		}
	case wmDestroy:
		procPostQuit.Call(0)
		return 0
	}
	result, _, _ := procDefWindowProc.Call(hwnd, message, wParam, lParam)
	return result
}

// runWindow shows a fixed-size window with the given controls and runs until it closes.
// ready runs once after the controls exist (fill in text, set focus).
func runWindow(title string, width, height int, controls []control,
	clicks map[int]func(*window), ready func(*window)) error {
	enableVisualStyles()
	dpi := screenDPI()
	scale := func(value int) int { return value * dpi / 96 }
	instance, _, _ := procModuleHandle.Call(0)
	cursor, _, _ := procLoadCursor.Call(0, idcArrow)
	class := wndClassEx{wndProc: syscall.NewCallback(windowProc), instance: instance, cursor: cursor,
		background: colorBtnFace + 1, className: wide(windowClass)}
	class.size = uint32(unsafe.Sizeof(class))
	if atom, _, err := procRegisterClass.Call(uintptr(unsafe.Pointer(&class))); atom == 0 {
		return errors.New("cannot register the window: " + err.Error())
	}
	outer := rect{right: int32(scale(width)), bottom: int32(scale(height))}
	procAdjustRect.Call(uintptr(unsafe.Pointer(&outer)), wsOverlappedFixed, 0)
	w, h := int(outer.right-outer.left), int(outer.bottom-outer.top)
	screenW, _, _ := procSystemMetrics.Call(0)
	screenH, _, _ := procSystemMetrics.Call(1)
	hwnd, _, err := procCreateWindow.Call(0, uintptr(unsafe.Pointer(wide(windowClass))),
		uintptr(unsafe.Pointer(wide(title))), wsOverlappedFixed,
		uintptr(max(0, (int(screenW)-w)/2)), uintptr(max(0, (int(screenH)-h)/2)), uintptr(w), uintptr(h),
		0, 0, instance, 0)
	if hwnd == 0 {
		return errors.New("cannot open the window: " + err.Error())
	}
	current = &window{hwnd: hwnd, controls: map[int]uintptr{}, clicks: clicks}
	font := newFont(scale(15))
	for _, c := range controls {
		child, _, _ := procCreateWindow.Call(uintptr(c.exStyle), uintptr(unsafe.Pointer(wide(c.class))),
			uintptr(unsafe.Pointer(wide(c.text))), uintptr(wsChild|wsVisible|c.style),
			uintptr(scale(c.x)), uintptr(scale(c.y)), uintptr(scale(c.w)), uintptr(scale(c.h)),
			hwnd, uintptr(c.id), instance, 0)
		procSendMessage.Call(child, wmSetFont, font, 1)
		current.controls[c.id] = child
	}
	ready(current)
	procShowWindow.Call(hwnd, 5) // SW_SHOW
	procUpdateWindow.Call(hwnd)
	var m msg
	for {
		got, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(got) <= 0 {
			return nil
		}
		// IsDialogMessage gives Tab / Shift+Tab between the boxes and buttons.
		if handled, _, _ := procIsDialogMessage.Call(hwnd, uintptr(unsafe.Pointer(&m))); handled != 0 {
			continue
		}
		procTranslate.Call(uintptr(unsafe.Pointer(&m)))
		procDispatch.Call(uintptr(unsafe.Pointer(&m)))
	}
}
