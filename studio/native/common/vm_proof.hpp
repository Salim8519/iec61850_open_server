// A virtual machine without a license of its own may run on a licensed computer (the VM host key,
// docs/architecture/license-binding-contract.md). The Go backend asks the host for a proof and
// writes it as vm_proof.json beside license.cert:
//   {"certificate": {payload, signature}, "nonce": 32 hex, "vm_hwid": 64 hex, "signature": base64url,
//    "time": the host's Unix seconds, signed}
// The certificate is the host's (a lifetime license with a "host_key"); the signature is the host
// key's Ed25519 signature over "scadabench-vm-proof-v2|" + nonce + "|" + vm_hwid + "|" + time (decimal).
// This guard accepts the file only for its own machine ID and only while that signed time is at most
// 24 hours old, at most 10 minutes ahead of this clock and at most 24 hours older than the newest time
// this computer has seen (the license clock anchor), so editing "time" or setting the clock back never
// brings an old proof back.
#pragma once
#include "license_clock.hpp"
#include <string>
#include <vector>

namespace license {

constexpr long long kVmProofMaxAge = 24 * 3600;
constexpr long long kVmProofFutureTolerance = 600;

// text is the content of vm_proof.json; hwid is this computer's machine ID; clock holds the
// cancelled license IDs and the newest time seen (a cancelled host license is refused here too).
Verdict verify_vm_proof(const std::string& text, const std::vector<PublicKey>& keys, const std::string& hwid,
                        long long now, const ClockState& clock);

}  // namespace license
