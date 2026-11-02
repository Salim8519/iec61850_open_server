package runtime

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
	"testing"

	"studio-engine/internal/apperr"
	"studio-engine/internal/portholder"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

func stubHolder(t *testing.T, holder portholder.Holder, found bool) {
	saved := holderOf
	holderOf = func(string, int) (portholder.Holder, bool) { return holder, found }
	t.Cleanup(func() { holderOf = saved })
}

func bindError(code syscall.Errno) error {
	return &net.OpError{Op: "listen", Net: "tcp4", Err: os.NewSyscallError("bind", code)}
}

// With no holder found, "in use" keeps the old message word for word.
func TestListenProblemInUseKeepsTheOldText(t *testing.T) {
	stubHolder(t, portholder.Holder{}, false)
	failure := listenProblem("C06_BCU", "127.50.0.2", 102, bindError(wsaAddressInUse))
	old := fmt.Sprintf("C06_BCU cannot listen on 127.50.0.2:102 because another program is using it (%s). "+
		"Close that program or change the port.", osErrorText(bindError(wsaAddressInUse)))
	if failure.Error() != old {
		t.Fatalf("message %q\nwant %q", failure.Error(), old)
	}
	var studio *apperr.Error
	if !errors.As(failure, &studio) || studio.Code != "validation" {
		t.Fatal("not answered as a Studio error")
	}
	if _, has := failure.event.Get("holder_pid"); has {
		t.Fatal("event names a holder that was not found")
	}
}

func TestListenProblemNamesWhatWindowsAnswered(t *testing.T) {
	everywhere := portholder.Holder{PID: 4120, Program: "s7oiehsx64.exe", Address: portholder.Any}
	cases := []struct {
		name   string
		holder portholder.Holder
		found  bool
		code   syscall.Errno
		want   []string
	}{
		{"holds every address", everywhere, true, wsaAccess,
			[]string{"s7oiehsx64.exe (process 4120) holds port 102 on every address", "[WinError 10013]"}},
		{"refused, nobody found", portholder.Holder{}, false, wsaAccess,
			[]string{"Windows refused it", "another program holds port 102 for itself", "security software"}},
		{"in use, named", portholder.Holder{PID: 77, Program: "other.exe", Address: "127.50.0.2"}, true,
			wsaAddressInUse, []string{"because other.exe (process 77) is using it"}},
		{"address not usable", portholder.Holder{}, false, wsaAddressNotValid,
			[]string{"not usable on this PC right now", "cable is plugged in", "[WinError 10049]"}},
	}
	for _, item := range cases {
		stubHolder(t, item.holder, item.found)
		failure := listenProblem("C06_BCU", "127.50.0.2", 102, bindError(item.code))
		for _, part := range item.want {
			if !strings.Contains(failure.Error(), part) {
				t.Errorf("%s: %q lacks %q", item.name, failure.Error(), part)
			}
		}
		if got := pyval.Str(failure.event, "error"); !strings.Contains(got, fmt.Sprint(int(item.code))) {
			t.Errorf("%s: event error %q", item.name, got)
		}
		if item.found && pyval.Str(failure.event, "holder_program") != item.holder.Program {
			t.Errorf("%s: event %v", item.name, failure.event)
		}
	}
}

// A real holder on the IED's address: the check still refuses, and names this test program.
func TestPreflightNamesARealHolder(t *testing.T) {
	config := net.ListenConfig{Control: exclusiveAddress}
	holder, err := config.Listen(contextBackground(), "tcp4", "127.50.0.11:0")
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Close()
	port := holder.Addr().(*net.TCPAddr).Port
	ied := pyval.Obj("id", "a", "name", "A", "ip", "127.50.0.11", "port", int64(port))
	err = preflight([]*pyjson.Object{ied}, pyval.Obj("mode", "loopback"), nil)
	var refused *listenFailure
	if !errors.As(err, &refused) {
		t.Fatalf("preflight answered %v", err)
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("(process %d)", os.Getpid())) {
		t.Fatalf("holder not named: %v", err)
	}
	if pyval.Str(refused.event, "endpoint") != fmt.Sprintf("127.50.0.11:%d", port) {
		t.Fatalf("event %v", refused.event)
	}
}
