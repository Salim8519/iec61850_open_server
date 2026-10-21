// Trial clock rules of the native guard, identical to license_policy.py / trial_clock.py
// (ADR-0046 addendum 2026-10-19): the anchor, the license server's signed time and the
// grace start are each read from three places (a dot-file beside license.cert, a REG_SZ
// under HKCU\Software\ScadaBench, and license_state.json); unreadable copies are ignored.
// The same three places hold the IDs of cancelled licenses (revoked_ids.py, ADR-0097):
// a certificate whose ID is in ANY copy is refused, whatever license.cert says.
#pragma once
#include "license_guard.hpp"
#include <set>
#include <string>
#include <vector>

namespace license {

constexpr long long kTrialClockTolerance = 48 * 3600;
constexpr long long kMissingAnchorGrace = 3 * 86400;

struct ClockState {
    long long anchor = 0;        // latest local time seen (max of the copies)
    long long confirmed_at = 0;  // newest genuine signed server time (0 = none)
    long long grace_start = 0;   // when every anchor copy was found missing (0 = never)
    std::set<std::string> revoked;  // IDs of cancelled licenses (union of every copy)
};

// Whole file as text ("" when missing or unreadable).
std::string read_text_file(const std::string& path);

// True and fills time when text is a genuine "server-time" stamp of one of keys.
bool verify_server_time(const std::string& text, const std::vector<PublicKey>& keys, long long& time);

// directory ends with a path separator. registry=false skips HKCU (test tool).
ClockState read_clock_state(const std::string& directory, const std::vector<PublicKey>& keys, bool registry);

// Certificate check plus the trial-only rules: a generous rollback tolerance against the
// anchors and the signed time, a trial ended by server time, and one online check within
// 3 days only after every anchor copy went missing. Server outages never stop a trial.
// A genuine certificate whose license ID is in clock.revoked is refused ("cancelled").
Verdict verify_license(const std::string& text, const std::vector<PublicKey>& keys, const std::string& hwid,
                       long long now, const ClockState& clock, long long hwid_issued_before = 0);

}  // namespace license
