package main

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"studio-engine/internal/licensing/hostkey"
)

type noteLog struct {
	mu    sync.Mutex
	lines []string
}

func (l *noteLog) logf(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines = append(l.lines, strings.TrimSpace(strings.ReplaceAll(format, "%s", "")+strings.Join(stringify(args), " ")))
}

func stringify(args []any) []string {
	out := []string{}
	for _, a := range args {
		if text, ok := a.(string); ok {
			out = append(out, text)
		}
	}
	return out
}

func (l *noteLog) text() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return strings.Join(l.lines, "\n")
}

func adapterOf(name string, gateway bool, address string) hostkey.Adapter {
	ip, network, _ := net.ParseCIDR(address)
	a := hostkey.Adapter{Name: name, Up: true, IPv4: []net.IPNet{{IP: ip, Mask: network.Mask}}}
	if gateway {
		a.Gateways = []net.IP{net.ParseIP("192.168.1.1")}
	}
	return a
}

func TestTheHelperServesOnlyWhileThisPCsLicenseAllowsAndOnlyOnVirtualSwitches(t *testing.T) {
	var allowed atomic.Bool
	log := &noteLog{}
	h := &helper{logf: log.logf, wanted: func() bool { return true },
		check: func() error {
			if allowed.Load() {
				return nil
			}
			return errors.New("This license belongs to a different computer.")
		},
		adapters: func() ([]hostkey.Adapter, error) {
			return []hostkey.Adapter{adapterOf("Ethernet", true, "192.168.1.20/24"),
				adapterOf("vEthernet (Default Switch)", false, "172.20.80.1/20"),
				adapterOf("vEthernet (External)", true, "192.168.1.21/24")}, nil
		}}
	if endpoints := h.endpoints(); len(endpoints) != 0 {
		t.Fatalf("the helper opened %v although this PC's license does not allow it", endpoints)
	}
	if !strings.Contains(log.text(), "different computer") {
		t.Fatalf("the refusal was not written down in words: %q", log.text())
	}
	allowed.Store(true)
	endpoints := h.endpoints()
	if len(endpoints) != 1 || endpoints[0].Addr != "172.20.80.1:47862" || endpoints[0].Subnet == nil ||
		endpoints[0].Subnet.String() != "172.20.80.0/20" {
		t.Fatalf("endpoints %+v", endpoints)
	}
	allowed.Store(false) // the license is cancelled while the helper runs
	if endpoints := h.endpoints(); len(endpoints) != 0 {
		t.Fatalf("the helper kept serving after the license was refused: %v", endpoints)
	}
}

func TestTheHelperEndsByItselfWhenSharingIsSwitchedOff(t *testing.T) {
	var wanted atomic.Bool
	wanted.Store(true)
	h := &helper{logf: (&noteLog{}).logf, wanted: wanted.Load, check: func() error { return nil },
		adapters:  func() ([]hostkey.Adapter, error) { return nil, nil },
		pollEvery: 10 * time.Millisecond, rescan: 10 * time.Millisecond,
		extra: func() []hostkey.Endpoint { return []hostkey.Endpoint{{Addr: "127.0.0.1:0"}} }}
	done := make(chan struct{})
	go func() { h.run(context.Background()); close(done) }()
	select {
	case <-done:
		t.Fatal("the helper ended while sharing was on")
	case <-time.After(100 * time.Millisecond):
	}
	wanted.Store(false) // the Run value was removed in Settings
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("the helper kept running after sharing was switched off")
	}
}

func TestAnAdapterListThatCannotBeReadIsSaidOnceAndServesNothing(t *testing.T) {
	log := &noteLog{}
	h := &helper{logf: log.logf, check: func() error { return nil },
		adapters: func() ([]hostkey.Adapter, error) { return nil, errors.New("the adapter list is locked") }}
	for index := 0; index < 3; index++ {
		if endpoints := h.endpoints(); len(endpoints) != 0 {
			t.Fatalf("endpoints %v", endpoints)
		}
	}
	if count := strings.Count(log.text(), "adapter list is locked"); count != 1 {
		t.Fatalf("the same problem was written %d times", count)
	}
}

func TestTheReleaseProgramTakesNoInputFromTheEnvironmentAndHasNoTestSeam(t *testing.T) {
	if testSeam != nil {
		t.Fatal("a seam is installed in a build without the hostkeytest tag")
	}
	sources, err := filepath.Glob("*.go")
	if err != nil || len(sources) < 3 {
		t.Fatalf("sources %v %v", sources, err)
	}
	for _, name := range sources {
		if strings.HasSuffix(name, "_test.go") || name == "seam_hostkeytest.go" {
			continue
		}
		data, _ := os.ReadFile(name)
		for _, banned := range []string{"SCADABENCH_HOSTKEY_TEST", "os.Getenv", "os.Args", "flag."} {
			if strings.Contains(string(data), banned) {
				t.Errorf("%s contains %q: a release program takes no input from outside", name, banned)
			}
		}
	}
}

func TestTheLogStartsAfreshWhenItGrowsTooBig(t *testing.T) {
	dir := t.TempDir()
	file := openLog(dir)
	if file == nil {
		t.Fatal("no log file")
	}
	file.WriteString("one line\n")
	file.Close()
	path := filepath.Join(dir, "vm_host.log")
	if err := os.WriteFile(path, make([]byte, 300*1024), 0o666); err != nil {
		t.Fatal(err)
	}
	file = openLog(dir)
	file.Close()
	if info, err := os.Stat(path); err != nil || info.Size() != 0 {
		t.Fatalf("the oversized log was kept: %v", err)
	}
}
