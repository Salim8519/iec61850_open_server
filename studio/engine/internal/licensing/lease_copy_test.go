package licensing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/licensing/tpm"
)

// Copies, refunds and old servers: the lease cases of lease_test.go that involve a second party.

func TestAClonedCopyIsCaughtByItsStaleTokenAndTheOriginalSurvives(t *testing.T) {
	original := leaseEnv(t)
	clone := original.copyTo(tpm.None{}) // same files, same Machine ID, a second computer
	original.advance(1)
	if verdict, err := original.service.VerifyOnlineIfDue(true); err != nil || verdict != "valid" {
		t.Fatalf("original: %v %v", verdict, err)
	}
	clone.advance(0) // the clone checks later and still holds the token from before
	status := clone.status()
	if !status.Licensed {
		t.Fatalf("the copy was refused before it asked the server: %+v", status)
	}
	clone.advance(1)
	if _, err := clone.service.CheckOnlineNow(); err != nil {
		t.Fatalf("check on the copy: %v", err)
	}
	status = clone.status()
	if status.Licensed || status.Reason != MovedText {
		t.Fatalf("the copy kept its license: %+v", status)
	}
	if !clone.service.revoked.Contains(bindKey) {
		t.Fatal("the cancelled license was not remembered on the copy")
	}
	// Renaming the certificate back does not help: cancelled stays cancelled.
	entries, _ := os.ReadDir(clone.home)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "license.cert.moved-") {
			data, _ := os.ReadFile(filepath.Join(clone.home, entry.Name()))
			os.WriteFile(filepath.Join(clone.home, "license.cert"), data, 0o666)
		}
	}
	if status := clone.status(); status.Licensed {
		t.Fatalf("a renamed-back certificate licensed the copy again: %+v", status)
	}
	original.advance(1)
	if status := original.status(); !status.Licensed {
		t.Fatalf("the original lost its license: %+v", status)
	}
	if verdict, _ := original.service.VerifyOnlineIfDue(true); verdict != nil && verdict != "valid" {
		t.Fatalf("original check: %v", verdict)
	}
}

func TestARefundedLeaseLicenseStopsAtTheNextCheck(t *testing.T) {
	env := leaseEnv(t)
	env.server.licenses[bindKey].revoked = true
	env.advance(7)
	if verdict, err := env.service.VerifyOnlineIfDue(true); err != nil || verdict != "revoked" {
		t.Fatalf("%v %v", verdict, err)
	}
	if status := env.status(); status.Licensed || status.Reason != RevokedText {
		t.Fatalf("%+v", status)
	}
}

func TestAnActivationFromAServerWithoutLeaseTokensStillWorks(t *testing.T) {
	env := newBindEnv(t, tpm.None{})
	env.activate()
	os.Remove(filepath.Join(env.home, ".lease_token"))
	delete(env.registry, "LeaseToken")
	state := filepath.Join(env.home, "license_state.json")
	data, _ := os.ReadFile(state)
	os.WriteFile(state, []byte(strings.ReplaceAll(string(data), "lease_token", "other_field")), 0o666)
	if env.service.lease.CurrentToken() != "" {
		t.Fatal("the token survived")
	}
	// With no token the first check sends none, as a lease that has just been activated offline.
	env.server.licenses[bindKey].token = ""
	env.advance(7)
	if verdict, err := env.service.VerifyOnlineIfDue(true); err != nil || verdict != "valid" {
		t.Fatalf("%v %v", verdict, err)
	}
	if last := env.server.calls[len(env.server.calls)-1]; last != "verify:" {
		t.Fatalf("sent %q", last)
	}
}
