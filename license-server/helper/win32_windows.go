package main

import (
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

// Plain Win32 through the standard syscall package: no GUI library, no x/sys, so the
// helper stays one small file with no networking code. user32 and gdi32 are KnownDLLs
// and always load from System32.
var (
	user32              = syscall.NewLazyDLL("user32.dll")
	gdi32               = syscall.NewLazyDLL("gdi32.dll")
	procRegisterClass   = user32.NewProc("RegisterClassExW")
	procCreateWindow    = user32.NewProc("CreateWindowExW")
	procDefWindowProc   = user32.NewProc("DefWindowProcW")
	procGetMessage      = user32.NewProc("GetMessageW")
	procTranslate       = user32.NewProc("TranslateMessage")
	procDispatch        = user32.NewProc("DispatchMessageW")
	procIsDialogMessage = user32.NewProc("IsDialogMessageW")
	procPostQuit        = user32.NewProc("PostQuitMessage")
	procSendMessage     = user32.NewProc("SendMessageW")
	procSetWindowText   = user32.NewProc("SetWindowTextW")
	procGetWindowText   = user32.NewProc("GetWindowTextW")
	procGetTextLength   = user32.NewProc("GetWindowTextLengthW")
	procShowWindow      = user32.NewProc("ShowWindow")
	procUpdateWindow    = user32.NewProc("UpdateWindow")
	procLoadCursor      = user32.NewProc("LoadCursorW")
	procMessageBox      = user32.NewProc("MessageBoxW")
	procSetDPIAware     = user32.NewProc("SetProcessDPIAware")
	procGetDC           = user32.NewProc("GetDC")
	procReleaseDC       = user32.NewProc("ReleaseDC")
	procSetFocus        = user32.NewProc("SetFocus")
	procAdjustRect      = user32.NewProc("AdjustWindowRect")
	procSystemMetrics   = user32.NewProc("GetSystemMetrics")
	procCreateFont      = gdi32.NewProc("CreateFontW")
	procDeviceCaps      = gdi32.NewProc("GetDeviceCaps")
	procCreateActCtx    = kernel32.NewProc("CreateActCtxW")
	procActivateActCtx  = kernel32.NewProc("ActivateActCtx")
	procModuleHandle    = kernel32.NewProc("GetModuleHandleW")
	procSystemDirectory = kernel32.NewProc("GetSystemDirectoryW")
)

const (
	wsOverlappedFixed = 0x00C00000 | 0x00080000 | 0x00020000 // caption, close, minimise
	wsChild           = 0x40000000
	wsVisible         = 0x10000000
	wsTabStop         = 0x00010000
	wsExClientEdge    = 0x00000200
	esPassword        = 0x0020
	esAutoHScroll     = 0x0080
	wmDestroy         = 0x0002
	wmSetFont         = 0x0030
	wmCommand         = 0x0111
	bnClicked         = 0
	colorBtnFace      = 15
	idcArrow          = 32512
	logPixelsY        = 90
	mbIconError       = 0x10
)

type point struct{ x, y int32 }

type rect struct{ left, top, right, bottom int32 }

type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
	private uint32
}

type wndClassEx struct {
	size       uint32
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   uintptr
	icon       uintptr
	cursor     uintptr
	background uintptr
	menuName   *uint16
	className  *uint16
	iconSmall  uintptr
}

type actCtx struct {
	size         uint32
	flags        uint32
	source       *uint16
	architecture uint16
	langID       uint16
	assemblyDir  *uint16
	resourceName uintptr
	application  *uint16
	module       uintptr
}

func wide(text string) *uint16 {
	pointer, _ := syscall.UTF16PtrFromString(text)
	return pointer
}

func setText(hwnd uintptr, text string) {
	text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\n", "\r\n")
	procSetWindowText.Call(hwnd, uintptr(unsafe.Pointer(wide(text))))
}

// getText reads a control's text; the temporary buffer is zeroed afterwards so a
// typed password does not linger in that copy.
func getText(hwnd uintptr) string {
	length, _, _ := procGetTextLength.Call(hwnd)
	buffer := make([]uint16, length+1)
	procGetWindowText.Call(hwnd, uintptr(unsafe.Pointer(&buffer[0])), length+1)
	text := syscall.UTF16ToString(buffer)
	clear(buffer)
	return text
}

func showError(text string) {
	procMessageBox.Call(0, uintptr(unsafe.Pointer(wide(text))),
		uintptr(unsafe.Pointer(wide("ScadaBench License Helper"))), mbIconError)
}

// enableVisualStyles turns on modern Windows controls by activating the common-controls
// v6 manifest that ships inside System32\shell32.dll (resource 124). Best effort: when
// it fails the window still works, with the classic look.
func enableVisualStyles() {
	buffer := make([]uint16, 260)
	n, _, _ := procSystemDirectory.Call(uintptr(unsafe.Pointer(&buffer[0])), 260)
	if n == 0 {
		return
	}
	shell := filepath.Join(syscall.UTF16ToString(buffer[:n]), "shell32.dll")
	context := actCtx{flags: 0x008, source: wide(shell), resourceName: 124} // ACTCTX_FLAG_RESOURCE_NAME_VALID
	context.size = uint32(unsafe.Sizeof(context))
	handle, _, _ := procCreateActCtx.Call(uintptr(unsafe.Pointer(&context)))
	if handle == 0 || handle == ^uintptr(0) {
		return
	}
	var cookie uintptr
	procActivateActCtx.Call(handle, uintptr(unsafe.Pointer(&cookie)))
}

// screenDPI makes the window sharp and correctly sized on high-resolution screens.
func screenDPI() int {
	procSetDPIAware.Call()
	dc, _, _ := procGetDC.Call(0)
	defer procReleaseDC.Call(0, dc)
	dpi, _, _ := procDeviceCaps.Call(dc, logPixelsY)
	if dpi == 0 {
		return 96
	}
	return int(dpi)
}

func newFont(height int) uintptr {
	font, _, _ := procCreateFont.Call(uintptr(-height), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0,
		uintptr(unsafe.Pointer(wide("Segoe UI")))) // DEFAULT_CHARSET, CLEARTYPE_QUALITY
	return font
}
