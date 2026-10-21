// Ed25519 check of a {payload, signature} bundle, shared by license certificates and
// the license server's signed time stamps. The payload is verified as the exact string.
#pragma once
#include <array>
#include <cstdint>
#include <string>
#include <vector>
#include <nlohmann/json.hpp>

namespace license {

using PublicKey = std::array<std::uint8_t, 32>;

enum class BundleCheck { ok, damaged, unreadable_signature, bad_signature };

// Parses bundle (optionally wrapped as {"certificate": {...}}), checks the signature
// against keys (selected by the payload's "kid"), and fills payload on success.
BundleCheck check_signed_bundle(const nlohmann::json& bundle, const std::vector<PublicKey>& keys,
                                nlohmann::json& payload);

// Decodes base64url or standard base64, padded or not; false for any other character.
bool decode_base64url(const std::string& text, std::vector<std::uint8_t>& out);

// The 8-hex key id carried as "kid" (first 4 bytes of SHA-256 of the public key).
std::string key_id(const PublicKey& key);

}  // namespace license
