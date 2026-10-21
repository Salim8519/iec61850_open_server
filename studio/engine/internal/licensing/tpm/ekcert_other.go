//go:build !windows

package tpm

// registryEKCertificate finds nothing outside Windows.
func registryEKCertificate([]byte) []byte { return nil }
