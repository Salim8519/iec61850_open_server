package tpm

import (
	"crypto/x509"
	"encoding/binary"
)

// certPropertyID is CERT_CERT_PROP_ID: in a serialized Windows certificate element this
// property holds the certificate's DER bytes.
const certPropertyID = 32

// certificateFromElement returns the DER certificate inside blob, which is either the bare
// DER certificate or a serialized Windows certificate element (a list of properties, each
// "id uint32, 1 uint32, length uint32, data", little-endian). ok is false for anything
// else. Windows keeps the endorsement key certificates of firmware TPMs (AMD, Intel) in
// such elements under HKLM\SYSTEM\CurrentControlSet\Services\TPM\WMI\Endorsement.
func certificateFromElement(blob []byte) (der []byte, ok bool) {
	if certificate, err := x509.ParseCertificate(blob); err == nil {
		return certificate.Raw, true
	}
	for position := 0; position+12 <= len(blob); {
		identifier := binary.LittleEndian.Uint32(blob[position:])
		length := int(binary.LittleEndian.Uint32(blob[position+8:]))
		position += 12
		if length < 0 || position+length > len(blob) {
			return nil, false
		}
		if identifier == certPropertyID {
			if certificate, err := x509.ParseCertificate(blob[position : position+length]); err == nil {
				return certificate.Raw, true
			}
			return nil, false
		}
		position += length
	}
	return nil, false
}

// matchingCertificate returns the first certificate of blobs whose public key is the
// endorsement key public (PKIX DER); nil when none matches.
func matchingCertificate(blobs [][]byte, ekPublic []byte) []byte {
	for _, blob := range blobs {
		der, ok := certificateFromElement(blob)
		if !ok {
			continue
		}
		certificate, err := x509.ParseCertificate(der)
		if err != nil {
			continue
		}
		public, err := x509.MarshalPKIXPublicKey(certificate.PublicKey)
		if err == nil && string(public) == string(ekPublic) {
			return der
		}
	}
	return nil
}
