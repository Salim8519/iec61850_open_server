// Package hostkey is the VM host key of ADR-0099 (docs/architecture/license-binding-contract.md):
// a licensed PC can prove its license to the virtual machines on it, offline.
//
// The licensed PC owns an Ed25519 host key pair. The private key is kept encrypted with Windows
// DPAPI (current user), the public key goes into the certificate. The host helper
// (cmd/scadabench_hostkey) answers a proof request on TCP port Port, only on the adapters of
// virtual switches, and only while this PC's license is valid. A VM sends a fresh nonce and its own
// machine ID and accepts the answer when the certificate verifies with the release keys and the
// signature verifies with the certificate's host key.
//
// This package knows nothing about certificates: the licensing package checks them and gives the
// server its Responder. Everything that touches Windows (DPAPI, adapters, the Run key, the mutex)
// sits behind a small seam so every rule is tested with fakes.
package hostkey

// Constants shared by the host helper, the app and the VM side.
const (
	// Port is the TCP port of the proof service on the host's virtual-switch adapters.
	Port = 47862
	// ProofPrefix starts the signed message: ProofPrefix + nonce + "|" + vm_hwid.
	ProofPrefix = "scadabench-vm-proof-v1|"
	// HelperExe is the host helper program shipped beside the native workers.
	HelperExe = "scadabench_hostkey.exe"
	// RunValueName is the HKCU Run value that starts the helper at logon (only while sharing is on).
	RunValueName = "ScadaBenchVMHost"
	// RunKeyPath is the per-user Run key.
	RunKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	// MutexName makes the helper a single instance per Windows session.
	MutexName = `Local\ScadaBenchVMHost`
	// KeyFile is the DPAPI-protected private key in the license home.
	KeyFile = "host_key.dpapi"
	// MaxRequestBytes bounds one request line.
	MaxRequestBytes = 4096
	// MaxAnswerBytes bounds one answer line (a certificate is about 1 KiB).
	MaxAnswerBytes = 64 * 1024
)
