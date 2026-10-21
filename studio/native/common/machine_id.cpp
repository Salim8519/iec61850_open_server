#include "machine_id.hpp"
#include "license_guard.hpp"
#include "obfuscate.hpp"
#include <algorithm>
#include <cctype>
#include <stdexcept>
#ifdef _WIN32
#include <windows.h>
#endif

namespace license {
namespace {

const char* const kSpace = " \t\r\n\v\f";

std::string trimmed(std::string text) {
    const auto first = text.find_first_not_of(kSpace);
    if (first == std::string::npos) return "";
    text.erase(0, first);
    text.erase(text.find_last_not_of(kSpace) + 1);
    return text;
}

std::string ascii_lower(std::string text) {
    for (auto& c : text) {
        if (c >= 'A' && c <= 'Z') c = static_cast<char>(c + 32);
    }
    return text;
}

// The first three UUID fields are little-endian in the raw SMBIOS bytes.
std::string swap_uuid_fields(const std::string& hex32) {
    auto flip = [&](std::size_t from, std::size_t size) {
        std::string out;
        for (std::size_t i = size; i >= 2; i -= 2) out += hex32.substr(from + i - 2, 2);
        return out;
    };
    return flip(0, 8) + flip(8, 4) + flip(12, 4) + hex32.substr(16);
}

bool placeholder_uuid(const std::string& value) {
    static const char* const kDisplayed[] = {"03000200040005000006000700080009", "00020003000400050006000700080009",
                                             "123456781234567890abcddeefaabbcc"};
    for (const char* plain : kDisplayed) {
        if (value == plain || value == swap_uuid_fields(plain)) return true;
    }
    return false;
}

// String number index (1-based) of the string set data[from, to), which excludes the last NUL.
std::string string_number(const std::vector<std::uint8_t>& data, std::size_t from, std::size_t to, int index) {
    if (index < 1 || to <= from) return "";
    int current = 1;
    std::string text;
    for (std::size_t i = from; i < to; ++i) {
        if (data[i] == 0) {
            if (current == index) return text;
            ++current;
            text.clear();
        } else {
            text.push_back(static_cast<char>(data[i]));
        }
    }
    return current == index ? text : "";
}

}  // namespace

std::string clean_firmware_uuid(const std::string& raw) {
    const std::string value = ascii_lower(trimmed(raw));
    if (value.size() != 32) return "";
    if (value.find_first_not_of("0123456789abcdef") != std::string::npos) return "";
    bool repeated = true;
    for (std::size_t i = 2; i < 32; i += 2) repeated = repeated && value.compare(i, 2, value, 0, 2) == 0;
    return repeated || placeholder_uuid(value) ? "" : value;
}

std::string clean_board_serial(const std::string& raw) {
    const std::string text = trimmed(raw);
    const std::string lower = ascii_lower(text);
    if (lower.empty() || lower == "default string" || lower == "to be filled by o.e.m." || lower == "none" ||
        lower == "n/a" || lower == "0") {
        return "";
    }
    const bool repeated = std::all_of(text.begin(), text.end(), [&](char c) { return c == text[0]; });
    return repeated ? "" : text;
}

std::string compose_hwid(const std::string& firmware_uuid, const std::string& board_serial,
                         const std::string& machine_guid) {
    const std::string firmware = clean_firmware_uuid(firmware_uuid), board = clean_board_serial(board_serial);
    const std::string extra = firmware.empty() && board.empty() ? ascii_lower(trimmed(machine_guid)) : "";
    return sha256_hex(OBF_STR("scadabench-hwid-v3|") + firmware + "|" + board + "|" + extra);
}

std::string smbios_system_uuid(const std::vector<std::uint8_t>& raw) {
    if (raw.size() <= 8) return "";
    const std::uint8_t* data = raw.data() + 8;
    const std::size_t span = raw.size() - 8;
    static const char* hex = "0123456789abcdef";
    for (std::size_t pos = 0; pos + 4 <= span;) {
        const std::uint8_t kind = data[pos], length = data[pos + 1];
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

std::string smbios_board_serial(const std::vector<std::uint8_t>& raw) {
    if (raw.size() <= 8) return "";
    const std::vector<std::uint8_t> data(raw.begin() + 8, raw.end());
    for (std::size_t pos = 0; pos + 4 <= data.size();) {
        const std::uint8_t kind = data[pos];
        const std::size_t length = data[pos + 1];
        if (length < 4 || pos + length > data.size()) return "";
        std::size_t end = pos + length;
        while (end + 1 < data.size() && !(data[end] == 0 && data[end + 1] == 0)) ++end;
        if (kind == 2 && length >= 8) {
            return string_number(data, pos + length, std::min(end, data.size()), data[pos + 7]);
        }
        if (kind == 127) return "";
        pos = end + 2;
    }
    return "";
}

#ifdef _WIN32
namespace {
std::vector<std::uint8_t> smbios_table() {
    const DWORD provider = 0x52534D42;  // 'RSMB' raw SMBIOS firmware table
    const DWORD size = GetSystemFirmwareTable(provider, 0, nullptr, 0);
    if (size == 0) return {};
    std::vector<std::uint8_t> buffer(size);
    if (GetSystemFirmwareTable(provider, 0, buffer.data(), size) == 0) return {};
    return buffer;
}

std::string machine_guid() {
    wchar_t guid[128] = {};
    DWORD size = sizeof(guid);
    auto path = OBF_WSTR("SOFTWARE\\Microsoft\\Cryptography");
    auto name = OBF_WSTR("MachineGuid");
    if (RegGetValueW(HKEY_LOCAL_MACHINE, path.c_str(), name.c_str(), RRF_RT_REG_SZ | RRF_SUBKEY_WOW6464KEY,
                     nullptr, guid, &size) != ERROR_SUCCESS) {
        return "";
    }
    std::string narrow;
    for (const wchar_t* c = guid; *c; ++c) narrow.push_back(static_cast<char>(*c));
    return narrow;
}
}  // namespace
#endif

std::string compute_hwid() {
#ifdef _WIN32
    const auto table = smbios_table();
    const std::string firmware = smbios_system_uuid(table), board = smbios_board_serial(table);
    std::string guid;
    if (clean_firmware_uuid(firmware).empty() && clean_board_serial(board).empty()) {
        guid = machine_guid();  // only PCs without firmware identity fall back to this
        if (trimmed(guid).empty()) throw std::runtime_error("Cannot read this computer's hardware identity");
    }
    return compose_hwid(firmware, board, guid);
#else
    throw std::runtime_error("License checks require Windows");
#endif
}

}  // namespace license
