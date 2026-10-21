package tpm

import (
	"bytes"
	"errors"
)

// The blob kept as ak.blob wraps the key's storage name so the key can be deleted again:
// "SBAK1", one length byte, the key name, then go-attestation's own blob.
const blobMagic = "SBAK1"

func wrapBlob(name string, inner []byte) []byte {
	out := append([]byte(blobMagic), byte(len(name)))
	return append(append(out, name...), inner...)
}

func unwrapBlob(blob []byte) (name string, inner []byte, err error) {
	if !bytes.HasPrefix(blob, []byte(blobMagic)) || len(blob) < len(blobMagic)+1 {
		return "", nil, errors.New("the stored attestation key is not in a known format")
	}
	size := int(blob[len(blobMagic)])
	rest := blob[len(blobMagic)+1:]
	if size == 0 || len(rest) <= size {
		return "", nil, errors.New("the stored attestation key is damaged")
	}
	return string(rest[:size]), rest[size:], nil
}
