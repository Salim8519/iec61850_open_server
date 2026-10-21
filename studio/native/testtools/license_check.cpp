// Test-only command line for the native license guard (never packaged).
//   studio_license_check hwid                                (this computer's machine ID v3)
//   studio_license_check compose FIRMWARE_HEX BOARD_SERIAL MACHINE_GUID   (any may be "")
//   studio_license_check smbios RAW_SMBIOS_HEX               (prints "<system UUID>|<board serial>")
//   studio_license_check verify CERT_FILE PUBLIC_KEY_HEX HWID NOW [ANCHOR]
//   studio_license_check verify-clock CERT_FILE PUBLIC_KEY_HEX HWID NOW FOLDER   (no registry)
//   studio_license_check enforced
#include "license_guard.hpp"
#include "integrity.hpp"
#include "license_clock.hpp"
#include "machine_id.hpp"
#include <cstdlib>
#include <fstream>
#include <iostream>
#include <sstream>
#include <string>

namespace {
license::PublicKey parse_key(const std::string& hex) {
    if (hex.size() != 64) throw std::runtime_error("public key must be 64 hex characters");
    license::PublicKey key{};
    for (std::size_t i = 0; i < key.size(); ++i) {
        key[i] = static_cast<std::uint8_t>(std::stoul(hex.substr(i * 2, 2), nullptr, 16));
    }
    return key;
}
}  // namespace

int main(int argc, char** argv) {
    try {
        const std::string command = argc > 1 ? argv[1] : "";
        if (command == "hwid") {
            std::cout << license::compute_hwid() << std::endl;
            return 0;
        }
        if (command == "enforced") {
            std::cout << (license::enforced() ? 1 : 0) << std::endl;
            return 0;
        }
        if (command == "compose" && argc == 5) {
            std::cout << license::compose_hwid(argv[2], argv[3], argv[4]) << std::endl;
            return 0;
        }
        if (command == "smbios" && argc == 3) {
            const std::string hex = argv[2];
            std::vector<std::uint8_t> raw;
            for (std::size_t i = 0; i + 1 < hex.size(); i += 2) {
                raw.push_back(static_cast<std::uint8_t>(std::stoul(hex.substr(i, 2), nullptr, 16)));
            }
            std::cout << license::smbios_system_uuid(raw) << "|" << license::smbios_board_serial(raw) << std::endl;
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
            auto verdict = license::verify_certificate_text(text.str(), {key}, argv[4], std::atoll(argv[5]), anchor);
            std::cout << (verdict.ok ? "ok" : "fail: " + verdict.reason) << std::endl;
            return verdict.ok ? 0 : 1;
        }
        if (command == "verify-clock" && argc == 7) {
            std::string folder = argv[6];
            if (!folder.empty() && folder.back() != '\\' && folder.back() != '/') folder += '\\';
            const std::vector<license::PublicKey> keys{parse_key(argv[3])};
            const auto clock = license::read_clock_state(folder, keys, false);
            auto verdict = license::verify_license(license::read_text_file(argv[2]), keys, argv[4],
                                                   std::atoll(argv[5]), clock);
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
        std::cerr << "usage: studio_license_check hwid|enforced|compose|verify|verify-clock|verify-module ..." << std::endl;
        return 2;
    } catch (const std::exception& error) {
        std::cerr << error.what() << std::endl;
        return 3;
    }
}
