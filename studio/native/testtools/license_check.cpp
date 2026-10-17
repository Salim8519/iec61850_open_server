// Test-only command line for the native license guard (never packaged).
//   studio_license_check hwid
//   studio_license_check compose GUID SERIAL_HEX
//   studio_license_check legacy-hwid
//   studio_license_check compose-legacy GUID SERIAL_HEX
//   studio_license_check verify CERT_FILE PUBLIC_KEY_HEX HWID NOW [ANCHOR] [ISSUED_BEFORE]
//   studio_license_check enforced
#include "license_guard.hpp"
#include "integrity.hpp"
#include <cstdlib>
#include <fstream>
#include <iostream>
#include <sstream>
#include <string>

int main(int argc, char** argv) {
    try {
        const std::string command = argc > 1 ? argv[1] : "";
        if (command == "hwid") {
            std::cout << license::compute_hwid() << std::endl;
            return 0;
        }
        if (command == "legacy-hwid") {
            std::cout << license::compute_legacy_hwid() << std::endl;
            return 0;
        }
        if (command == "compose-legacy" && argc >= 4) {
            auto serial = static_cast<std::uint32_t>(std::stoul(argv[3], nullptr, 16));
            std::cout << license::compose_legacy_hwid(argv[2], serial) << std::endl;
            return 0;
        }
        if (command == "enforced") {
            std::cout << (license::enforced() ? 1 : 0) << std::endl;
            return 0;
        }
        if (command == "compose" && argc >= 4) {
            auto serial = static_cast<std::uint32_t>(std::stoul(argv[3], nullptr, 16));
            std::cout << license::compose_hwid(argv[2], serial, argc > 4 ? argv[4] : "") << std::endl;
            return 0;
        }
        if (command == "verify" && argc >= 6) {
            std::ifstream stream(argv[2], std::ios::binary);
            std::ostringstream text;
            text << stream.rdbuf();
            const std::string hex = argv[3];
            if (hex.size() != 64) throw std::runtime_error("public key must be 64 hex characters");
            license::PublicKey key{};
            for (std::size_t i = 0; i < key.size(); ++i) {
                key[i] = static_cast<std::uint8_t>(std::stoul(hex.substr(i * 2, 2), nullptr, 16));
            }
            long long anchor = argc > 6 ? std::atoll(argv[6]) : 0;
            long long issued_before = argc > 7 ? std::atoll(argv[7]) : 0;
            auto verdict = license::verify_certificate_text(text.str(), {key}, argv[4], std::atoll(argv[5]), anchor,
                                                            issued_before);
            std::cout << (verdict.ok ? "ok" : "fail: " + verdict.reason) << std::endl;
            return verdict.ok ? 0 : 1;
        }
        if (command == "verify-module" && argc == 5) {
            const std::string hex = argv[4];
            if (hex.size() != 64) throw std::runtime_error("public key must be 64 hex characters");
            license::PublicKey key{};
            for (std::size_t i = 0; i < key.size(); ++i) {
                key[i] = static_cast<std::uint8_t>(std::stoul(hex.substr(i * 2, 2), nullptr, 16));
            }
            integrity::verify_module(argv[2], argv[3], {key});
            std::cout << "ok" << std::endl;
            return 0;
        }
        std::cerr << "usage: studio_license_check hwid|enforced|compose|verify|verify-module ..." << std::endl;
        return 2;
    } catch (const std::exception& error) {
        std::cerr << error.what() << std::endl;
        return 3;
    }
}
