package licensing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/licensing/hostkey"
)

// The clocks of the VM proof (ADR-0099, LIC-81): the host signs its own time into every proof, so the
// time in vm_proof.json cannot be edited, a host with a wrong clock is refused, and a clock set back
// cannot bring an old proof back.

// licensedFrom reports whether a computer that starts with proof file text (and a host that is gone)
// is licensed.
func licensedFrom(t *testing.T, rig *vmRig, text string) bool {
	t.Helper()
	home := hkTempDir(t)
	if err := os.WriteFile(filepath.Join(home, VMProofFile), []byte(text), 0o666); err != nil {
		t.Fatal(err)
	}
	status, err := rig.vmOn(home).Current(true)
	if err != nil {
		t.Fatal(err)
	}
	return status.Licensed && status.Mode == ModeVMHost
}

// editedFile is the proof file with its fields changed afterwards (what a text editor does).
func editedFile(t *testing.T, data []byte, change func(map[string]any)) string {
	t.Helper()
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	change(document)
	text, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	return string(text)
}

func TestAVMThatRestartsStaysLicensedFromItsProofFileWithoutAskingAnyone(t *testing.T) {
	rig := newVMRig(t)
	rig.untilLicensed()
	rig.stop()
	asked := rig.dials.Load()
	rig.advance(3600)
	status, err := rig.vmOn(rig.vmHome).Current(true)
	if err != nil || !status.Licensed || status.Mode != ModeVMHost || status.Plan != "lifetime" {
		t.Fatalf("the restarted VM: %+v (%v)", status, err)
	}
	if rig.dials.Load() != asked {
		t.Fatal("a valid proof file was not enough: the VM asked the network")
	}
}

func TestEditingTheTimeInTheProofFileMakesItWorthlessToTheBackend(t *testing.T) {
	rig := newVMRig(t)
	rig.untilLicensed()
	rig.stop() // nobody can renew it
	original, err := os.ReadFile(filepath.Join(rig.vmHome, VMProofFile))
	if err != nil {
		t.Fatal(err)
	}
	signed := int64(*rig.now)
	rig.advance(3600)
	if !licensedFrom(t, rig, string(original)) {
		t.Fatal("the unedited proof must license the VM an hour later (the edits below must be what changes it)")
	}
	younger := editedFile(t, original, func(d map[string]any) { d["time"] = int64(*rig.now) })
	oneSecond := editedFile(t, original, func(d map[string]any) { d["time"] = signed + 1 })
	asFloat := editedFile(t, original, func(d map[string]any) { d["time"] = float64(signed) + 0.5 })
	asText := editedFile(t, original, func(d map[string]any) { d["time"] = "1760000000" })
	for name, text := range map[string]string{"made younger": younger, "one second later": oneSecond,
		"a fraction": asFloat, "as text": asText} {
		if licensedFrom(t, rig, text) {
			t.Errorf("a proof file whose time was edited (%s) licensed the VM", name)
		}
	}
	rig.advance(25 * 3600) // the proof is now older than a day: the one attack worth making
	if licensedFrom(t, rig, string(original)) {
		t.Fatal("a proof older than 24 hours licensed the VM")
	}
	if licensedFrom(t, rig, editedFile(t, original, func(d map[string]any) { d["time"] = int64(*rig.now) })) {
		t.Fatal("an expired proof was made young again by editing its time")
	}
}

func TestAProofThatBelongsToAnotherMachineOrIsDamagedIsNeverLoaded(t *testing.T) {
	rig := newVMRig(t)
	rig.untilLicensed()
	rig.stop()
	original, _ := os.ReadFile(filepath.Join(rig.vmHome, VMProofFile))
	if !licensedFrom(t, rig, string(original)) {
		t.Fatal("the unedited proof must license this VM")
	}
	other := editedFile(t, original, func(d map[string]any) { d["vm_hwid"] = strings.Repeat("cd", 32) })
	nonce := editedFile(t, original, func(d map[string]any) { d["nonce"] = strings.Repeat("00", 16) })
	for name, text := range map[string]string{"another machine": other, "another nonce": nonce, "empty": "",
		"not JSON": "{", "an array": "[]", "no certificate": editedFile(t, original, func(d map[string]any) { delete(d, "certificate") })} {
		if licensedFrom(t, rig, text) {
			t.Errorf("%s licensed the VM", name)
		}
	}
}

