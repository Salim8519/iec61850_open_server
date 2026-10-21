package certs

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestHostKeyIsAlwaysOneStrictSpelling(t *testing.T) {
	good := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	if len(good) != 43 || !ValidHostKey(good) {
		t.Fatalf("a 32-byte key in base64url is refused: %q", good)
	}
	for name, text := range map[string]string{"empty": "", "padded": good + "=", "short": good[:40],
		"long": good + "AAAA", "standard alphabet": strings.Repeat("+", 43), "tail bits": good[:42] + "B",
		"newline": good + "\n", "16 bytes": base64.RawURLEncoding.EncodeToString(make([]byte, 16))} {
		if ValidHostKey(text) {
			t.Errorf("%s was accepted: %q", name, text)
		}
	}
}

func TestPayloadPutsHostKeyBetweenExpiresAtAndHWIDAndOmitsItWhenThereIsNone(t *testing.T) {
	signer, _ := Generate(t.TempDir(), false)
	key := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	with, _ := signer.Sign(Payload{HostKey: key, HWID: "h", Type: "lifetime"})
	want := `{"email":"","expires_at":0,"host_key":"` + key + `","hwid":"h","issued_at":0,"kid":"` + signer.KeyID +
		`","license_id":"","max_machines":0,"type":"lifetime","version":1}`
	if with.Payload != want {
		t.Fatalf("payload with a host key:\n%s\n%s", with.Payload, want)
	}
	without, _ := signer.Sign(Payload{HWID: "h", Type: "lifetime"})
	if strings.Contains(without.Payload, "host_key") {
		t.Fatalf("a payload without a host key names one: %s", without.Payload)
	}
	if payload, err := Verify(signer.Public(), with); err != nil || payload.HostKey != key {
		t.Fatalf("verify: %v %+v", err, payload)
	}
}
