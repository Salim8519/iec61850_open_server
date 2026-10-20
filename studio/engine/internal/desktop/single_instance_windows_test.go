//go:build windows

package desktop

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"studio-engine/internal/testutil"
)

// Vectors from CPython 3.12.6 (studio/.venv): single_instance.event_name(case).
var eventNameVectors = []struct{ dir, folder, name string }{
	{`C:\Users\Salim\AppData\Local\ScadaBench`, `c:\users\salim\appdata\local\scadabench`, `Local\ScadaBench-Show-c2baaff68291a2b9a1d1c517`},
	{`c:/Temp/Some Folder/`, `c:\temp\some folder`, `Local\ScadaBench-Show-b561d14793e3fb571ce6cdc0`},
	{`D:\Data\ScadaBench-data\..\X`, `d:\data\x`, `Local\ScadaBench-Show-b7166b593cc2a0d42aa3ddff`},
	{"C:\\Donn\u00e9es\\\u00c4\u00d6\\\u0130x\u03a3", "c:\\donn\u00e9es\\\u00e4\u00f6\\\u0130x\u03c3", `Local\ScadaBench-Show-c2cbcaa6bdf50c44cb416351`},
	{`\\server\Share\ScadaBench`, `\\server\share\scadabench`, `Local\ScadaBench-Show-542c1477f4d3d66932ee26a4`},
	{`C:\a\b.\`, `c:\a\b`, `Local\ScadaBench-Show-6337bd8590408e37ec96ceca`},
	{`C:\`, `c:\`, `Local\ScadaBench-Show-3dc8472eaebe1a9d728dd16a`},
}

func TestShowEventNameMatchesPython(t *testing.T) {
	for _, vector := range eventNameVectors {
		if folder := normcase(abspath(vector.dir)); folder != vector.folder {
			t.Errorf("%q: folder %q want %q", vector.dir, folder, vector.folder)
		}
		if name := ShowEventName(vector.dir); name != vector.name {
			t.Errorf("%q: name %q want %q", vector.dir, name, vector.name)
		}
	}
}

// uniqueDir gives each test its own event name, so a running ScadaBench is never signalled.
func uniqueDir(t *testing.T) string {
	return filepath.Join(testutil.TempDir(t), fmt.Sprintf("desktop-test-%d-%d", os.Getpid(), time.Now().UnixNano()))
}

func TestRequestShowWithoutListener(t *testing.T) {
	if RequestShow(uniqueDir(t)) {
		t.Fatal("nobody listens, so nobody was asked")
	}
}

func TestShowListenerRoundTrip(t *testing.T) {
	dir := uniqueDir(t)
	shown := make(chan int, 4)
	calls := 0
	closeListener := StartShowListener(dir, func() {
		calls++
		if calls == 1 {
			panic("focus failed")
		}
		shown <- calls
	}, nil)
	defer closeListener()
	for want := 1; want <= 3; want++ {
		if !RequestShow(dir) {
			t.Fatalf("request %d was not delivered", want)
		}
		if want == 1 {
			time.Sleep(200 * time.Millisecond) // the first onShow panics; nothing arrives
			continue
		}
		select {
		case got := <-shown:
			if got != want {
				t.Fatalf("onShow call %d, want %d", got, want)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("request %d never reached onShow (listener stopped after a panic?)", want)
		}
	}
	closeListener()
	closeListener()
	if RequestShow(dir) {
		t.Fatal("a closed listener must release its event")
	}
}

func TestShowRunningInstanceAsksListener(t *testing.T) {
	if Headless() {
		t.Skip("needs a terminal; headless would show a message box")
	}
	dir := uniqueDir(t)
	shown := make(chan struct{}, 1)
	closeListener := StartShowListener(dir, func() { shown <- struct{}{} }, nil)
	if code := ShowRunningInstance(dir, errors.New("busy")); code != 0 {
		t.Fatalf("listener present: exit %d", code)
	}
	select {
	case <-shown:
	case <-time.After(5 * time.Second):
		t.Fatal("running instance was not asked to show")
	}
	closeListener()
	if code := ShowRunningInstance(dir, errors.New("busy")); code != 1 {
		t.Fatalf("no listener: exit %d", code)
	}
}

func TestFocusProcessWindowWithoutWindow(t *testing.T) {
	if FocusProcessWindow(0) {
		t.Fatal("pid 0 is never focused")
	}
	if FocusProcessWindow(os.Getpid()) {
		t.Fatal("the test process has no visible top-level window")
	}
}
