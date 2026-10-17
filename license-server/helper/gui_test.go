package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

var (
	procFindWindowEx  = user32.NewProc("FindWindowExW")
	procWindowProcess = user32.NewProc("GetWindowThreadProcessId")
	procGetDlgItem    = user32.NewProc("GetDlgItem")
	procPostMessage   = user32.NewProc("PostMessageW")
)

const wmSetText, wmGetText, wmClose = 0x000C, 0x000D, 0x0010

// Builds the real windowed program, seals it like setup does, opens it and works the
// real window from outside: type into the password box and press the two buttons.
// LOCALAPPDATA points at a temporary folder, so this PC's real license is untouched.
func TestHelperWindowMakesAndRemovesWithTwoButtons(t *testing.T) {
	work := t.TempDir()
	plain := filepath.Join(work, "setup.exe")
	build := exec.Command("go", "build", "-trimpath", "-ldflags", "-H=windowsgui", "-o", plain, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Skipf("cannot build the helper: %v %s", err, out)
	}
	program, _ := os.ReadFile(plain)
	sealed := filepath.Join(work, sealedName)
	if err := os.WriteFile(sealed, appendVault(program, testVault(t)), 0o700); err != nil {
		t.Fatal(err)
	}
	appData := filepath.Join(work, "appdata")
	t.Setenv("LOCALAPPDATA", appData)
	run := exec.Command(sealed)
	if err := run.Start(); err != nil {
		t.Fatal(err)
	}
	defer run.Process.Kill()
	hwnd := findHelperWindow(t, run.Process.Pid)
	item := func(id int) uintptr { child, _, _ := procGetDlgItem.Call(hwnd, uintptr(id)); return child }
	press := func(password string, button int) string {
		procSendMessage.Call(item(idPassword), wmSetText, 0, uintptr(unsafe.Pointer(wide(password))))
		procSendMessage.Call(hwnd, wmCommand, uintptr(button), item(button)) // BN_CLICKED in the high word = 0
		return windowText(item(idResult))
	}
	if !strings.HasPrefix(windowText(item(idStatus)), "This PC: no license") {
		t.Fatalf("status at start: %q", windowText(item(idStatus)))
	}
	if got := press("not the password", idMake); !strings.Contains(got, "wrong password") {
		t.Fatalf("wrong password: %q", got)
	}
	if got := press(testPassword, idMake); !strings.Contains(got, "licensed forever") {
		t.Fatalf("make: %q", got)
	}
	payload, err := readPayload(licenseHome())
	if mine, _ := machineID(); err != nil || payload.HWID != mine {
		t.Fatalf("license after the Make button: %+v, %v", payload, err)
	}
	if !strings.HasPrefix(windowText(item(idStatus)), "This PC: licensed forever") {
		t.Fatalf("status after make: %q", windowText(item(idStatus)))
	}
	if got := press(testPassword, idMake); !strings.Contains(got, "already has a working license") {
		t.Fatalf("second make: %q", got)
	}
	saved := clipboardText()
	defer setClipboardText(0, saved)
	if got := press("", idCopy); !strings.Contains(got, "Copied the license text") {
		t.Fatalf("copy: %q", got)
	}
	if file, _ := os.ReadFile(filepath.Join(licenseHome(), "license.cert")); clipboardText() != string(file) {
		t.Fatal("the clipboard does not hold the license text")
	}
	if got := press(testPassword, idRemove); !strings.Contains(got, "removed") {
		t.Fatalf("remove: %q", got)
	}
	if _, err := os.Stat(filepath.Join(appData, "ScadaBench", "license.cert")); !os.IsNotExist(err) {
		t.Fatal("the license is still active after the Remove button")
	}
	procPostMessage.Call(hwnd, wmClose, 0, 0)
	done := make(chan error, 1)
	go func() { done <- run.Wait() }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the window did not close")
	}
}

// findHelperWindow returns the helper window owned by the process this test started,
// never another helper the owner may have open on the same desktop.
func findHelperWindow(t *testing.T, pid int) uintptr {
	t.Helper()
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		var hwnd uintptr
		for {
			hwnd, _, _ = procFindWindowEx.Call(0, hwnd, uintptr(unsafe.Pointer(wide(windowClass))), 0)
			if hwnd == 0 {
				break
			}
			var owner uint32
			procWindowProcess.Call(hwnd, uintptr(unsafe.Pointer(&owner)))
			if int(owner) == pid {
				return hwnd
			}
		}
	}
	t.Fatal("the helper window did not open")
	return 0
}

func clipboardText() string {
	procOpenClipboard.Call(0)
	defer procCloseClipboard.Call()
	memory, _, _ := user32.NewProc("GetClipboardData").Call(cfUnicodeText)
	if memory == 0 {
		return ""
	}
	size, _, _ := kernel32.NewProc("GlobalSize").Call(memory)
	buffer := make([]uint16, size/2+1)
	pointer, _, _ := procGlobalLock.Call(memory)
	procMoveMemory.Call(uintptr(unsafe.Pointer(&buffer[0])), pointer, size)
	procGlobalUnlock.Call(memory)
	return syscall.UTF16ToString(buffer)
}

func windowText(hwnd uintptr) string {
	buffer := make([]uint16, 1024)
	procSendMessage.Call(hwnd, wmGetText, uintptr(len(buffer)), uintptr(unsafe.Pointer(&buffer[0])))
	return syscall.UTF16ToString(buffer)
}
