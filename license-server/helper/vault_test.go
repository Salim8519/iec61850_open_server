package main

import (
	"bytes"
	"errors"
	"testing"
)

var fastKDF = kdfParams{Time: 1, Memory: 1024, Threads: 1}

func TestHelperVaultOpensOnlyWithThePassword(t *testing.T) {
	secrets := Secrets{Seed: "PRIVATESEED", KeyID: "abcd1234", Created: "2026-10-17T00:00:00Z"}
	vault, err := sealSecrets(secrets, "correct horse battery staple", fastKDF)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(vault, []byte("PRIVATESEED")) || bytes.Contains(vault, []byte("abcd1234")) {
		t.Fatal("the vault contains plain secrets")
	}
	got, err := openSecrets(vault, "correct horse battery staple")
	if err != nil || got != secrets {
		t.Fatalf("open with the right password: %+v, %v", got, err)
	}
	if _, err := openSecrets(vault, "wrong horse battery staple"); !errors.Is(err, errWrongPassword) {
		t.Fatalf("wrong password gave %v", err)
	}
	tampered := bytes.Replace(vault, []byte(`"data":"`), []byte(`"data":"AA`), 1)
	if _, err := openSecrets(tampered, "correct horse battery staple"); err == nil {
		t.Fatal("an altered vault was accepted")
	}
	if _, err := sealSecrets(secrets, "sevench", fastKDF); err == nil {
		t.Fatal("a short password was accepted")
	}
}

func TestHelperVaultRidesOnTheProgramAndCanBeReplaced(t *testing.T) {
	program := []byte("MZ fake program bytes")
	if splitVault(program) != nil {
		t.Fatal("an unsealed program reported a vault")
	}
	first := appendVault(program, []byte(`{"v":1}`))
	if got := splitVault(first); string(got) != `{"v":1}` {
		t.Fatalf("splitVault = %q", got)
	}
	second := appendVault(first, []byte(`{"v":2}`))
	if !bytes.HasPrefix(second, program) || string(splitVault(second)) != `{"v":2}` ||
		len(second) != len(program)+len(`{"v":2}`)+8+len(vaultMagic) {
		t.Fatal("re-sealing did not replace the old vault")
	}
}

// BenchmarkHelperUnlockCost measures one password check at the real settings: this is
// also the cost of every single guess for anyone trying passwords offline.
func BenchmarkHelperUnlockCost(b *testing.B) {
	salt := make([]byte, 16)
	for range b.N {
		deriveKey("a long owner pass phrase", salt, defaultKDF)
	}
}
