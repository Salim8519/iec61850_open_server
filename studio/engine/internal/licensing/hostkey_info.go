package licensing

import (
	"studio-engine/internal/licensing/hostkey"
)

// HostKeyInfo tells Settings whether this PC can let its virtual machines use the license, and if
// not, why in plain words.
type HostKeyInfo struct {
	Available bool
	Reason    string // set when not available
	Key       string // the masked host key when available
}

// NoHostKeyBuildText is shown by a build that keeps no host key (the parity build).
const NoHostKeyBuildText = "This copy of ScadaBench cannot share its license with virtual machines."

// HostResponder is the responder of this service's own license, with this service's seams.
func (s *Service) HostResponder() *HostResponder {
	return NewHostResponder(HostOptions{Home: s.options.Home, Keys: s.options.Keys, HWID: s.HWID,
		Protector: s.options.HostKey.Protector, Registry: s.options.Registry, Clock: s.options.Clock})
}

// HostKeyInfo applies the same rules the host helper applies before it answers.
func (s *Service) HostKeyInfo() HostKeyInfo {
	if !s.options.Enforce {
		return HostKeyInfo{Reason: DevelopmentText}
	}
	if s.options.HostKey.Protector == nil {
		return HostKeyInfo{Reason: NoHostKeyBuildText}
	}
	if current, err := s.Current(false); err == nil && current.Mode == ModeVMHost {
		return HostKeyInfo{Reason: HostIsVMText}
	}
	license, err := s.HostResponder().Check()
	if err != nil {
		return HostKeyInfo{Reason: err.Error()}
	}
	return HostKeyInfo{Available: true, Key: hostkey.MaskKey(license.public)}
}
