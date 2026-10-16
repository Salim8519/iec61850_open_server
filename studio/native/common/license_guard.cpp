#include "license_guard.hpp"
#include "obfuscate.hpp"
#include "release_key.hpp"
#include <nlohmann/json.hpp>
#include <algorithm>
#include <cctype>
#include <cstdio>
#include <cstdlib>
#include <ctime>
#include <fstream>
#include <mutex>
#include <sstream>
#include <stdexcept>
extern "C" {
#include "monocypher-ed25519.h"
}
#ifdef _WIN32
#include <windows.h>
#include <bcrypt.h>
#endif

namespace license {
namespace {

constexpr long long kClockTolerance = 300;

int base64_value(char c) {
    if (c >= 'A' && c <= 'Z') return c - 'A';
    if (c >= 'a' && c <= 'z') return c - 'a' + 26;
    if (c >= '0' && c <= '9') return c - '0' + 52;
    if (c == '-' || c == '+') return 62;
    if (c == '_' || c == '/') return 63;
    return -1;
}

// Decode base64url or standard base64, with or without padding.
bool decode_base64(const std::string& text, std::vector<std::uint8_t>& out) {
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

std::string compose_hwid(const std::string& machine_guid, std::uint32_t volume_serial) {
    std::string guid = machine_guid;
    guid.erase(0, guid.find_first_not_of(" \t\r\n"));
    guid.erase(guid.find_last_not_of(" \t\r\n") + 1);
    std::transform(guid.begin(), guid.end(), guid.begin(), [](unsigned char c) { return std::tolower(c); });
    char serial[9];
    std::snprintf(serial, sizeof(serial), "%08X", volume_serial);
    return sha256_hex(OBF_STR("substation-studio-hwid-v1|") + guid + "|" + serial);
}

std::string compute_hwid() {
#ifdef _WIN32
    wchar_t guid[128] = {};
    DWORD size = sizeof(guid);
    auto path = OBF_WSTR("SOFTWARE\\Microsoft\\Cryptography");
    auto name = OBF_WSTR("MachineGuid");
    if (RegGetValueW(HKEY_LOCAL_MACHINE, path.c_str(), name.c_str(), RRF_RT_REG_SZ | RRF_SUBKEY_WOW6464KEY,
                     nullptr, guid, &size) != ERROR_SUCCESS) {
        throw std::runtime_error("Cannot read this computer's machine ID");
    }
    wchar_t drive[MAX_PATH] = L"C:";
    GetEnvironmentVariableW(L"SystemDrive", drive, MAX_PATH);
    std::wstring root = std::wstring(drive) + L"\\";
    DWORD serial = 0;
    if (!GetVolumeInformationW(root.c_str(), nullptr, 0, &serial, nullptr, nullptr, nullptr, 0)) {
        throw std::runtime_error("Cannot read the system drive serial number");
    }
    std::string narrow;
    for (const wchar_t* c = guid; *c; ++c) narrow.push_back(static_cast<char>(*c));
    return compose_hwid(narrow, serial);
#else
    throw std::runtime_error("License checks require Windows");
#endif
}

Verdict verify_certificate_text(const std::string& text, const std::vector<PublicKey>& keys,
                                const std::string& hwid, long long now, long long anchor) {
    if (keys.empty()) return fail("This build has no license key configured");
    nlohmann::json bundle, payload;
    try {
        bundle = nlohmann::json::parse(text);
        if (bundle.contains("certificate")) bundle = bundle.at("certificate");
        payload = nlohmann::json::parse(bundle.at("payload").get<std::string>());
    } catch (const std::exception&) {
        return fail("The license file is damaged");
    }
    const std::string message = bundle.at("payload").get<std::string>();
    std::vector<std::uint8_t> signature;
    if (!bundle.contains("signature") || !bundle.at("signature").is_string() ||
        !decode_base64(bundle.at("signature").get<std::string>(), signature) || signature.size() != 64) {
        return fail("The license signature is unreadable");
    }
    const std::string kid = payload.value("kid", "");
    bool signed_ok = false;
    for (const auto& key : keys) {
        if (!kid.empty() && key_id(key) != kid) continue;
        if (crypto_ed25519_check(signature.data(), key.data(),
                                 reinterpret_cast<const std::uint8_t*>(message.data()), message.size()) == 0) {
            signed_ok = true;
            break;
        }
    }
    if (!signed_ok) return fail("The license file is damaged or was changed");
    try {
        if (payload.at("version").get<int>() != 1) return fail("Unsupported license version");
        const auto type = payload.at("type").get<std::string>();
        if (type != "trial" && type != "lifetime") return fail("Unknown license type");
        if (payload.at("hwid").get<std::string>() != hwid) return fail("The license belongs to another computer");
        const long long issued = payload.at("issued_at").get<long long>();
        const long long expires = payload.value("expires_at", 0LL);
        if (now + kClockTolerance < std::max(issued, anchor)) return fail("System clock rollback detected");
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

std::string read_file(const std::string& path) {
    std::ifstream stream(path, std::ios::binary);
    std::ostringstream content;
    content << stream.rdbuf();
    return content.str();
}

std::string certificate_path() {
    if (const char* explicit_path = std::getenv("STUDIO_LICENSE_CERT_PATH")) return explicit_path;
    const char* base = std::getenv("LOCALAPPDATA");
    return std::string(base ? base : ".") + OBF_STR("\\ScadaBench\\license.cert");
}

void check_debugger() {
    BOOL remote = FALSE;
    CheckRemoteDebuggerPresent(GetCurrentProcess(), &remote);
    if (IsDebuggerPresent() || remote) throw std::runtime_error("Studio cannot run under a debugger");
}

}  // namespace

void require_valid_license() {
    static std::mutex lock;
    static bool verified = false;
    std::lock_guard<std::mutex> guard(lock);
    if (verified) return;
    check_debugger();
    const auto path = certificate_path();
    const auto text = read_file(path);
    if (text.empty()) throw std::runtime_error("No license on this computer. Activate Studio first.");
    const auto directory = path.substr(0, path.find_last_of("\\/") + 1);
    const auto anchor_text = read_file(directory + ".time_anchor");
    const long long anchor = anchor_text.empty() ? 0 : std::atoll(anchor_text.c_str());
    auto verdict = verify_certificate_text(text, release_keys(), compute_hwid(),
                                           static_cast<long long>(std::time(nullptr)), anchor);
    if (!verdict.ok) throw std::runtime_error("License check failed: " + verdict.reason);
    verified = true;
}
#else
void require_valid_license() {}
#endif

}  // namespace license
