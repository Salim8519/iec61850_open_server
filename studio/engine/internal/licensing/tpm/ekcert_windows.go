//go:build windows

package tpm

import "golang.org/x/sys/windows/registry"

// ekStores are the registry stores where Windows keeps the endorsement key certificates it
// found in, or fetched for, this computer's TPM (RSA and ECC).
var ekStores = []string{
	`SYSTEM\CurrentControlSet\Services\TPM\WMI\Endorsement\EKCertStore\Certificates`,
	`SYSTEM\CurrentControlSet\Services\TPM\WMI\Endorsement\EKCertStoreECC\Certificates`,
}

// registryEKCertificate reads the certificate of the endorsement key ekPublic from the
// Windows store (read-only, no administrator rights). The Platform Crypto Provider does not
// hand these over to ordinary users on firmware TPMs, but the registry copy is readable.
func registryEKCertificate(ekPublic []byte) []byte {
	var blobs [][]byte
	for _, path := range ekStores {
		store, err := registry.OpenKey(registry.LOCAL_MACHINE, path,
			registry.ENUMERATE_SUB_KEYS|registry.WOW64_64KEY)
		if err != nil {
			continue
		}
		names, _ := store.ReadSubKeyNames(-1)
		store.Close()
		for _, name := range names {
			entry, err := registry.OpenKey(registry.LOCAL_MACHINE, path+`\`+name,
				registry.QUERY_VALUE|registry.WOW64_64KEY)
			if err != nil {
				continue
			}
			if blob, _, err := entry.GetBinaryValue("Blob"); err == nil {
				blobs = append(blobs, blob)
			}
			entry.Close()
		}
	}
	return matchingCertificate(blobs, ekPublic)
}
