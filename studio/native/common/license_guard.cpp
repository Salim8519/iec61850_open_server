#include "license_guard.hpp"
#include "integrity.hpp"
#include "license_clock.hpp"
#include "machine_id.hpp"
#include "obfuscate.hpp"
#include "release_key.hpp"
#include "signed_bundle.hpp"
#include <nlohmann/json.hpp>
#include <algorithm>
#include <cctype>
#include <cstdio>
#include <cstdlib>
#include <ctime>
#include <mutex>
#include <stdexcept>
#ifdef _WIN32
#include <windows.h>
#include <bcrypt.h>
#endif

namespace license {
namespace {

Verdict fail(const std::string& reason) { return {false, reason}; }

}  // namespace

std::string sha256_hex(const std::string& text) {
    unsigned char digest[32] = {};
#ifdef _WIN32
    if (BCryptHash(BCRYPT_SHA256_ALG_HANDLE, nullptr, 0,
                   reinterpret_cast<PUCHAR>(const_cast<char*>(text.data())),
                   static_cast<ULONG>(text.size()), digest, sizeof(digest)) != 0) {
        throw std::runtime_error("SHA-256 is unavailable on this system");
    }
#else
    throw std::runtime_error("License checks require Windows");
#endif
    static const char* hex = "0123456789abcdef";
    std::string result;
    for (unsigned char byte : digest) {
        result.push_back(hex[byte >> 4]);
        result.push_back(hex[byte & 15]);
    }
    return result;
}

Verdict verify_certificate_text(const std::string& text, const std::vector<PublicKey>& keys,
                                const std::string& hwid, long long now, long long anchor,
                                long long tolerance) {
    if (keys.empty()) return fail("This build has no license key configured");
    nlohmann::json bundle, payload;
    try {
        bundle = nlohmann::json::parse(text);
    } catch (const std::exception&) {
        return fail("The license file is damaged");
    }
    switch (check_signed_bundle(bundle, keys, payload)) {
        case BundleCheck::ok: break;
        case BundleCheck::damaged: return fail("The license file is damaged");
        case BundleCheck::unreadable_signature: return fail("The license signature is unreadable");
        case BundleCheck::bad_signature: return fail("The license file is damaged or was changed");
    }
    try {
        if (payload.at("version").get<int>() != 1) return fail("Unsupported license version");
        const auto type = payload.at("type").get<std::string>();
        if (type != "trial" && type != "lifetime") return fail("Unknown license type");
        if (payload.at("hwid").get<std::string>() != hwid) return fail("The license belongs to another computer");
        const long long issued = payload.at("issued_at").get<long long>();
        const long long expires = payload.value("expires_at", 0LL);
        if (now + tolerance < std::max(issued, anchor)) return fail("System clock rollback detected");
        if (expires > 0 && now >= expires) return fail("The license has expired");
    } catch (const std::exception&) {
        return fail("The license file is damaged");
    }
    return {true, ""};
}

std::vector<PublicKey> release_keys() {
    std::vector<PublicKey> keys;
    for (std::size_t k = 0; k < release_key::kKeyCount; ++k) {
        PublicKey key{};
        for (std::size_t i = 0; i < key.size(); ++i) {
            key[i] = release_key::kMaskedKeys[k][i] ^ release_key::kMask[i % release_key::kMask.size()];
        }
        keys.push_back(key);
    }
    return keys;
}

bool enforced() {
#ifdef STUDIO_ENFORCE_LICENSE
    return true;
#else
    return false;
#endif
}

#ifdef STUDIO_ENFORCE_LICENSE
namespace {

std::string certificate_path() {
    if (const char* explicit_path = std::getenv("STUDIO_LICENSE_CERT_PATH")) return explicit_path;
    const char* base = std::getenv("LOCALAPPDATA");
    return std::string(base ? base : ".") + OBF_STR("\\ScadaBench\\license.cert");
}

}  // namespace

void require_valid_license() {
    static std::mutex lock;
    static bool verified = false;
    std::lock_guard<std::mutex> guard(lock);
    if (verified) return;
    integrity::check_no_debugger();
    // The packaged backend passes its own compiled-module path; verify it is the exact
    // module we shipped. Absent in development and tests (no packaged module), so skipped.
    if (const char* module = std::getenv("STUDIO_MODULE_PATH")) {
        const char* signature = std::getenv("STUDIO_MODULE_SIG");
        integrity::verify_module(module, signature ? signature : "", integrity::module_keys());
    }
    const auto path = certificate_path();
    const auto text = read_text_file(path);
    if (text.empty()) throw std::runtime_error("No license on this computer. Activate Studio first.");
    const auto directory = path.substr(0, path.find_last_of("\\/") + 1);
    const auto now = static_cast<long long>(std::time(nullptr));
    // Anchor, signed server time and grace start from all three places (ADR-0046 addendum).
    const auto clock = read_clock_state(directory, release_keys(), true);
    // A chip-bound license (ADR-0098) also needs the TPM proof; the Go backend does that. The
    // backend starts the workers only for a licensed computer, so this guard checks the
    // certificate, the machine ID v3, the clock and the cancelled IDs.
    const auto verdict = verify_license(text, release_keys(), compute_hwid(), now, clock);
    if (!verdict.ok) throw std::runtime_error("License check failed: " + verdict.reason);
    verified = true;
}
#else
void require_valid_license() {}
#endif

}  // namespace license
