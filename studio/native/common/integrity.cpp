#include "integrity.hpp"
#include "module_key.hpp"
#include <fstream>
#include <sstream>
#include <stdexcept>
extern "C" {
#include "monocypher-ed25519.h"
}
#ifdef _WIN32
#include <windows.h>
#include <intrin.h>
#endif

namespace integrity {
namespace {
#ifdef _WIN32

// All of these are signals Windows sets ONLY when a debugger is present, so they
// never falsely trip on a slow or virtualised machine. No timing heuristics are
// used, on purpose: a timing check would risk blocking a legitimate user.
const unsigned char* process_environment_block() {
    return reinterpret_cast<const unsigned char*>(__readgsqword(0x60));  // x64 PEB
}

bool peb_being_debugged() {
    const unsigned char* peb = process_environment_block();
    return peb != nullptr && peb[0x02] != 0;  // PEB.BeingDebugged
}

bool peb_ntglobalflag_debugger_heap() {
    const unsigned char* peb = process_environment_block();
    if (peb == nullptr) return false;
    DWORD flags = *reinterpret_cast<const DWORD*>(peb + 0xBC);  // PEB.NtGlobalFlag
    // These heap flags are set together when a process is created under a debugger.
    return (flags & 0x70) != 0;  // TAIL_CHECK | FREE_CHECK | VALIDATE_PARAMETERS
}
#endif
}  // namespace

void check_no_debugger() {
#ifdef _WIN32
    BOOL remote = FALSE;
    CheckRemoteDebuggerPresent(GetCurrentProcess(), &remote);
    if (IsDebuggerPresent() || remote || peb_being_debugged() || peb_ntglobalflag_debugger_heap()) {
        throw std::runtime_error("ScadaBench cannot run while a debugger is attached");
    }
#endif
}

std::vector<license::PublicKey> module_keys() {
    std::vector<license::PublicKey> keys;
    for (std::size_t k = 0; k < module_key::kKeyCount; ++k) {
        license::PublicKey key{};
        for (std::size_t i = 0; i < key.size(); ++i) {
            key[i] = module_key::kMaskedKeys[k][i] ^ module_key::kMask[i % module_key::kMask.size()];
        }
        keys.push_back(key);
    }
    return keys;
}

namespace {
std::string read_all(const std::string& path) {
    std::ifstream stream(path, std::ios::binary);
    if (!stream) throw std::runtime_error("ScadaBench program files were changed or damaged. Reinstall ScadaBench.");
    std::ostringstream content;
    content << stream.rdbuf();
    return content.str();
}
}  // namespace

void verify_module(const std::string& module_path, const std::string& signature_path,
                   const std::vector<license::PublicKey>& keys) {
    const std::string tamper = "ScadaBench program files were changed or damaged. Reinstall ScadaBench.";
    if (keys.empty()) throw std::runtime_error(tamper);
    const std::string digest = license::sha256_hex(read_all(module_path));  // 64 lowercase hex chars
    const std::string signature = read_all(signature_path);
    if (signature.size() != 64) throw std::runtime_error(tamper);
    const auto* message = reinterpret_cast<const std::uint8_t*>(digest.data());
    const auto* sig = reinterpret_cast<const std::uint8_t*>(signature.data());
    for (const auto& key : keys) {
        if (crypto_ed25519_check(sig, key.data(), message, digest.size()) == 0) return;
    }
    throw std::runtime_error(tamper);
}

}  // namespace integrity
