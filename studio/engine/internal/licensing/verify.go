package licensing

import (
	"crypto/ed25519"
	"math/big"
)

// fieldPrime is p = 2^255 - 19.
var fieldPrime = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(19))

// canonicalPoint reports whether a 32-byte point encoding is one ed25519_verify.py's
// _decompress accepts as far as the encoding goes: y below p, and no sign bit on x = 0
// (y = ±1). crypto/ed25519 alone would also accept those non-canonical public keys.
func canonicalPoint(encoded []byte) bool {
	little := make([]byte, 32)
	for index := range little {
		little[index] = encoded[31-index]
	}
	sign := little[0] >> 7
	little[0] &= 0x7f
	y := new(big.Int).SetBytes(little)
	if y.Cmp(fieldPrime) >= 0 {
		return false
	}
	if sign == 1 {
		pMinusOne := new(big.Int).Sub(fieldPrime, big.NewInt(1))
		if y.Cmp(big.NewInt(1)) == 0 || y.Cmp(pMinusOne) == 0 {
			return false
		}
	}
	return true
}

// Verify returns true only for a valid Ed25519 signature of message under publicKey, with
// the same verdicts as ed25519_verify.verify (RFC 8032 section 6, cofactorless): wrong
// lengths, invalid or non-canonical points and S >= L are refused.
func Verify(publicKey, message, signature []byte) bool {
	if len(publicKey) != ed25519.PublicKeySize || len(signature) != ed25519.SignatureSize {
		return false
	}
	if !canonicalPoint(publicKey) {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(publicKey), message, signature)
}
