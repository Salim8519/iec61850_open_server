package store

import (
	"errors"
	"strings"
	"testing"
	"time"

	"substation-license-server/internal/certs"
)

var trialNow = time.Unix(1_760_000_000, 0)

const month = 30 * 24 * time.Hour

func parts(guid, volume, firmware string) TrialParts {
	hash := func(kind, value string) string {
		if value == "" {
			return ""
		}
		return certs.PartHash(kind, value)
	}
	return TrialParts{GUID: hash("guid", guid), Volume: hash("volume", volume), Firmware: hash("firmware", firmware)}
}

func machine(n int) string { return strings.Repeat(string(rune('a'+n%6)), 63) + string(rune('0'+n/6)) }

func ask(db *Store, hwid string, p TrialParts, ip, email string, at time.Time) (Trial, string, error) {
	return db.StartTrialFor(TrialRequest{HWID: hwid, Parts: p, IP: ip, EmailHash: email, TrialDays: 7,
		IPEmailAfter: 3, IPWindow: month}, at)
}

func reason(err error) string {
	var refused *TrialRefused
	if errors.As(err, &refused) {
		return refused.Reason + ":" + refused.Matched
	}
	return ""
}

var (
	pcA        = parts("guid-1", "AAAA0001", "11112222333344445555666677778888")
	afterTrial = trialNow.Add(month)
)

func TestCheaterWhoseEveryPartMatchesAnEndedTrialIsRefused(t *testing.T) {
	db := open(t)
	ask(db, machineA, pcA, "1.1.1.1", "", trialNow)
	// A forged or recomputed machine ID with exactly the same hardware.
	_, _, err := ask(db, machineB, pcA, "2.2.2.2", "", afterTrial)
	if !errors.Is(err, ErrTrialUsed) || reason(err) != "hardware_match:guid+volume+firmware" {
		t.Fatalf("all-parts match not refused: %v (%s)", err, reason(err))
	}
}

func TestTwoOfThreeAsksForAnEmailOncePerAddress(t *testing.T) {
	db := open(t)
	ask(db, machineA, pcA, "1.1.1.1", "", trialNow)
	// Cloned lab VM (or a spoofed MachineGuid): two parts still match.
	clone := parts("guid-1", "AAAA0001", "99998888777766665555444433332222")
	_, _, err := ask(db, machineB, clone, "2.2.2.2", "", afterTrial)
	if !errors.Is(err, ErrEmailRequired) || reason(err) != "needs_email:guid+volume" {
		t.Fatalf("2 of 3 was not an email step: %v (%s)", err, reason(err))
	}
	if _, outcome, err := ask(db, machineB, clone, "2.2.2.2", "mail-1", afterTrial); err != nil || outcome != "new" {
		t.Fatalf("confirmed email did not start the trial: %v", err)
	}
	other := parts("guid-1", "AAAA0001", "55556666777788889999aaaabbbbcccc")
	if _, _, err := ask(db, machineC, other, "3.3.3.3", "mail-1", afterTrial.Add(month)); !errors.Is(err, ErrEmailUsed) {
		t.Fatalf("one address gave two trials: %v", err)
	}
}

func TestOneMatchingPartAloneNeverAsksAnything(t *testing.T) {
	db := open(t)
	ask(db, machineA, pcA, "1.1.1.1", "", trialNow)
	vm := parts("guid-2", "AAAA0001", "99998888777766665555444433332222")
	if _, outcome, err := ask(db, machineB, vm, "2.2.2.2", "", afterTrial); err != nil || outcome != "new" {
		t.Fatalf("one shared part had an effect: %v %s", err, outcome)
	}
}

func TestSameHardwareWithNewMachineIDKeepsTheRunningTrial(t *testing.T) {
	db := open(t)
	first, _, _ := ask(db, machineA, pcA, "1.1.1.1", "", trialNow)
	updated := parts("guid-1", "AAAA0001", "00000000000000000000000000000001")
	trial, outcome, err := ask(db, machineB, updated, "1.1.1.1", "", trialNow.Add(2*24*time.Hour))
	if err != nil || outcome != "continued" || trial.ExpiresAt != first.ExpiresAt || trial.HWID != machineB {
		t.Fatalf("running trial not continued: %+v %s %v", trial, outcome, err)
	}
}

func TestPlaceholderFirmwareUUIDsAreNeverCompared(t *testing.T) {
	for _, junk := range []string{"", strings.Repeat("0", 32), strings.Repeat("f", 32),
		"00020003000400050006000700080009", "03000200040005000006000700080009"} {
		db := open(t)
		ask(db, machineA, parts("guid-1", "AAAA0001", junk), "1.1.1.1", "", trialNow)
		// Another PC with the same placeholder UUID and (by coincidence) the same volume serial.
		if _, _, err := ask(db, machineB, parts("guid-2", "AAAA0001", junk), "2.2.2.2", "", afterTrial); err != nil {
			t.Fatalf("placeholder UUID %q counted as a match: %v", junk, err)
		}
	}
	if !IsJunkPart(certs.PartHash("firmware", SwapUUIDFields("03000200040005000006000700080009"))) {
		t.Fatal("raw byte order of the placeholder not recognised")
	}
}

