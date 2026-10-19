#include "license_guard.hpp"
#include "integrity.hpp"
#include "license_clock.hpp"
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

namespace {
// "<guid lowercased, trimmed>|<%08X serial>", the inputs shared by v1 and v2.
std::string machine_text(const std::string& machine_guid, std::uint32_t volume_serial) {
    std::string guid = machine_guid;
    guid.erase(0, guid.find_first_not_of(" \t\r\n"));
    guid.erase(guid.find_last_not_of(" \t\r\n") + 1);
    std::transform(guid.begin(), guid.end(), guid.begin(), [](unsigned char c) { return std::tolower(c); });
    char serial[9];
    std::snprintf(serial, sizeof(serial), "%08X", volume_serial);
    return guid + "|" + serial;
}
}  // namespace

std::string compose_hwid(const std::string& machine_guid, std::uint32_t volume_serial,
                         const std::string& firmware_uuid) {
    return sha256_hex(OBF_STR("substation-studio-hwid-v2|") + machine_text(machine_guid, volume_serial) +
                      "|" + firmware_uuid);
}

std::string compose_legacy_hwid(const std::string& machine_guid, std::uint32_t volume_serial) {
    return sha256_hex(OBF_STR("substation-studio-hwid-v1|") + machine_text(machine_guid, volume_serial));
}

#ifdef _WIN32
namespace {
// Lowercase hex of the 16 raw SMBIOS Type 1 (System Information) UUID bytes, or ""
// when unavailable. Raw bytes are used verbatim to match hwid.py exactly.
std::string firmware_uuid_hex() {
    const DWORD provider = 0x52534D42;  // 'RSMB' raw SMBIOS firmware table
    DWORD size = GetSystemFirmwareTable(provider, 0, nullptr, 0);
    if (size == 0) return "";
    std::vector<std::uint8_t> buffer(size);
    if (GetSystemFirmwareTable(provider, 0, buffer.data(), size) == 0 || buffer.size() <= 8) return "";
    const std::uint8_t* data = buffer.data() + 8;  // skip 8-byte RawSMBIOSData header
    std::size_t span = buffer.size() - 8, pos = 0;
    static const char* hex = "0123456789abcdef";
    while (pos + 4 <= span) {
        std::uint8_t kind = data[pos], length = data[pos + 1];
        if (length < 4) break;
        if (kind == 1 && pos + 24 <= span) {
            std::string out;
            for (std::size_t i = 0; i < 16; ++i) {
                out.push_back(hex[data[pos + 8 + i] >> 4]);
                out.push_back(hex[data[pos + 8 + i] & 15]);
            }
            return out;
        }
        pos += length;
        while (pos + 1 < span && !(data[pos] == 0 && data[pos + 1] == 0)) ++pos;
        pos += 2;
    }
    return "";
}

void read_machine_inputs(std::string& narrow, DWORD& serial) {
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
    if (!GetVolumeInformationW(root.c_str(), nullptr, 0, &serial, nullptr, nullptr, nullptr, 0)) {
        throw std::runtime_error("Cannot read the system drive serial number");
    }
    for (const wchar_t* c = guid; *c; ++c) narrow.push_back(static_cast<char>(*c));
}
}  // namespace
#endif

std::string compute_hwid() {
#ifdef _WIN32
    std::string guid;
    DWORD serial = 0;
    read_machine_inputs(guid, serial);
    return compose_hwid(guid, serial, firmware_uuid_hex());
#else
    throw std::runtime_error("License checks require Windows");
#endif
}

std::string compute_legacy_hwid() {
#ifdef _WIN32
    std::string guid;
    DWORD serial = 0;
    read_machine_inputs(guid, serial);
    return compose_legacy_hwid(guid, serial);
#else
    throw std::runtime_error("License checks require Windows");
#endif
}

Verdict verify_certificate_text(const std::string& text, const std::vector<PublicKey>& keys,
                                const std::string& hwid, long long now, long long anchor,
                                long long hwid_issued_before, long long tolerance) {
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
        if (hwid_issued_before > 0 && issued >= hwid_issued_before) {
            return fail("The license belongs to another computer");
        }
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
    auto verdict = verify_license(text, release_keys(), compute_hwid(), now, clock);
    if (!verdict.ok && verdict.reason.find("another computer") != std::string::npos) {
        // Licenses activated by pre-v2 builds stay valid on the same PC (ADR-0051).
        auto legacy = verify_license(text, release_keys(), compute_legacy_hwid(), now, clock, kLegacyIssuedBefore);
        if (legacy.ok) verdict = legacy;
    }
    if (!verdict.ok) throw std::runtime_error("License check failed: " + verdict.reason);
    verified = true;
}
#else
void require_valid_license() {}
#endif

}  // namespace license
