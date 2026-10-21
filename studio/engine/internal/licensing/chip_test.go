package licensing

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/licensing/tpm"
	"studio-engine/internal/pyjson"
)

func init() { chipRetryDelay = 0 } // tests do not wait for the chip's one retry

func isChipLicense(payload *pyjson.Object) bool {
	ak, _ := payload.Get("ak")
	return ak != nil && ak != ""
}

func TestALicenseTiedToTheChipNeedsNoServerAndWorksOfflineForever(t *testing.T) {
	chip := tpm.NewFake()
	env := newBindEnv(t, chip)
	env.activate()
	status := env.status()
	if !status.Licensed || status.Binding != BindingTPM || status.Plan != "lifetime" {
		t.Fatalf("status after activation: %+v", status)
	}
	if !isChipLicense(env.certificatePayload()) || len(env.server.calls) != 2 {
		t.Fatalf("calls %v: expected the chip challenge, then the answer", env.server.calls)
	}
	if blob, err := os.ReadFile(filepath.Join(env.home, "ak.blob")); err != nil || len(blob) == 0 {
		t.Fatalf("ak.blob: %v", err)
	}
	env.server.offline = true
	for _, days := range []float64{1, 40, 400, 3650} { // years later, never online
		env.advance(days)
		if status := env.status(); !status.Licensed || status.Binding != BindingTPM || status.OnlineCheckRequired {
			t.Fatalf("after %v days offline: %+v", days, status)
		}
	}
	if len(env.server.calls) != 2 {
		t.Fatalf("a chip license asked the server again while offline: %v", env.server.calls)
	}
}

func TestACopyOfAChipLicenseOnAnotherComputerCannotProveItself(t *testing.T) {
	env := newBindEnv(t, tpm.NewFake())
	env.activate()
	clone := env.copyTo(tpm.NewFake()) // same files and Machine ID, another chip
	status := clone.status()
	if status.Licensed || status.Reason != ChipFailedText || status.Binding != BindingTPM {
		t.Fatalf("the copy was licensed or told the wrong thing: %+v", status)
	}
	if !env.status().Licensed {
		t.Fatal("the original lost its license")
	}
	if _, err := clone.service.Activate(bindKey); err == nil {
		t.Fatal("the copy activated the same key again on its own chip")
	}
}

func TestAChipThatIsSwitchedOffNeverLicenses(t *testing.T) {
	env := newBindEnv(t, tpm.NewFake())
	env.activate()
	broken := env.copyTo(tpm.None{})
	if status := broken.status(); status.Licensed || status.Reason != ChipFailedText {
		t.Fatalf("a computer without a chip was licensed: %+v", status)
	}
}

func TestABusyChipDoesNotCutOffAWorkingLicenseForTenMinutes(t *testing.T) {
	chip := tpm.NewFake()
	env := newBindEnv(t, chip)
	env.activate()
	if !env.status().Licensed {
		t.Fatal("not licensed")
	}
	chip.ProveError = errors.New("the chip is busy")
	env.advance(5.0 / 1440) // five minutes
	if status := env.status(); !status.Licensed {
		t.Fatalf("a chip that worked five minutes ago and is busy now ended the license: %+v", status)
	}
	env.advance(10.0 / 1440) // fifteen minutes after the last proof
	if status := env.status(); status.Licensed || status.Reason != ChipFailedText {
		t.Fatalf("a chip that stays silent kept the license: %+v", status)
	}
	chip.ProveError = nil
	if !env.status().Licensed {
		t.Fatal("the license did not come back with the chip")
	}
}

func TestEveryProofUsesAFreshNonce(t *testing.T) {
	chip := tpm.NewFake()
	env := newBindEnv(t, chip)
	env.activate()
	env.status()
	before := chip.Proofs
	env.advance(2.0 / 1440) // past the one-minute reuse of a proof
	env.status()
	if chip.Proofs != before+1 {
		t.Fatalf("proofs %d -> %d: a status after the cache expired must ask the chip again", before, chip.Proofs)
	}
	env.status() // inside the reuse window
	if chip.Proofs != before+1 {
		t.Fatalf("a second status within a minute asked the chip again (%d proofs)", chip.Proofs)
	}
}

