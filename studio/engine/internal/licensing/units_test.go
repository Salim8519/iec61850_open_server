package licensing

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"

	"studio-engine/internal/pyjson"
)

func TestReleaseKeysMatchPython(t *testing.T) {
	v := loadUnits(t)
	keys := PublicKeys()
	if len(keys) != len(v.ReleaseKeys) || DefaultServerURL != v.DefaultURL || !Configured() {
		t.Fatalf("keys %d vs %d, url %q", len(keys), len(v.ReleaseKeys), v.DefaultURL)
	}
	for kid, key := range v.ReleaseKeys {
		if hex.EncodeToString(keys[kid]) != key {
			t.Errorf("release key %s differs", kid)
		}
	}
	if KeyID(fromHex(t, v.TestKey.Public)) != v.TestKey.Kid {
		t.Error("key id of the test key differs")
	}
}

func hexOrNil(data []byte, err error) *string {
	if err != nil {
		return nil
	}
	text := hex.EncodeToString(data)
	return &text
}

func sameText(a, b *string) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }

func TestBase64MatchesPython(t *testing.T) {
	for _, c := range loadUnits(t).Base64 {
		signature, err := decodeSignature(c.Text)
		if got := hexOrNil(signature, err); !sameText(got, c.Strip) {
			t.Errorf("_decode_signature(%q) = %v, want %v", c.Text, got, c.Strip)
		}
		if got := hexOrNil(URLSafeB64Decode(PadBase64(c.Text))); !sameText(got, c.Pad) {
			t.Errorf("urlsafe_b64decode(%q + pad) = %v, want %v", c.Text, got, c.Pad)
		}
	}
}

func TestEd25519MatchesPython(t *testing.T) {
	for index, c := range loadUnits(t).Ed25519 {
		got := Verify(fromHex(t, c.Public), fromHex(t, c.Message), fromHex(t, c.Signature))
		if got != c.OK {
			t.Errorf("case %d: Verify = %v, want %v", index, got, c.OK)
		}
	}
}

func keysNamed(t *testing.T, name string) map[string][]byte {
	switch name {
	case "test":
		return testKeys(t)
	case "none":
		return map[string][]byte{}
	}
	return nil
}

func TestCertificatesMatchPython(t *testing.T) {
	for _, c := range loadUnits(t).Certificates {
		payload, err := func() (*pyjson.Object, error) {
			bundle, err := ParseBundle(c.Text)
			if err != nil {
				return nil, err
			}
			return VerifyBundle(bundle, keysNamed(t, c.Keys))
		}()
		want, ok := c.okString()
		var certificate *CertificateError
		var crash *UncaughtError
		switch {
		case ok && (err != nil || compactText(t, payload) != want):
			t.Errorf("%q (%s): got %v %v, want %s", c.Text, c.Keys, payload, err, want)
		case c.Error == "CertificateError" && (!errors.As(err, &certificate) || certificate.Message != c.Message):
			t.Errorf("%q (%s): got %v, want CertificateError %q", c.Text, c.Keys, err, c.Message)
		case c.Error != "" && c.Error != "CertificateError" && !errors.As(err, &crash):
			t.Errorf("%q (%s): got %v, want uncaught %s", c.Text, c.Keys, err, c.Error)
		}
	}
}

func TestPolicyMatchesPython(t *testing.T) {
	for _, c := range loadUnits(t).Policy {
		var payload *pyjson.Object
		if c.Payload != nil {
			payload = mustDecode(t, *c.Payload).(*pyjson.Object)
		}
		status, err := EvaluateCertificate(payload, "ab"+repeat("ab", 31), c.Now, c.MaxSeen, c.Dev)
		want, ok := c.okString()
		if ok && (err != nil || compactText(t, status.Object()) != want) {
			t.Errorf("%v now=%d seen=%d: got %s %v\nwant %s", c.Payload, c.Now, c.MaxSeen,
				compactText(t, status.Object()), err, want)
		}
		if !ok && err == nil {
			t.Errorf("%v: want %s, got %s", c.Payload, c.Error, compactText(t, status.Object()))
		}
	}
}

func repeat(text string, count int) string {
	out := ""
	for range count {
		out += text
	}
	return out
}

func TestKeysAndMasksMatchPython(t *testing.T) {
	v := loadUnits(t)
	for _, c := range v.NormalizeKey {
		key, err := NormalizeKey(mustDecode(t, c.Input))
		want, ok := c.okString()
		var action *ActionError
		if ok && (err != nil || key != want) {
			t.Errorf("normalize_key(%s) = %q %v, want %q", c.Input, key, err, want)
		}
		if !ok && (!errors.As(err, &action) || action.Message != c.Message || action.Code != c.Code) {
			t.Errorf("normalize_key(%s) error %v, want %q %v", c.Input, err, c.Message, c.Code)
		}
	}
	for _, c := range v.MaskKey {
		if got := compactText(t, MaskKey(mustDecode(t, c.Input))); got != c.Output {
			t.Errorf("mask_key(%s) = %s, want %s", c.Input, got, c.Output)
		}
	}
}

func TestMachineIDRulesMatchPython(t *testing.T) {
	v := loadUnits(t)
	for _, c := range v.Compose {
		parts := MachineParts{MachineGUID: c.GUID, VolumeSerial: c.Serial, FirmwareUUID: c.UUID}
		if Compose(parts) != c.V2 || ComposeLegacy(parts) != c.V1 {
			t.Errorf("compose(%q) differs", c.GUID)
		}
	}
	for _, c := range v.BoundHWID {
		legacy := ""
		if c.Legacy != nil {
			legacy = *c.Legacy
		}
		got, _ := BoundHWID(mustDecode(t, c.Cert), mustDecode(t, c.Issued), c.Current, legacy)
		want, ok := c.okString()
		if !ok {
			want = c.Current // Python's _machine_for falls back to the current ID
		}
		if got != want {
			t.Errorf("bound_hwid(%s, %s) = %q, want %q", c.Cert, c.Issued, got, want)
		}
	}
	for _, c := range v.SMBIOS {
		if got := SMBIOSSystemUUID(fromHex(t, c.Raw)); got != c.UUID {
			t.Errorf("smbios %s = %q, want %q", c.Raw, got, c.UUID)
		}
	}
}

func TestSolveMatchesPython(t *testing.T) {
	for _, c := range loadUnits(t).Solve {
		counter, err := Solve(c.Nonce, c.Difficulty)
		var want int64
		if c.OK != nil {
			if json.Unmarshal(c.OK, &want) != nil || err != nil || counter != want {
				t.Errorf("solve(%q, %d) = %d %v, want %d", c.Nonce, c.Difficulty, counter, err, want)
			}
			continue
		}
		var server *ServerError
		if !errors.As(err, &server) || server.Message != c.Message || server.Code != c.Code {
			t.Errorf("solve(%q, %d) error %v, want %q", c.Nonce, c.Difficulty, err, c.Message)
		}
	}
}
