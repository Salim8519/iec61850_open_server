// Independent license check inside the native workers. Release builds configured
// with -DSTUDIO_ENFORCE_LICENSE=ON refuse to serve MMS/GOOSE/SV without a valid
// certificate for this machine; development builds compile the same code but
// require_valid_license() returns immediately. There is no runtime switch.
#pragma once
#include <array>
#include <cstdint>
#include <string>
#include <vector>

namespace license {

using PublicKey = std::array<std::uint8_t, 32>;

struct Verdict {
    bool ok = false;
    std::string reason;
};

// SHA-256 of text as 64 lowercase hex characters (Windows CNG).
std::string sha256_hex(const std::string& text);

// The machine ID (v3, ADR-0098) is in machine_id.hpp.

// Verify a license.cert JSON text. now/anchor are Unix seconds. tolerance is how far the
// clock may sit behind the anchor (trials pass a generous one).
Verdict verify_certificate_text(const std::string& text, const std::vector<PublicKey>& keys,
                                const std::string& hwid, long long now, long long anchor,
                                long long tolerance = 300);

std::vector<PublicKey> release_keys();

// True only in builds configured with STUDIO_ENFORCE_LICENSE.
bool enforced();

// Throws std::runtime_error with a user-readable reason when enforcement is on and
// the license is missing or invalid; caches a successful check for the process.
void require_valid_license();

}  // namespace license
