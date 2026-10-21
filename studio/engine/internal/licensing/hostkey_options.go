package licensing

import (
	"time"

	"studio-engine/internal/licensing/hostkey"
	"studio-engine/internal/pyjson"
)

// HostKeyOptions are the seams of the VM host key (ADR-0099). The zero value switches the feature
// off completely: no key is made or offered and a virtual machine never asks a host. Only the app
// composition (wire_platform.go) fills it in; the parity build and most tests leave it empty.
type HostKeyOptions struct {
	// Protector keeps the private host key (Windows DPAPI in the app, a fake in tests).
	Protector hostkey.Protector
	// VM is the virtual machine side: how a computer without a license of its own finds and asks
	// the host it runs on. nil means this computer never asks.
	VM *VMOptions
}

// VMOptions is how a computer looks for its host. Zero values mean the defaults of the app.
type VMOptions struct {
	// Hosts returns the addresses to ask ("ip" or "ip:port"); nil means the default gateway and
	// the ".1" of every local subnet (hostkey.CandidateHosts).
	Hosts func() []string
	// Dial opens a connection; nil means a plain TCP dial.
	Dial hostkey.Dialer
	// Timeout bounds one search (default 2.5 s).
	Timeout time.Duration
	// RetryAfter is the pause before a failed search starts again (default 30 s).
	RetryAfter time.Duration
}

const (
	vmDefaultTimeout    = 2500 * time.Millisecond
	vmDefaultRetryAfter = 30 * time.Second
	// vmProofMaxAge is how long a host's proof licenses a virtual machine; the native guard uses the same.
	vmProofMaxAge = 24 * 3600
	// vmRefreshAfter is the age at which a proof is renewed in the background.
	vmRefreshAfter = 6 * 3600
)

// Texts of the VM host key (docs/specs/licensing.md LIC-70 to LIC-79). Plain words, no jargon.
const (
	HostNoCertificateText = "This PC has no license to share. Activate a lifetime license first."
	HostNotLifetimeText   = "Only a lifetime license can be shared with virtual machines."
	HostNoKeyInCertText   = "This license was activated before virtual machine sharing existed. " +
		"Enter your license key again on this PC to turn it on."
	HostKeyUnusableText = "The virtual machine key of this license cannot be opened on this PC. " +
		"Enter your license key again to make a new one."
	HostKeyMismatchText = "The virtual machine key on this PC does not belong to this license. " +
		"Enter your license key again to make a new one."
	HostIsVMText = "This computer is a virtual machine that uses its host's license. " +
		"Share the license from the host PC."
)

// certHostKey is the optional "host_key" of a certificate: base64url of a 32-byte public key. A
// value that is not exactly that counts as absent.
func certHostKey(payload *pyjson.Object) string {
	value, _ := payload.Get("host_key")
	text, _ := value.(string)
	if _, err := hostkey.DecodeKey(text); err != nil {
		return ""
	}
	return text
}
