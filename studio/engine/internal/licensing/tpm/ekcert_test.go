package tpm

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"math/big"
	"testing"
	"time"
)

func testCertificate(t *testing.T, key *ecdsa.PrivateKey) []byte {
	t.Helper()
	template := &x509.Certificate{SerialNumber: big.NewInt(7), Subject: pkix.Name{CommonName: "fake EK"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return der
}

// element is a serialized Windows certificate element: hash and flags properties, then the
// certificate (property 32), each "id, 1, length, data" little-endian.
func element(der []byte) []byte {
	var out []byte
	add := func(id uint32, data []byte) {
		head := make([]byte, 12)
		binary.LittleEndian.PutUint32(head, id)
		binary.LittleEndian.PutUint32(head[4:], 1)
		binary.LittleEndian.PutUint32(head[8:], uint32(len(data)))
		out = append(append(out, head...), data...)
	}
	add(0x5C, []byte{0, 8, 0, 0})
	add(3, make([]byte, 20))
	add(certPropertyID, der)
	return out
}

func TestCertificateIsFoundInAWindowsElementOrBare(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	der := testCertificate(t, key)
	for name, blob := range map[string][]byte{"bare": der, "element": element(der)} {
		got, ok := certificateFromElement(blob)
		if !ok || string(got) != string(der) {
			t.Errorf("%s: certificate not found", name)
		}
	}
	for _, blob := range [][]byte{nil, []byte("nonsense"), element(der)[:30], append(element(der)[:24], 0xff, 0xff, 0xff, 0x7f)} {
		if _, ok := certificateFromElement(blob); ok {
			t.Errorf("blob %v gave a certificate", blob)
		}
	}
}

func TestOnlyTheCertificateOfThisEndorsementKeyIsUsed(t *testing.T) {
	mine, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	theirs, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	public, _ := x509.MarshalPKIXPublicKey(&mine.PublicKey)
	mineDER, theirDER := testCertificate(t, mine), testCertificate(t, theirs)
	got := matchingCertificate([][]byte{element(theirDER), []byte("junk"), element(mineDER)}, public)
	if string(got) != string(mineDER) {
		t.Fatal("the certificate of this chip's key was not chosen")
	}
	if matchingCertificate([][]byte{element(theirDER)}, public) != nil {
		t.Fatal("a certificate of another key was used")
	}
}
