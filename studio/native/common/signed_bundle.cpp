#include "signed_bundle.hpp"
#include "license_guard.hpp"
extern "C" {
#include "monocypher-ed25519.h"
}

namespace license {
namespace {

int base64_value(char c) {
    if (c >= 'A' && c <= 'Z') return c - 'A';
    if (c >= 'a' && c <= 'z') return c - 'a' + 26;
    if (c >= '0' && c <= '9') return c - '0' + 52;
    if (c == '-' || c == '+') return 62;
    if (c == '_' || c == '/') return 63;
    return -1;
}

}  // namespace

// Decode base64url or standard base64, with or without padding.
bool decode_base64url(const std::string& text, std::vector<std::uint8_t>& out) {
    unsigned buffer = 0;
    int bits = 0;
    for (char c : text) {
        if (c == '=') break;
        int value = base64_value(c);
        if (value < 0) return false;
        buffer = (buffer << 6) | static_cast<unsigned>(value);
        bits += 6;
        if (bits >= 8) {
            bits -= 8;
            out.push_back(static_cast<std::uint8_t>((buffer >> bits) & 0xFF));
        }
    }
    return true;
}

std::string key_id(const PublicKey& key) {
    return sha256_hex(std::string(reinterpret_cast<const char*>(key.data()), key.size())).substr(0, 8);
}

BundleCheck check_signed_bundle(const nlohmann::json& input, const std::vector<PublicKey>& keys,
                                nlohmann::json& payload) {
    nlohmann::json bundle = input;
    std::string message;
    try {
        if (bundle.contains("certificate")) bundle = bundle.at("certificate");
        message = bundle.at("payload").get<std::string>();
        payload = nlohmann::json::parse(message);
    } catch (const std::exception&) {
        return BundleCheck::damaged;
    }
    std::vector<std::uint8_t> signature;
    if (!bundle.contains("signature") || !bundle.at("signature").is_string() ||
        !decode_base64url(bundle.at("signature").get<std::string>(), signature) || signature.size() != 64) {
        return BundleCheck::unreadable_signature;
    }
    const std::string kid = payload.is_object() ? payload.value("kid", "") : "";
    for (const auto& key : keys) {
        if (!kid.empty() && key_id(key) != kid) continue;
        if (crypto_ed25519_check(signature.data(), key.data(),
                                 reinterpret_cast<const std::uint8_t*>(message.data()), message.size()) == 0) {
            return BundleCheck::ok;
        }
    }
    return BundleCheck::bad_signature;
}

}  // namespace license
