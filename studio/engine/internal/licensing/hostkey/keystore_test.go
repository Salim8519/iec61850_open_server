package hostkey

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func store(t *testing.T, owner string) KeyStore {
	return KeyStore{Path: filepath.Join(t.TempDir(), KeyFile), Protector: FakeProtector{Owner: owner}}
}

func TestTheKeyPairIsMadeOnceAndKeptAcrossRestarts(t *testing.T) {
	k := store(t, "alice@PC1")
	if _, err := k.Load(); !errors.Is(err, ErrNoKey) {
		t.Fatalf("a PC without a key said %v", err)
	}
	first, err := k.LoadOrCreate()
	if err != nil {
		t.Fatal(err)
	}
	second, err := k.LoadOrCreate()
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("the key changed between two reads: %v", err)
	}
	loaded, err := k.Load()
	if err != nil || !bytes.Equal(first, loaded) {
		t.Fatalf("Load: %v", err)
	}
	text, err := k.PublicText()
	if err != nil || text != EncodeKey(first.Public().(ed25519.PublicKey)) {
		t.Fatalf("public text %q: %v", text, err)
	}
}

func TestACopiedKeyFileCannotBeOpenedByAnotherPCOrUser(t *testing.T) {
	original := store(t, "alice@PC1")
	key, err := original.LoadOrCreate()
	if err != nil {
		t.Fatal(err)
	}
	blob, _ := os.ReadFile(original.Path)
	stolen := store(t, "bob@PC2")
	if err := os.WriteFile(stolen.Path, blob, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := stolen.Load(); !errors.Is(err, ErrUnprotect) {
		t.Fatalf("another PC opened the blob: %v", err)
	}
	// Activating there makes a different pair (the file is replaced), never the stolen one.
	fresh, err := stolen.LoadOrCreate()
	if err != nil || bytes.Equal(fresh, key) {
		t.Fatalf("the stolen key was reused: %v", err)
	}
}

func TestADamagedOrEmptyKeyFileIsNeverAKey(t *testing.T) {
	k := store(t, "alice@PC1")
	for name, content := range map[string][]byte{"empty": {}, "junk": []byte("not a blob"),
		"short seed": append([]byte(fakeMagic+"alice@PC1|"), 1, 2, 3)} {
		if err := os.WriteFile(k.Path, content, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := k.Load(); err == nil {
			t.Errorf("%s was loaded as a key", name)
		}
	}
}

func TestWithoutAProtectorNoKeyIsEverMade(t *testing.T) {
	k := KeyStore{Path: filepath.Join(t.TempDir(), KeyFile)}
	if _, err := k.LoadOrCreate(); err == nil {
		t.Fatal("a key was made without a way to protect it")
	}
	if _, err := os.Stat(k.Path); err == nil {
		t.Fatal("a key file was written")
	}
}

func TestWindowsDPAPIRoundTripsAndRefusesAChangedBlob(t *testing.T) {
	protector := System()
	secret := bytes.Repeat([]byte{0x5a}, 32)
	blob, err := protector.Protect(secret)
	if err != nil {
		t.Skipf("DPAPI is not available here: %v", err)
	}
	if bytes.Contains(blob, secret) {
		t.Fatal("the DPAPI blob holds the secret in the clear")
	}
	back, err := protector.Unprotect(blob)
	if err != nil || !bytes.Equal(back, secret) {
		t.Fatalf("round trip: %v", err)
	}
	damaged := append([]byte(nil), blob...)
	damaged[len(damaged)/2] ^= 0xff
	if _, err := protector.Unprotect(damaged); !errors.Is(err, ErrUnprotect) {
		t.Fatalf("a damaged blob was opened: %v", err)
	}
	if _, err := protector.Unprotect([]byte("not a dpapi blob")); !errors.Is(err, ErrUnprotect) {
		t.Fatalf("junk was opened: %v", err)
	}
	k := KeyStore{Path: filepath.Join(t.TempDir(), KeyFile), Protector: protector}
	private, err := k.LoadOrCreate()
	if err != nil {
		t.Fatal(err)
	}
	if file, _ := os.ReadFile(k.Path); bytes.Contains(file, private.Seed()) {
		t.Fatal("host_key.dpapi holds the seed in the clear")
	}
	if loaded, err := k.Load(); err != nil || !bytes.Equal(loaded, private) {
		t.Fatalf("the real key file does not load back: %v", err)
	}
}
