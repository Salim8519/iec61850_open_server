// Runtime self-protection for the native workers, kept separate from the license
// verification itself (license_guard.cpp) so each file has one responsibility.
//
// check_no_debugger() throws when a debugger is attached. It uses only signals that
// Windows sets *exclusively* while a process is actually being debugged, so it never
// falsely blocks a legitimate user on a slow machine or virtual machine. No timing
// heuristics are used for that reason.
#pragma once
#include <string>
#include <vector>
#include "license_guard.hpp"

namespace integrity {

// Throws std::runtime_error when a debugger is attached to this process.
void check_no_debugger();

// The module-integrity public keys embedded in this worker (from module_key.hpp).
std::vector<license::PublicKey> module_keys();

// Verify the shipped compiled backend module against its detached Ed25519 signature
// (a raw 64-byte file next to the module), using module_keys(). The worker reads the
// file bytes itself, so a patched launcher cannot substitute a forged module.
// Throws std::runtime_error when the module or signature is missing or does not match.
void verify_module(const std::string& module_path, const std::string& signature_path,
                   const std::vector<license::PublicKey>& keys);

}  // namespace integrity
