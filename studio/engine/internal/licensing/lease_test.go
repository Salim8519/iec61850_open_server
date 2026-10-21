package licensing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/licensing/tpm"
	"studio-engine/internal/pyjson"
)

func bodyText(object *pyjson.Object) string {
	data, _ := pyjson.Flask(object)
	return string(data)
}

// leaseEnv is a computer without a usable chip, so its license is a 30-day lease.
func leaseEnv(t *testing.T) *bindEnv {
	env := newBindEnv(t, tpm.None{})
	env.activate()
	return env
}

func TestALeaseWorksOfflineForThirtyDaysThenAsksForTheInternet(t *testing.T) {
	env := leaseEnv(t)
	status := env.status()
	if !status.Licensed || status.Binding != BindingLease {
		t.Fatalf("status %+v", status)
	}
	env.server.offline = true
	env.advance(29.9)
	if status := env.status(); !status.Licensed || status.OnlineCheckRequired {
		t.Fatalf("day 29.9 offline: %+v", status)
	}
	env.advance(0.2) // day 30.1
	status = env.status()
	if status.Licensed || !status.OnlineCheckRequired || status.Reason != LeaseExpiredText || status.Binding != BindingLease {
		t.Fatalf("day 30.1 offline: %+v", status)
	}
	if LeaseExpiredText != "Connect this computer to the internet once so ScadaBench can confirm the license. "+
		"Computers with a security chip (TPM) do not need this." {
		t.Fatal("the text of the contract changed")
	}
	if _, err := env.service.CheckOnlineNow(); err == nil {
		t.Fatal("Check online now claimed success while the server is unreachable")
	}
	env.server.offline = false
	body, err := env.service.CheckOnlineNow()
	if err != nil {
		t.Fatalf("check online: %v", err)
	}
	if licensed, _ := body.Get("licensed"); licensed != true {
		t.Fatalf("one online check did not renew the lease: %s", bodyText(body))
	}
	if !env.status().Licensed {
		t.Fatal("not licensed after the check")
	}
}

func TestEveryValidAnswerRenewsTheLeaseAndPassesTheTokenOn(t *testing.T) {
	env := leaseEnv(t)
	first := env.service.lease.CurrentToken()
	if first == "" {
		t.Fatal("the activation did not store the lease token")
	}
	env.advance(7)
	verdict, err := env.service.VerifyOnlineIfDue(true)
	if err != nil || verdict != "valid" {
		t.Fatalf("background check: %v %v", verdict, err)
	}
	if last := env.server.calls[len(env.server.calls)-1]; last != "verify:"+first {
		t.Fatalf("the check sent %q, want the token from the activation", last)
	}
	second := env.service.lease.CurrentToken()
	if second == "" || second == first {
		t.Fatalf("the new token was not stored: %q -> %q", first, second)
	}
	env.server.offline = true
	env.advance(25) // day 32 after activation, 25 after the check
	if status := env.status(); !status.Licensed {
		t.Fatalf("a lease confirmed 25 days ago ended: %+v", status)
	}
	env.advance(6) // 31 days after the check
	if status := env.status(); status.Licensed || !status.OnlineCheckRequired {
		t.Fatalf("a lease unconfirmed for 31 days kept working: %+v", status)
	}
}

func TestTheBackgroundCheckIsQuietAndRareEnough(t *testing.T) {
	env := leaseEnv(t)
	before := len(env.server.calls)
	for i := 0; i < 5; i++ { // the loop wakes every 30 minutes
		env.advance(0.5 / 24)
		if verdict, err := env.service.VerifyOnlineIfDue(true); verdict != nil || err != nil {
			t.Fatalf("a check ran too soon: %v %v", verdict, err)
		}
	}
	if len(env.server.calls) != before {
		t.Fatalf("the server was asked within hours of the last answer: %v", env.server.calls[before:])
	}
	env.advance(0.25)
	if verdict, _ := env.service.VerifyOnlineIfDue(false); verdict != "valid" {
		t.Fatalf("no check after the interval: %v", verdict)
	}
	env.server.offline = true
	env.advance(1)
	if verdict, err := env.service.VerifyOnlineIfDue(true); verdict != nil || err != nil {
		t.Fatalf("an unreachable server was reported by the quiet check: %v %v", verdict, err)
	}
}

func TestOneDeletedCopyOfTheTokenChangesNothing(t *testing.T) {
	env := leaseEnv(t)
	env.advance(0.5)
	env.service.VerifyOnlineIfDue(true)
	token := env.service.lease.CurrentToken()
	os.Remove(filepath.Join(env.home, ".lease_token"))
	delete(env.registry, "LeaseToken")
	if got := env.service.lease.CurrentToken(); got != token {
		t.Fatalf("with two copies deleted the token is %q, want %q", got, token)
	}
	// A copy that missed a write (antivirus) holds an older token: the newest copy wins.
	env.service.lease.Token.Write("5:" + strings.Repeat("0", 32))
	env.service.lease.SetToken(1_760_999_999, token)
	os.WriteFile(filepath.Join(env.home, ".lease_token"), []byte("5:"+strings.Repeat("0", 32)), 0o666)
	if got := env.service.lease.CurrentToken(); got != token {
		t.Fatalf("an old copy beat the new token: %q", got)
	}
	for _, junk := range []string{"", "nonsense", "x:" + token, "5:short", "5:" + strings.Repeat("G", 32)} {
		env.service.lease.Token.Write(junk)
		if env.service.lease.CurrentToken() != "" {
			t.Fatalf("%q was accepted as a lease token", junk)
		}
	}
}