func TestOfficeOfFourOnOneIPOnlyTypesAnEmailForTheFourth(t *testing.T) {
	db := open(t)
	office := "100.64.0.1"
	for n := 0; n < 3; n++ {
		if _, _, err := ask(db, machine(n), parts("g"+string(rune('0'+n)), "", ""), office, "", trialNow); err != nil {
			t.Fatalf("colleague %d refused: %v", n+1, err)
		}
	}
	fourth := parts("g4", "", "")
	if _, _, err := ask(db, machine(3), fourth, office, "", trialNow); reason(err) != "needs_email:busy_ip" {
		t.Fatalf("fourth trial from one IP: %v", err)
	}
	if _, _, err := ask(db, machine(3), fourth, office, "mail-4", trialNow); err != nil {
		t.Fatalf("fourth colleague with email refused: %v", err)
	}
	if _, outcome, _ := ask(db, machine(0), TrialParts{}, office, "", trialNow.Add(time.Hour)); outcome != "existing" {
		t.Fatal("a running trial was affected by the busy network")
	}
	if _, _, err := ask(db, machine(4), parts("g5", "", ""), office, "", trialNow.Add(month+time.Hour)); err != nil {
		t.Fatalf("network still counted after the window: %v", err)
	}
	if found, _ := db.ResetTrialIP(office); !found {
		t.Fatal("owner IP reset found nothing")
	}
}

func TestOldClientsWithoutPartsKeepTheMachineIDRule(t *testing.T) {
	db := open(t)
	if _, outcome, err := ask(db, machineA, TrialParts{}, "1.1.1.1", "", trialNow); err != nil || outcome != "new" {
		t.Fatal(err)
	}
	if _, outcome, _ := ask(db, machineA, TrialParts{}, "1.1.1.1", "", trialNow.Add(time.Hour)); outcome != "existing" {
		t.Fatal("reinstall did not return the same trial")
	}
	if _, _, err := ask(db, machineB, TrialParts{}, "2.2.2.2", "", afterTrial); err != nil {
		t.Fatal("another machine ID without parts refused")
	}
	if _, _, err := ask(db, machineA, TrialParts{}, "1.1.1.1", "", afterTrial); reason(err) != "same_machine:" {
		t.Fatalf("expired machine got a second trial: %v", err)
	}
}

func TestEmailCodeIsSingleUseLimitedAndBoundToTheAddress(t *testing.T) {
	db := open(t)
	expires := trialNow.Add(15 * time.Minute).Unix()
	db.SaveEmailCode(machineA, EmailCode{EmailHash: "mail-1", MAC: "right", Expires: expires})
	check := func(code string) func(string) bool { return func(mac string) bool { return mac == code } }
	if err := db.ConsumeEmailCode(machineA, "mail-2", check("right"), trialNow); !errors.Is(err, ErrBadCode) {
		t.Fatal("code accepted for another address")
	}
	for i := 0; i < 4; i++ {
		db.ConsumeEmailCode(machineA, "mail-1", check("wrong"), trialNow)
	}
	if db.ConsumeEmailCode(machineA, "mail-1", check("right"), trialNow) != nil {
		t.Fatal("right code refused after four wrong tries")
	}
	if !errors.Is(db.ConsumeEmailCode(machineA, "mail-1", check("right"), trialNow), ErrBadCode) {
		t.Fatal("code used twice")
	}
	db.SaveEmailCode(machineA, EmailCode{EmailHash: "mail-1", MAC: "right", Expires: expires})
	for i := 0; i < 5; i++ {
		db.ConsumeEmailCode(machineA, "mail-1", check("wrong"), trialNow)
	}
	if !errors.Is(db.ConsumeEmailCode(machineA, "mail-1", check("right"), trialNow), ErrBadCode) {
		t.Fatal("guessing beyond five tries allowed")
	}
	db.SaveEmailCode(machineA, EmailCode{EmailHash: "mail-1", Waived: true, Expires: expires})
	if db.ConsumeEmailCode(machineA, "mail-1", check("anything"), trialNow) != nil {
		t.Fatal("mail outage waiver not honoured")
	}
	db.SaveEmailCode(machineA, EmailCode{EmailHash: "mail-1", MAC: "right", Expires: expires})
	if !errors.Is(db.ConsumeEmailCode(machineA, "mail-1", check("right"), trialNow.Add(time.Hour)), ErrBadCode) {
		t.Fatal("expired code accepted")
	}
}

func TestOwnerOverridesClearHardwareAndAddresses(t *testing.T) {
	db := open(t)
	ask(db, machineA, pcA, "1.1.1.1", "", trialNow)
	if _, _, err := ask(db, machineB, pcA, "1.1.1.1", "", afterTrial); err == nil {
		t.Fatal("expected a hardware match")
	}
	// The customer sends the Machine ID shown in ScadaBench; the owner resets it.
	if found, err := db.ResetTrial(machineB); err != nil || !found {
		t.Fatalf("refused machine not found: %v", err)
	}
	if _, outcome, err := ask(db, machineB, pcA, "2.2.2.2", "", afterTrial); err != nil || outcome != "new" {
		t.Fatalf("customer still refused after owner reset: %v", err)
	}
	if known, err := db.ResetTrialParts(pcA); err != nil || known != 3 {
		t.Fatalf("parts reset: %d %v", known, err)
	}
	ask(db, machineC, parts("x", "", ""), "3.3.3.3", "mail-9", trialNow)
	if found, _ := db.AllowEmail("mail-9"); !found {
		t.Fatal("allow-email found nothing")
	}
	if used, _ := db.EmailUsed("mail-9"); used {
		t.Fatal("address still blocked after allow-email")
	}
}