func TestAHostWhoseClockIsMoreThanTenMinutesOffIsRefusedByTheVM(t *testing.T) {
	for _, skew := range []float64{1200, -1200, 601, -601} {
		rig := newVMRig(t)
		rig.host.skew = skew
		err := rig.vm.RefreshVMHost()
		if err == nil || !strings.Contains(err.Error(), VMClockSkewText) {
			t.Fatalf("a host %.0f s off was not refused with the clock text: %v", skew, err)
		}
		if _, statErr := os.Stat(filepath.Join(rig.vmHome, VMProofFile)); statErr == nil {
			t.Errorf("a proof file was written for a host %.0f s off", skew)
		}
		if status := rig.status(); status.Licensed {
			t.Errorf("a host %.0f s off licensed the VM: %+v", skew, status)
		}
	}
	for _, skew := range []float64{0, 599, -599, 600, -600} {
		rig := newVMRig(t)
		rig.host.skew = skew
		if err := rig.vm.RefreshVMHost(); err != nil {
			t.Errorf("a host %.0f s off (inside ten minutes) was refused: %v", skew, err)
		}
	}
}

func TestAnAnswerWhoseTimeWasChangedOrLeftOutIsRefused(t *testing.T) {
	rig := newVMRig(t)
	for name, change := range map[string]func(*hostkey.Answer){
		"a second later":  func(a *hostkey.Answer) { a.Time++ },
		"a second before": func(a *hostkey.Answer) { a.Time-- },
		"no time":         func(a *hostkey.Answer) { a.Time = 0 },
		"a negative time": func(a *hostkey.Answer) { a.Time = -a.Time },
	} {
		rig.vm = rig.newVM()
		rig.dial = dynamic(func(request hostkey.Request) hostkey.Answer {
			answer := rig.answerOf(request)
			change(&answer)
			return answer
		})
		if err := rig.vm.RefreshVMHost(); err == nil {
			t.Errorf("an answer with %s was accepted", name)
		}
	}
	if status := rig.status(); status.Licensed {
		t.Fatalf("the VM was licensed by a changed answer: %+v", status)
	}
}

func TestSettingTheVMClockBackDoesNotBringAnExpiredProofBack(t *testing.T) {
	rig := newVMRig(t)
	rig.untilLicensed()
	rig.stop() // the host is gone: only the proof it signed is left
	rig.advance(23 * 3600)
	if status := rig.status(); !status.Licensed {
		t.Fatalf("23 hours after the proof: %+v", status)
	}
	rig.advance(2 * 3600) // 25 hours after the proof: it has run out, and the VM has seen that time
	if status := rig.status(); status.Licensed {
		t.Fatalf("25 hours after the proof: %+v", status)
	}
	for _, back := range []float64{-15 * 3600, -24 * 3600, -3600} {
		rig.advance(back) // someone sets the clock back to where the proof looks young by the clock
		if status := rig.status(); status.Licensed {
			t.Fatalf("an expired proof was revived by setting the clock back %.0f s: %+v", -back, status)
		}
		rig.advance(-back)
	}
	// The same through a restart: the file is read again, and the newest time seen is still known.
	rig.advance(-15 * 3600)
	if status, err := rig.vmOn(rig.vmHome).Current(true); err != nil || status.Licensed {
		t.Fatalf("a restart with the clock set back: %+v (%v)", status, err)
	}
}

func TestAClockThatWobblesBackLittleNeverCostsAProofItsLife(t *testing.T) {
	rig := newVMRig(t)
	rig.untilLicensed()
	rig.stop()
	rig.advance(20 * 3600)
	rig.status()
	rig.advance(-120) // a clock correction of two minutes
	if status := rig.status(); !status.Licensed {
		t.Fatalf("a two-minute correction ended the proof: %+v", status)
	}
}

func TestTheProofItselfRefusesAnAgeBeyondADayOrTheNewestTimeSeen(t *testing.T) {
	rig := newVMRig(t)
	now := int64(*rig.now)
	if err := rig.vm.proofLive(now, now-24*3600); err != nil {
		t.Fatalf("a proof exactly a day old is still allowed: %v", err)
	}
	if err := rig.vm.proofLive(now, now-24*3600-1); err == nil || err.Error() != VMProofOldText {
		t.Fatalf("a proof a second older than a day: %v", err)
	}
	if err := rig.vm.proofLive(now, now+600); err != nil {
		t.Fatalf("a proof ten minutes ahead is allowed: %v", err)
	}
	if err := rig.vm.proofLive(now, now+601); err == nil || err.Error() != VMProofOldText {
		t.Fatalf("a proof from more than ten minutes ahead: %v", err)
	}
	// The newest time ever seen is 200 s further on: the clock was set back 200 s, which the certificate's
	// own 300 s tolerance allows, but a proof that is then more than a day old by that time is refused.
	rig.vm.anchor.Advance(now + 200)
	if err := rig.vm.proofLive(now, now-24*3600); err == nil || err.Error() != ClockRollback {
		t.Fatalf("a day-old proof with a newer anchor: %v", err)
	}
	if err := rig.vm.proofLive(now, now-3600); err != nil {
		t.Fatalf("a young proof with a newer anchor: %v", err)
	}
}
