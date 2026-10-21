package tpm

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rsa"
	_ "crypto/sha1"   // the AK may sign quotes with SHA-1 (Windows Platform Crypto Provider)
	_ "crypto/sha256" // or SHA-256
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/google/go-attestation/attest"
	"github.com/google/go-tpm/legacy/tpm2"
)

// A proof is what the chip answers to Prove: the quote (TPMS_ATTEST the chip generated,
// containing the nonce as its qualifying data) and the AK's signature over it. Wire layout,
// private to this package: uint32 big-endian quote length, the quote, the signature.
const maxProofPart = 1 << 16

func packProof(quote, signature []byte) []byte {
	out := make([]byte, 4, 4+len(quote)+len(signature))
	binary.BigEndian.PutUint32(out, uint32(len(quote)))
	return append(append(out, quote...), signature...)
}

func unpackProof(proof []byte) (quote, signature []byte, err error) {
	if len(proof) < 4 {
		return nil, nil, errors.New("the proof is too short")
	}
	size := int(binary.BigEndian.Uint32(proof))
	if size <= 0 || size > maxProofPart || len(proof)-4-size <= 0 || len(proof)-4-size > maxProofPart {
		return nil, nil, errors.New("the proof is not well formed")
	}
	return proof[4 : 4+size], proof[4+size:], nil
}

// VerifyProof checks, without any hardware, that proof is a genuine quote of nonce signed by
// the attestation key akPublic (a TPMT_PUBLIC): the signature verifies, the signed structure
// is a TPM-generated quote and its qualifying data equals nonce. The PCR values the quote
// also covers are not compared: only possession of the chip's key matters here.
func VerifyProof(akPublic, nonce, proof []byte) error {
	if len(nonce) == 0 {
		return errors.New("no nonce")
	}
	key, err := attest.ParseAKPublic(akPublic)
	if err != nil {
		return fmt.Errorf("the attestation key is unreadable: %w", err)
	}
	quote, rawSignature, err := unpackProof(proof)
	if err != nil {
		return err
	}
	signature, err := tpm2.DecodeSignature(bytes.NewBuffer(rawSignature))
	if err != nil {
		return fmt.Errorf("the signature is unreadable: %w", err)
	}
	hash := key.Hash.New()
	hash.Write(quote)
	digest := hash.Sum(nil)
	switch public := key.Public.(type) {
	case *rsa.PublicKey:
		if signature.RSA == nil {
			return errors.New("the signature does not fit an RSA key")
		}
		if signature.Alg == tpm2.AlgRSAPSS {
			err = rsa.VerifyPSS(public, key.Hash, digest, signature.RSA.Signature, nil)
		} else {
			err = rsa.VerifyPKCS1v15(public, key.Hash, digest, signature.RSA.Signature)
		}
		if err != nil {
			return fmt.Errorf("the signature is not valid: %w", err)
		}
	case *ecdsa.PublicKey:
		if signature.ECC == nil || !ecdsa.Verify(public, digest, signature.ECC.R, signature.ECC.S) {
			return errors.New("the signature is not valid")
		}
	default:
		return fmt.Errorf("unsupported attestation key type %T", public)
	}
	attested, err := tpm2.DecodeAttestationData(quote)
	if err != nil {
		return fmt.Errorf("the quote is unreadable: %w", err)
	}
	if attested.Type != tpm2.TagAttestQuote {
		return errors.New("the signed structure is not a quote")
	}
	if !bytes.Equal(attested.ExtraData, nonce) {
		return errors.New("the quote answers a different nonce")
	}
	return nil
}
