#include "license_clock.hpp"
#include "obfuscate.hpp"
#include "signed_bundle.hpp"
#include <nlohmann/json.hpp>
#include <algorithm>
#include <cerrno>
#include <cstdlib>
#include <fstream>
#include <sstream>
#ifdef _WIN32
#include <windows.h>
#endif

namespace license {
namespace {

struct Place {
    const char* file;      // dot-file beside license.cert
    const char* state;     // license_state.json field
    const wchar_t* value;  // HKCU\Software\ScadaBench value
};
const Place kAnchor{".time_anchor", "time_anchor", L"TimeAnchor"};
const Place kServerTime{".server_time", "server_time", L"ServerTime"};
const Place kGraceStart{".grace_start", "grace_start", L"GraceStart"};

std::string trimmed(std::string text) {
    text.erase(0, text.find_first_not_of(" \t\r\n"));
    text.erase(text.find_last_not_of(" \t\r\n") + 1);
    return text;
}

std::string registry_text(const wchar_t* name) {
#ifdef _WIN32
    wchar_t buffer[4096] = {};
    DWORD size = sizeof(buffer);
    auto key = OBF_WSTR("Software\\ScadaBench");
    if (RegGetValueW(HKEY_CURRENT_USER, key.c_str(), name, RRF_RT_REG_SZ, nullptr, buffer, &size) != ERROR_SUCCESS) {
        return "";
    }
    std::string narrow;
    for (const wchar_t* c = buffer; *c; ++c) narrow.push_back(static_cast<char>(*c));  // values are ASCII
    return narrow;
#else
    (void)name;
    return "";
#endif
}

// The copies of one value that exist, as text.
std::vector<std::string> copies(const std::string& directory, const nlohmann::json& state, const Place& place,
                                bool registry) {
    std::vector<std::string> found;
    found.push_back(trimmed(read_text_file(directory + place.file)));
    if (registry) found.push_back(trimmed(registry_text(place.value)));
    if (state.is_object() && state.contains(place.state) && state.at(place.state).is_string()) {
        found.push_back(trimmed(state.at(place.state).get<std::string>()));
    }
    found.erase(std::remove(found.begin(), found.end(), std::string()), found.end());
    return found;
}

bool parse_int(const std::string& text, long long& value) {
    errno = 0;
    char* end = nullptr;
    value = std::strtoll(text.c_str(), &end, 10);
    return errno == 0 && end != text.c_str() && *end == '\0';
}

Verdict fail(const std::string& reason) { return {false, reason}; }

}  // namespace

std::string read_text_file(const std::string& path) {
    std::ifstream stream(path, std::ios::binary);
    std::ostringstream content;
    content << stream.rdbuf();
    return content.str();
}

bool verify_server_time(const std::string& text, const std::vector<PublicKey>& keys, long long& time) {
    try {
        nlohmann::json payload;
        if (check_signed_bundle(nlohmann::json::parse(text), keys, payload) != BundleCheck::ok) return false;
        if (payload.value("purpose", "") != "server-time" || payload.value("version", 0) != 1) return false;
        time = payload.at("time").get<long long>();
        return time > 0;
    } catch (const std::exception&) {
        return false;
    }
}

ClockState read_clock_state(const std::string& directory, const std::vector<PublicKey>& keys, bool registry) {
    ClockState clock;
    nlohmann::json state;
    try {
        state = nlohmann::json::parse(read_text_file(directory + "license_state.json"));
    } catch (const std::exception&) {
        state = nlohmann::json::object();
    }
    long long value = 0;
    for (const auto& text : copies(directory, state, kAnchor, registry)) {
        if (parse_int(text, value)) clock.anchor = std::max(clock.anchor, value);
    }
    for (const auto& text : copies(directory, state, kServerTime, registry)) {
        if (verify_server_time(text, keys, value) && value > clock.confirmed_at) clock.confirmed_at = value;
    }
    for (const auto& text : copies(directory, state, kGraceStart, registry)) {
        if (parse_int(text, value) && value > 0 && (clock.grace_start == 0 || value < clock.grace_start)) {
            clock.grace_start = value;
        }
    }
    return clock;
}

Verdict verify_license(const std::string& text, const std::vector<PublicKey>& keys, const std::string& hwid,
                       long long now, const ClockState& clock, long long hwid_issued_before) {
    std::string type;
    long long expires = 0;
    try {  // read before the signature check only to pick the rules; verified just below
        auto bundle = nlohmann::json::parse(text);
        if (bundle.contains("certificate")) bundle = bundle.at("certificate");
        auto payload = nlohmann::json::parse(bundle.at("payload").get<std::string>());
        type = payload.value("type", "");
        expires = payload.value("expires_at", 0LL);
    } catch (const std::exception&) {
    }
    if (type != "trial") {  // lifetime: exactly the original rule
        return verify_certificate_text(text, keys, hwid, now, clock.anchor, hwid_issued_before);
    }
    const long long latest = std::max(clock.anchor, clock.confirmed_at);
    auto verdict = verify_certificate_text(text, keys, hwid, now, latest, hwid_issued_before, kTrialClockTolerance);
    if (!verdict.ok) return verdict;
    if (clock.confirmed_at > 0 && expires > 0 && clock.confirmed_at >= expires) return fail("The license has expired");
    const bool unconfirmed = clock.grace_start > 0 && clock.confirmed_at < clock.grace_start;
    if (unconfirmed && now >= clock.grace_start + kMissingAnchorGrace) {
        return fail("The trial needs one online check: its records on this computer were removed");
    }
    return verdict;
}

}  // namespace license
