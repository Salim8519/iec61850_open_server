#include "vm_proof.hpp"
#include "obfuscate.hpp"
#include "signed_bundle.hpp"
#include <nlohmann/json.hpp>
#include <algorithm>
#include <cctype>
extern "C" {
#include "monocypher-ed25519.h"
}

namespace license {
namespace {

Verdict fail(const std::string& reason) { return {false, reason}; }

bool lowercase_hex(const std::string& text, std::size_t length) {
    if (text.size() != length) return false;
    return std::all_of(text.begin(), text.end(),
                       [](unsigned char c) { return std::isdigit(c) || (c >= 'a' && c <= 'f'); });
}

// The host's license: genuine, lifetime, not cancelled, naming a host key. Fills host_key.
Verdict check_host_certificate(const nlohmann::json& certificate, const std::vector<PublicKey>& keys,
                               long long now, const ClockState& clock, PublicKey& host_key) {
    nlohmann::json payload;
    if (check_signed_bundle(certificate, keys, payload) != BundleCheck::ok) {
        return fail("The host's license file is damaged or was changed");
    }
    try {
        if (payload.at("version").get<int>() != 1) return fail("Unsupported license version");
        if (payload.at("type").get<std::string>() != "lifetime") return fail("Only a lifetime license can be shared");
        const long long issued = payload.at("issued_at").get<long long>();
        const long long expires = payload.value("expires_at", 0LL);
        if (now + 300 < std::max(issued, clock.anchor)) return fail("System clock rollback detected");
        if (expires > 0 && now >= expires) return fail("The license has expired");
        const auto id = payload.at("license_id").get<std::string>();
        if (clock.revoked.count(id)) return fail("The license was cancelled by the seller");
        std::vector<std::uint8_t> raw;
        if (!decode_base64url(payload.value("host_key", std::string()), raw) || raw.size() != host_key.size()) {
            return fail("The host's license has no virtual machine key");
        }
        std::copy(raw.begin(), raw.end(), host_key.begin());
    } catch (const std::exception&) {
        return fail("The host's license file is damaged");
    }
    return {true, ""};
}

}  // namespace

Verdict verify_vm_proof(const std::string& text, const std::vector<PublicKey>& keys, const std::string& hwid,
                        long long now, const ClockState& clock) {
    if (keys.empty()) return fail("This build has no license key configured");
    nlohmann::json proof, certificate;
    std::string nonce, vm_hwid, signature_text;
    long long time = 0;
    try {
        proof = nlohmann::json::parse(text);
        nonce = proof.at("nonce").get<std::string>();
        vm_hwid = proof.at("vm_hwid").get<std::string>();
        signature_text = proof.at("signature").get<std::string>();
        time = proof.at("time").get<long long>();
        certificate = proof.at("certificate");
    } catch (const std::exception&) {
        return fail("The virtual machine proof is damaged");
    }
    if (vm_hwid != hwid) return fail("The virtual machine proof is for another computer");
    if (time > now + kVmProofFutureTolerance || now - time > kVmProofMaxAge) {
        return fail("The virtual machine proof is out of date");
    }
    if (!lowercase_hex(nonce, 32) || !lowercase_hex(vm_hwid, 64)) return fail("The virtual machine proof is damaged");
    PublicKey host_key{};
    const auto host = check_host_certificate(certificate, keys, now, clock, host_key);
    if (!host.ok) return host;
    std::vector<std::uint8_t> signature;
    if (!decode_base64url(signature_text, signature) || signature.size() != 64) {
        return fail("The virtual machine proof is damaged");
    }
    const std::string message = OBF_STR("scadabench-vm-proof-v1|") + nonce + "|" + vm_hwid;
    if (crypto_ed25519_check(signature.data(), host_key.data(),
                             reinterpret_cast<const std::uint8_t*>(message.data()), message.size()) != 0) {
        return fail("The host did not confirm this virtual machine");
    }
    return {true, ""};
}

}  // namespace license
