// A virtual machine without a license of its own may run on a licensed computer (the VM host key,
// docs/architecture/license-binding-contract.md). The Go backend asks the host for a proof and
// writes it as vm_proof.json beside license.cert:
//   {"certificate": {payload, signature}, "nonce": 32 hex, "vm_hwid": 64 hex, "signature": base64url,
//    "time": Unix seconds}
// The certificate is the host's (a lifetime license with a "host_key"); the signature is the host
// key's Ed25519 signature over "scadabench-vm-proof-v1|" + nonce + "|" + vm_hwid. This guard accepts
// the file only for its own machine ID and only while "time" is at most 24 hours old.
#pragma once
#include "license_clock.hpp"
#include <string>
#include <vector>

namespace license {

constexpr long long kVmProofMaxAge = 24 * 3600;
constexpr long long kVmProofFutureTolerance = 300;

// text is the content of vm_proof.json; hwid is this computer's machine ID; clock holds the
// cancelled license IDs and the newest time seen (a cancelled host license is refused here too).
Verdict verify_vm_proof(const std::string& text, const std::vector<PublicKey>& keys, const std::string& hwid,
                        long long now, const ClockState& clock);

}  // namespace license
