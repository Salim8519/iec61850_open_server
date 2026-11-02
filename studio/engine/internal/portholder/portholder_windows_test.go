//go:build windows

package portholder

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The test itself listens on a loopback address; Find names this test program.
func TestFindNamesTheListeningProgram(t *testing.T) {
	socket, err := net.Listen("tcp4", "127.50.0.9:0")
	if err != nil {
		t.Fatal(err)
	}
	defer socket.Close()
	port := socket.Addr().(*net.TCPAddr).Port
	holder, found := Find("127.50.0.9", port)
	self, _ := os.Executable()
	if !found || holder.PID != uint32(os.Getpid()) || !strings.EqualFold(holder.Program, filepath.Base(self)) ||
		holder.Address != "127.50.0.9" {
		t.Fatalf("holder %+v found %v; want this program (%d, %s)", holder, found, os.Getpid(), filepath.Base(self))
	}
	if _, found := Find("127.50.0.10", port); found {
		t.Fatal("a listener on one address was reported for another")
	}
	if got := programOf(systemPID); got != "Windows (System)" {
		t.Fatal(got)
	}
}
