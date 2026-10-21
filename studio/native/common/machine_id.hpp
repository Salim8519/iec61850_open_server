// Machine ID v3 of the native guard (ADR-0098), identical to the Go engine's
// studio/engine/internal/licensing/hwid.go:
//   hwid = SHA256_HEX("scadabench-hwid-v3|" + firmware + "|" + board + "|" + extra)
// firmware = SMBIOS type 1 UUID (raw bytes, lowercase hex), board = SMBIOS type 2 baseboard
// serial number (trimmed); each "" when it is a placeholder every cheap PC shares. extra is ""
// unless BOTH are "", then the lowercased MachineGuid. Nothing here reads anything but the
// firmware table and, only for PCs without firmware identity, the MachineGuid.
// The native guard cannot do the TPM proof of a chip-bound license; the Go backend does it.
#pragma once
#include <cstdint>
#include <string>
#include <vector>

namespace license {

// Firmware part: lowercase hex, or "" for an empty, malformed, one-repeated-byte or placeholder UUID.
std::string clean_firmware_uuid(const std::string& raw);
// Board part: the serial trimmed of ASCII whitespace, or "" for "", "Default string",
// "To be filled by O.E.M.", "None", "N/A", "0" (any case) or one repeated character.
std::string clean_board_serial(const std::string& raw);

// The v3 machine ID of raw identifiers (junk is dropped here).
std::string compose_hwid(const std::string& firmware_uuid, const std::string& board_serial,
                         const std::string& machine_guid);

// Lowercase hex of the 16 raw UUID bytes of the first SMBIOS type 1 structure, or "".
// raw is a RawSMBIOSData blob (8-byte header).
std::string smbios_system_uuid(const std::vector<std::uint8_t>& raw);
// Serial number string of the first SMBIOS type 2 (baseboard) structure, or "".
std::string smbios_board_serial(const std::vector<std::uint8_t>& raw);

// This computer's v3 machine ID; throws std::runtime_error when nothing identifies it.
std::string compute_hwid();

}  // namespace license