func TestWithoutACertificateOrAWorkingChallengeTheLicenseIsALease(t *testing.T) {
	cases := map[string]func(*tpm.Fake, *bindServer){
		"no endorsement key certificate": func(c *tpm.Fake, _ *bindServer) { c.EKCertDER = nil },
		"the chip cannot be opened":      func(c *tpm.Fake, _ *bindServer) { c.EnrollError = tpm.ErrNoChip },
		"the chip cannot open the challenge": func(c *tpm.Fake, _ *bindServer) {
			c.ActivateError = errors.New("0x80280095")
		},
		"the server cannot use the chip": func(_ *tpm.Fake, s *bindServer) { s.acceptChip = false },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			chip := tpm.NewFake()
			env := newBindEnv(t, chip)
			change(chip, env.server)
			env.activate()
			status := env.status()
			if !status.Licensed || status.Binding != BindingLease {
				t.Fatalf("status %+v", status)
			}
			if isChipLicense(env.certificatePayload()) {
				t.Fatal("a chip license was issued")
			}
			if _, err := os.Stat(filepath.Join(env.home, "ak.blob")); err == nil {
				t.Fatal("an attestation key was kept for a lease license")
			}
			if chip.Keys() != 0 {
				t.Fatalf("%d unused attestation keys were left in the chip", chip.Keys())
			}
		})
	}
}

func TestAnOldServerThatKnowsNoChipsGivesALease(t *testing.T) {
	env := newBindEnv(t, tpm.NewFake())
	env.service = New(Options{Home: env.home, DataDir: env.home, Enforce: true, Keys: env.server.keys(),
		HWID: func() (string, error) { return env.hwid, nil }, Clock: func() float64 { return *env.clock },
		Debugger: func() bool { return false }, Logf: t.Logf, Registry: env.registry, Chip: env.chip,
		ClientFactory: func(string) Client { return plainOnly{env.server} }})
	env.activate()
	if status := env.status(); !status.Licensed || status.Binding != BindingLease {
		t.Fatalf("status %+v", status)
	}
	if env.chip.(*tpm.Fake).Enrolls != 0 {
		t.Fatal("a chip key was made for a server that cannot use it")
	}
}

// plainOnly hides the BindingClient methods, like the license server before ADR-0098.
type plainOnly struct{ server *bindServer }

func (p plainOnly) StartTrial(h string, parts *pyjson.Object, e, c any) (any, error) {
	return p.server.StartTrial(h, parts, e, c)
}
func (p plainOnly) RequestTrialCode(h, e string) (any, error) { return p.server.RequestTrialCode(h, e) }
func (p plainOnly) Activate(k, h string) (any, error)         { return p.server.Activate(k, h) }
func (p plainOnly) Verify(k any, h, n string) (any, error)    { return p.server.Verify(k, h, n) }

func TestAKeyForAnActivationThatFailsIsNotLeftInTheChip(t *testing.T) {
	chip := tpm.NewFake()
	env := newBindEnv(t, chip)
	env.server.licenses[bindKey].hwid = "cd" + env.hwid[2:] // bound to another computer
	env.server.licenses[bindKey].ek = "someone else's chip"
	_, err := env.service.Activate(bindKey)
	var server *ServerError
	if !errors.As(err, &server) || server.Code != "bound_elsewhere" {
		t.Fatalf("error %v", err)
	}
	if chip.Keys() != 0 || len(chip.Gone) != 1 {
		t.Fatalf("keys %d, deleted %v", chip.Keys(), chip.Gone)
	}
	if _, statErr := os.Stat(filepath.Join(env.home, "license.cert")); statErr == nil {
		t.Fatal("a certificate was stored")
	}
}

func TestActivatingAgainAfterAReinstallBindsTheNewKeyAndForgetsTheOldOne(t *testing.T) {
	chip := tpm.NewFake()
	env := newBindEnv(t, chip)
	env.activate()
	old, _ := os.ReadFile(filepath.Join(env.home, "ak.blob"))
	env.activate() // same key, same chip: a new attestation key is allowed
	fresh, _ := os.ReadFile(filepath.Join(env.home, "ak.blob"))
	if string(old) == string(fresh) || len(chip.Gone) != 1 || chip.Gone[0] != string(old) {
		t.Fatalf("old %q new %q deleted %v", old, fresh, chip.Gone)
	}
	if status := env.status(); !status.Licensed || status.Binding != BindingTPM {
		t.Fatalf("status %+v", status)
	}
	if chip.Keys() != 1 {
		t.Fatalf("%d keys in the chip", chip.Keys())
	}
}

func TestTheStatusNamesTheBindingOnlyForChipAndLeaseLicenses(t *testing.T) {
	env := newBindEnv(t, tpm.NewFake())
	env.activate()
	body, err := env.service.StatusBody("1.0")
	if err != nil || !strings.Contains(string(body), `"binding":"tpm"`) {
		t.Fatalf("%s %v", body, err)
	}
	lease := newBindEnv(t, tpm.None{})
	lease.activate()
	body, _ = lease.service.StatusBody("1.0")
	if !strings.Contains(string(body), `"binding":"lease"`) {
		t.Fatalf("%s", body)
	}
	plain := (Status{Licensed: true, Mode: "lifetime", Plan: "lifetime"}).Object()
	if _, found := plain.Get("binding"); found {
		t.Fatal("a certificate without a binding got a binding field")
	}
}