func TestEditingTheFilesCannotExtendALease(t *testing.T) {
	env := leaseEnv(t)
	env.advance(31)
	env.server.offline = true
	if env.status().Licensed {
		t.Fatal("the lease should have run out")
	}
	// A receipt that was never signed by the license server, or is for another license
	// or computer, or is not "valid", changes nothing.
	other := newBindServer(t, env.clock)
	forged := other.sign(map[string]any{"hwid": env.hwid, "lease_token": "", "license_id": bindKey, "nonce": "",
		"purpose": VerifyAnswerPurpose, "status": "valid", "time": int64(*env.clock) + 86400*365, "version": 1})
	env.service.lease.SetReceipt(Bundle{Payload: forged["payload"].(string), Signature: forged["signature"].(string)})
	genuine := env.server.sign(map[string]any{"hwid": env.hwid, "lease_token": "", "license_id": "SUBS-OTHR-OTHR-OTHR-OTHR",
		"nonce": "", "purpose": VerifyAnswerPurpose, "status": "valid", "time": int64(*env.clock), "version": 1})
	env.service.lease.SetReceipt(Bundle{Payload: genuine["payload"].(string), Signature: genuine["signature"].(string)})
	if status := env.status(); status.Licensed {
		t.Fatalf("a forged or foreign receipt renewed the lease: %+v", status)
	}
	for _, name := range []string{"valid-for-another-computer", "revoked"} {
		answer := map[string]any{"hwid": env.hwid, "lease_token": "", "license_id": bindKey, "nonce": "",
			"purpose": VerifyAnswerPurpose, "status": "valid", "time": int64(*env.clock), "version": 1}
		if name == "valid-for-another-computer" {
			answer["hwid"] = strings.Repeat("cd", 32)
		} else {
			answer["status"] = "revoked"
		}
		signed := env.server.sign(answer)
		env.service.lease.SetReceipt(Bundle{Payload: signed["payload"].(string), Signature: signed["signature"].(string)})
		if status := env.status(); status.Licensed {
			t.Fatalf("%s receipt renewed the lease: %+v", name, status)
		}
	}
	// Writing a future time into the dot-files does nothing either.
	os.WriteFile(filepath.Join(env.home, ".lease_answer"), []byte(`{"payload": "{}", "signature": "AAAA"}`), 0o666)
	env.registry["LeaseAnswer"] = "9999999999"
	if status := env.status(); status.Licensed {
		t.Fatalf("an edited receipt renewed the lease: %+v", status)
	}
}

func TestAServerTimeAheadOfTheClockIsARollbackNotARenewal(t *testing.T) {
	env := leaseEnv(t)
	env.advance(10)
	env.service.VerifyOnlineIfDue(true) // a signed answer stamped day 10
	env.server.offline = true
	env.advance(-5) // the clock is put back to day 5
	status := env.status()
	if status.Licensed || !strings.HasPrefix(status.Reason.(string), "System clock rollback detected") {
		t.Fatalf("rolling the clock back kept the lease: %+v", status)
	}
	os.Remove(filepath.Join(env.home, ".time_anchor")) // and the anchor deleted too: the signed time still guards
	delete(env.registry, "TimeAnchor")
	if status := env.status(); status.Licensed {
		t.Fatalf("deleting the anchor helped: %+v", status)
	}
}

func TestForgeryAndStaleAnswersAreTreatedLikeAnUnreachableServer(t *testing.T) {
	env := leaseEnv(t)
	env.advance(7)
	token := env.service.lease.CurrentToken()
	env.service = New(Options{Home: env.home, DataDir: env.home, Enforce: true, Keys: env.server.keys(),
		HWID: func() (string, error) { return env.hwid, nil }, Clock: func() float64 { return *env.clock },
		Debugger: func() bool { return false }, Logf: t.Logf, Registry: env.registry, Chip: env.chip,
		ClientFactory: func(string) Client { return liar{env.server} }})
	if verdict, err := env.service.VerifyOnlineIfDue(true); verdict != nil || err != nil {
		t.Fatalf("a replayed answer was believed: %v %v", verdict, err)
	}
	if env.service.lease.CurrentToken() != token || env.service.revoked.Contains(bindKey) {
		t.Fatal("a bad answer changed the lease state")
	}
}

// liar answers every verify with a genuine answer for some other nonce.
type liar struct{ server *bindServer }

func (l liar) StartTrial(h string, p *pyjson.Object, e, c any) (any, error) {
	return l.server.StartTrial(h, p, e, c)
}
func (l liar) RequestTrialCode(h, e string) (any, error) { return l.server.RequestTrialCode(h, e) }
func (l liar) Activate(k, h string) (any, error)         { return l.server.Activate(k, h) }
func (l liar) Verify(k any, h, _ string) (any, error)    { return l.server.Verify(k, h, "replayed-nonce") }
func (l liar) ActivateWithChip(k, h string, o *pyjson.Object) (any, error) {
	return l.server.ActivateWithChip(k, h, o)
}
func (l liar) ActivateChip(s, c string) (any, error) { return l.server.ActivateChip(s, c) }
func (l liar) VerifyLease(k any, h, _, t string) (any, error) {
	return l.server.VerifyLease(k, h, "replayed-nonce", t)
}
