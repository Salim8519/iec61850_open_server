package licensing

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// eventually waits for a background search to finish.
func eventually(t *testing.T, what string, done func() bool) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if done() {
			return
		}
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestAProofLicensesAVMForTwentyFourHoursAndIsRenewedInTheBackgroundAfterSix(t *testing.T) {
	rig := newVMRig(t)
	rig.untilLicensed()
	first := rig.dials.Load()
	rig.advance(3600) // an hour later: still licensed, nobody is asked again
	if status := rig.status(); !status.Licensed || rig.dials.Load() != first {
		t.Fatalf("after an hour: %+v, %d asks", status, rig.dials.Load()-first)
	}
	rig.advance(6 * 3600) // seven hours: licensed, and a renewal starts by itself
	if status := rig.status(); !status.Licensed || status.Mode != ModeVMHost {
		t.Fatalf("after seven hours: %+v", status)
	}
	eventually(t, "the background renewal", func() bool { return rig.dials.Load() > first })
	eventually(t, "the renewed proof file", func() bool {
		rig.vm.vm.mu.Lock()
		defer rig.vm.vm.mu.Unlock()
		return rig.vm.vm.proof != nil && rig.vm.vm.proof.at == int64(*rig.now)
	})
}

func TestWhenTheHostIsGoneTheProofEndsAfterTwentyFourHours(t *testing.T) {
	rig := newVMRig(t)
	rig.untilLicensed()
	rig.stop() // the host PC is switched off
	rig.advance(23 * 3600)
	if status := rig.status(); !status.Licensed {
		t.Fatalf("after 23 hours: %+v", status)
	}
	rig.advance(2 * 3600) // 25 hours since the proof
	eventually(t, "the unlicensed state", func() bool { return !rig.status().Licensed })
	if status := rig.status(); status.Reason != NoLicense {
		t.Fatalf("the VM must say what to do: %+v", status)
	}
}

func TestAVMNeverGetsPastTheIntegrityCheckWhateverItsHostSays(t *testing.T) {
	rig := newVMRig(t)
	rig.vm = New(Options{Home: rig.vmHome, DataDir: rig.vmHome, Enforce: true, Keys: rig.host.signer.keys(),
		HWID: func() (string, error) { return rig.vmHWID, nil }, Clock: func() float64 { return *rig.now },
		Debugger: func() bool { return true }, Logf: t.Logf, Registry: newHKRegistry(),
		HostKey: HostKeyOptions{VM: &VMOptions{Hosts: func() []string { return []string{rig.address} }}}})
	status := rig.status()
	if status.Licensed || status.Reason == NoLicense {
		t.Fatalf("a debugged VM was licensed or lost its real reason: %+v", status)
	}
	if rig.dials.Load() != 0 {
		t.Fatal("a computer that failed the integrity check still asked a host")
	}
}

func TestAFailedSearchWaitsBeforeItStartsAgain(t *testing.T) {
	rig := newVMRig(t)
	rig.stop()
	rig.address = "127.0.0.1:1"
	rig.status() // starts the first search
	eventually(t, "the first search to end", func() bool {
		rig.vm.vm.mu.Lock()
		defer rig.vm.vm.mu.Unlock()
		return !rig.vm.vm.running && rig.vm.vm.nextTry > 0
	})
	asked := rig.dials.Load()
	for index := 0; index < 5; index++ {
		rig.status()
	}
	time.Sleep(50 * time.Millisecond)
	if rig.dials.Load() != asked {
		t.Fatalf("searches started again at once (%d -> %d)", asked, rig.dials.Load())
	}
	rig.advance(31) // thirty seconds later it may try again
	rig.status()
	eventually(t, "the second search", func() bool { return rig.dials.Load() > asked })
}

func TestTheHostsLicenseIsNeverReportedToTheLicenseServerFromAVM(t *testing.T) {
	rig := newVMRig(t)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
	defer server.Close()
	vm := New(Options{Home: rig.vmHome, DataDir: rig.vmHome, Enforce: true, Keys: rig.host.signer.keys(),
		HWID: func() (string, error) { return rig.vmHWID, nil }, Clock: func() float64 { return *rig.now },
		Debugger: func() bool { return false }, Logf: t.Logf, Registry: newHKRegistry(), AllowTestServer: true,
		ClientFactory: func(string) Client { calls++; return nil },
		HostKey:       HostKeyOptions{VM: &VMOptions{Hosts: func() []string { return []string{rig.address} }}}})
	if err := vm.RefreshVMHost(); err != nil {
		t.Fatal(err)
	}
	if verdict, err := vm.VerifyOnlineIfDue(false); err != nil || verdict != nil || calls != 0 {
		t.Fatalf("the VM asked the server about the host's license: %v %v (%d calls)", verdict, err, calls)
	}
	if revoked := vm.revoked.All(); len(revoked) != 0 {
		t.Fatalf("the VM cancelled the host's license locally: %v", revoked)
	}
}

func TestDevelopmentAndBuildsWithoutAHostSearchNeverAskAnyone(t *testing.T) {
	rig := newVMRig(t)
	build := func(enforce bool, vm *VMOptions) *Service {
		return New(Options{Home: hkTempDir(t), DataDir: hkTempDir(t), Enforce: enforce, Keys: rig.host.signer.keys(),
			HWID: func() (string, error) { return rig.vmHWID, nil }, Clock: func() float64 { return *rig.now },
			Debugger: func() bool { return false }, Logf: t.Logf, Registry: newHKRegistry(), HostKey: HostKeyOptions{VM: vm}})
	}
	options := &VMOptions{Hosts: func() []string { rig.dials.Add(1); return []string{rig.address} }}
	for name, service := range map[string]*Service{"development": build(false, options), "no search configured": build(true, nil)} {
		status, err := service.Current(true)
		if err != nil {
			t.Fatal(err)
		}
		if name == "development" && status.Mode != "development" {
			t.Fatalf("development: %+v", status)
		}
		if name != "development" && status.Licensed {
			t.Fatalf("%s: %+v", name, status)
		}
	}
	time.Sleep(50 * time.Millisecond)
	if rig.dials.Load() != 0 {
		t.Fatal("a build that must not search asked a host")
	}
	if err := build(true, nil).RefreshVMHost(); err == nil {
		t.Fatal("RefreshVMHost without a search configured succeeded")
	}
}

func TestTheStatusOfAnOrdinaryComputerStaysWhatItWas(t *testing.T) {
	rig := newVMRig(t)
	ordinary := New(Options{Home: hkTempDir(t), DataDir: hkTempDir(t), Enforce: true, Keys: rig.host.signer.keys(),
		HWID: func() (string, error) { return rig.vmHWID, nil }, Clock: func() float64 { return *rig.now },
		Debugger: func() bool { return false }, Logf: t.Logf, Registry: newHKRegistry()})
	status, err := ordinary.Status()
	if err != nil {
		t.Fatal(err)
	}
	for _, extra := range []string{"host_key", "binding"} {
		if _, has := status.Get(extra); has {
			t.Fatalf("an unlicensed computer's status carries %q", extra)
		}
	}
	if licensed, _ := status.Get("licensed"); licensed != false {
		t.Fatalf("status %v", status.Keys())
	}
}
