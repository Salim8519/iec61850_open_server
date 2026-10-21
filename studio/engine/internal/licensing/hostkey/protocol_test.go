package hostkey

import (
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"
)

func pair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return public, private
}

var request = Request{Nonce: strings.Repeat("ab", 16), VMHWID: strings.Repeat("cd", 32)}

func TestTheSignedMessageIsExactlyTheContractsBytes(t *testing.T) {
	want := "scadabench-vm-proof-v1|" + request.Nonce + "|" + request.VMHWID
	if got := string(ProofMessage(request.Nonce, request.VMHWID)); got != want {
		t.Fatalf("message %q, want %q", got, want)
	}
}

func TestAProofVerifiesOnlyForItsNonceItsMachineAndItsKey(t *testing.T) {
	public, private := pair(t)
	signature := Sign(private, request)
	if strings.Contains(signature, "=") || !Verify(public, request, signature) {
		t.Fatalf("a genuine proof fails (or is padded): %q", signature)
	}
	other, _ := pair(t)
	cases := map[string]struct {
		key       ed25519.PublicKey
		request   Request
		signature string
	}{
		"another nonce":    {public, Request{Nonce: strings.Repeat("00", 16), VMHWID: request.VMHWID}, signature},
		"another machine":  {public, Request{Nonce: request.Nonce, VMHWID: strings.Repeat("ee", 32)}, signature},
		"another key":      {other, request, signature},
		"a damaged proof":  {public, request, signature[:len(signature)-2] + "AA"},
		"a short proof":    {public, request, signature[:20]},
		"a padded proof":   {public, request, signature + "="},
		"a proof and CRLF": {public, request, signature + "\r\n"},
		"no proof":         {public, request, ""},
		"a short key":      {public[:16], request, signature},
	}
	for name, c := range cases {
		if Verify(c.key, c.request, c.signature) {
			t.Errorf("%s was accepted", name)
		}
	}
}

func TestRequestsAreCheckedForShape(t *testing.T) {
	good := request
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := map[string]Request{
		"short nonce":     {Nonce: "ab", VMHWID: good.VMHWID},
		"upper case":      {Nonce: strings.Repeat("AB", 16), VMHWID: good.VMHWID},
		"not hex":         {Nonce: strings.Repeat("zz", 16), VMHWID: good.VMHWID},
		"short machine":   {Nonce: good.Nonce, VMHWID: strings.Repeat("cd", 31)},
		"machine upper":   {Nonce: good.Nonce, VMHWID: strings.Repeat("CD", 32)},
		"separator in id": {Nonce: good.Nonce, VMHWID: strings.Repeat("cd", 31) + "c|"},
		"empty":           {},
	}
	for name, r := range bad {
		if r.Validate() == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

func TestHostKeysAreStrictAndMaskedForDisplay(t *testing.T) {
	public, _ := pair(t)
	text := EncodeKey(public)
	if len(text) != 43 || strings.ContainsAny(text, "=+/") {
		t.Fatalf("a host key is 43 characters of base64url: %q", text)
	}
	if back, err := DecodeKey(text); err != nil || !back.Equal(public) {
		t.Fatalf("round trip: %v", err)
	}
	for name, bad := range map[string]string{"padded": text + "=", "newline": text + "\n", "short": text[:42],
		"long": text + "A", "tail bits": text[:42] + "B", "standard": strings.Repeat("+", 43), "empty": ""} {
		if _, err := DecodeKey(bad); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	if masked := MaskKey(text); strings.Contains(masked, text[6:len(text)-4]) || !strings.HasPrefix(masked, text[:6]) ||
		!strings.HasSuffix(masked, text[len(text)-4:]) {
		t.Fatalf("masked key %q", masked)
	}
	if MaskKey("short") != "" {
		t.Fatal("a text that is not a key must mask to nothing")
	}
}
