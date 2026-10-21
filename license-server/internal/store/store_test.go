package store

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var (
	machineA = strings.Repeat("a", 64)
	machineB = strings.Repeat("b", 64)
	machineC = strings.Repeat("c", 64)
)

func open(t *testing.T) *Store {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "licenses.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestOneTrialPerMachineEver(t *testing.T) {
	db := open(t)
	now := time.Unix(1_760_000_000, 0)
	first, err := db.StartTrial(machineA, "1.1.1.1", now, 7)
	if err != nil || first.ExpiresAt != now.Add(7*24*time.Hour).Unix() {
		t.Fatalf("first trial: %+v %v", first, err)
	}
	again, err := db.StartTrial(machineA, "2.2.2.2", now.Add(24*time.Hour), 7)
	if err != nil || again.IssuedAt != first.IssuedAt {
		t.Fatal("reinstall during trial must return the same trial, not a new one")
	}
	if _, err := db.StartTrial(machineA, "1.1.1.1", now.Add(8*24*time.Hour), 7); !errors.Is(err, ErrTrialUsed) {
		t.Fatalf("expired trial restarted: %v", err)
	}
	if db.TrialCount() != 1 {
		t.Fatal("trial count wrong")
	}
}

func TestLifetimeKeyStaysOnItsFirstPCForGood(t *testing.T) {
	db := open(t)
	now := time.Unix(1_760_000_000, 0)
	key := "SUBS-AAAA-BBBB-CCCC-DDDD"
	db.CreateLicense(License{Key: key, Type: "lifetime"})
	if _, err := db.Activate(key, machineA, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Activate(key, machineA, now); err != nil {
		t.Fatal("same machine must be able to re-activate")
	}
	for _, other := range []string{machineB, machineC} {
		_, err := db.Activate(key, other, now.Add(400*24*time.Hour))
		if !errors.Is(err, ErrBoundElsewhere) {
			t.Fatalf("another machine activated: %v", err)
		}
		if err.Error() != "This license key is already used on another computer. Each license works on one computer only." {
			t.Fatalf("customer text changed: %q", err)
		}
	}
	if license, _ := db.GetLicense(key); license.HWID != machineA {
		t.Fatal("the key left its first machine")
	}
	db.Revoke(key, "refunded")
	if _, err := db.Activate(key, machineA, now); !errors.Is(err, ErrRevoked) {
		t.Fatal("revoked key activated")
	}
	if _, err := db.Activate("SUBS-NOPE-NOPE-NOPE-NOPE", machineA, now); !errors.Is(err, ErrUnknownKey) {
		t.Fatal("unknown key accepted")
	}
}

func TestOldRecordsWithMovesStillReadAndStayBound(t *testing.T) {
	db := open(t)
	now := time.Unix(1_760_000_000, 0)
	key := "SUBS-AAAA-BBBB-CCCC-EEEE"
	db.CreateLicense(License{Key: key, Type: "lifetime", HWID: machineA, ActivatedAt: 5, Transfers: []int64{1, 2}})
	license, err := db.GetLicense(key)
	if err != nil || len(license.Transfers) != 2 || license.HWID != machineA {
		t.Fatalf("old record unreadable: %+v %v", license, err)
	}
	if _, err := db.Activate(key, machineB, now); !errors.Is(err, ErrBoundElsewhere) {
		t.Fatalf("old moved key bound to another machine: %v", err)
	}
	if after, _ := db.GetLicense(key); after.HWID != machineA || len(after.Transfers) != 2 || after.ActivatedAt != 5 {
		t.Fatalf("record changed: %+v", after)
	}
}

func TestIssueForOrderIsIdempotent(t *testing.T) {
	db := open(t)
	now := time.Unix(1_760_000_000, 0)
	calls := 0
	keyFactory := func() string { calls++; return "SUBS-ORDR-0000-0000-000" + string(rune('0'+calls)) }
	first, created, err := db.IssueForOrder(Order{OrderID: "O-1", CaptureID: "C-1", Email: "a@b.c"}, keyFactory, now)
	if err != nil || !created {
		t.Fatal("first fulfilment failed")
	}
	second, created, err := db.IssueForOrder(Order{OrderID: "O-1", CaptureID: "C-1"}, keyFactory, now)
	if err != nil || created || second.LicenseKey != first.LicenseKey || calls != 1 {
		t.Fatal("second fulfilment created another key")
	}
	found, ok, _ := db.FindOrderByCapture("C-1")
	if !ok || found.LicenseKey != first.LicenseKey {
		t.Fatal("order not found by capture id")
	}
}

func TestOwnerCanResetTrialAndDeleteKey(t *testing.T) {
	db := open(t)
	now := time.Unix(1_760_000_000, 0)
	db.StartTrial(machineA, "ip", now, 7)
	if found, err := db.ResetTrial(machineA); err != nil || !found {
		t.Fatal("trial not reset")
	}
	if _, err := db.StartTrial(machineA, "ip", now.Add(30*24*time.Hour), 7); err != nil {
		t.Fatal("trial not available again after reset")
	}
	db.CreateLicense(License{Key: "SUBS-DELE-TEME-NOWW-AAAA", Type: "lifetime"})
	if err := db.DeleteLicense("SUBS-DELE-TEME-NOWW-AAAA"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.GetLicense("SUBS-DELE-TEME-NOWW-AAAA"); !errors.Is(err, ErrUnknownKey) {
		t.Fatal("deleted key still present")
	}
}

func TestDatabaseRefusesSecondOpener(t *testing.T) {
	path := filepath.Join(t.TempDir(), "licenses.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := Open(path); !errors.Is(err, ErrDatabaseBusy) {
		t.Fatalf("second opener got %v", err)
	}
}
