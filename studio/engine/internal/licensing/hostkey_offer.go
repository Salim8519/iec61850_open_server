package licensing

import (
	"path/filepath"

	"studio-engine/internal/licensing/hostkey"
)

// keyStore is this PC's host key file (host_key.dpapi) in the license home.
func (s *Service) keyStore() hostkey.KeyStore {
	return hostkey.KeyStore{Path: filepath.Join(s.options.Home, hostkey.KeyFile), Protector: s.options.HostKey.Protector}
}

// offerHostKey puts this PC's public host key into the next activation request, creating the key
// pair at the first activation. A client that is not the real HTTP client (a test fake) is left
// alone, and so is every build without a Protector: no key, no host_key field.
func (s *Service) offerHostKey(client Client) {
	if s.options.HostKey.Protector == nil {
		return
	}
	http, isHTTP := client.(*HTTPClient)
	if !isHTTP {
		return
	}
	public, err := s.keyStore().PublicText()
	if err != nil {
		s.options.Logf("No virtual machine key is offered: %v", err)
		return
	}
	http.HostKey = public
}

// ActivationCode is the one text an offline activation needs: "<machine ID>.<host key>" (a bare
// machine ID when this build keeps no host key). The key pair is made here when it does not exist.
func (s *Service) ActivationCode() (string, error) {
	if err := s.requireEnforced(); err != nil {
		return "", err
	}
	hwid, err := s.HWID()
	if err != nil {
		return "", err
	}
	if s.options.HostKey.Protector == nil {
		return hwid, nil
	}
	public, err := s.keyStore().PublicText()
	if err != nil {
		s.options.Logf("The activation code has no virtual machine key: %v", err)
		return hwid, nil
	}
	return hwid + "." + public, nil
}
