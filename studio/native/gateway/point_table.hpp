#pragma once
#include "ipc/framing.hpp"
#include <cstdint>
#include <map>
#include <mutex>
#include <optional>
#include <string>
#include <vector>

// One gateway point. Values come from the backend (simulated or real IEDs); the
// quality keeps IEC 61850 bits so each protocol maps them in one place.
struct GatewayPoint {
    std::string id;
    std::string kind;  // single, double, float, integer, command_single, command_double
    int address = 0;
    bool monitor_only = false;
    double scale = 1.0;
    double value = 0.0;
    int quality = 0;          // IEC 61850 quality bits (validity in bits 0-1)
    bool source_ok = false;   // false until a value arrives, or while the IED is stopped
    int64_t time_ms = 0;
};

bool is_command(const std::string& kind);
// IEC 61850 validity: 0 good, 2 invalid, 3 questionable. No source counts as invalid.
bool quality_invalid(const GatewayPoint& point);
bool quality_questionable(const GatewayPoint& point);

class PointTable {
public:
    void load(const Json& points);
    // Apply {id, value, quality, time_ms, source_ok} items; returns the ids whose state changed.
    std::vector<std::string> update(const Json& values);
    std::vector<GatewayPoint> all() const;
    std::optional<GatewayPoint> find(const std::string& id) const;
    // table is the point kind for monitoring points and "command" for both command kinds.
    std::optional<GatewayPoint> at(const std::string& table, int address) const;
private:
    mutable std::mutex lock;
    std::map<std::string, GatewayPoint> points;
    std::map<std::pair<std::string, int>, std::string> by_address;
};
