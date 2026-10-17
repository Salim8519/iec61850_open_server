// Thread-safe table of gateway points shared by the IPC thread and protocol threads.
#include "point_table.hpp"
#include <set>
#include <stdexcept>

namespace {
const std::set<std::string> KINDS = {"single", "double", "float", "integer", "command_single", "command_double"};
}

bool is_command(const std::string& kind) { return kind.rfind("command_", 0) == 0; }
bool quality_invalid(const GatewayPoint& point) { return !point.source_ok || (point.quality & 3) == 2; }
bool quality_questionable(const GatewayPoint& point) { return (point.quality & 3) == 3; }

void PointTable::load(const Json& items) {
    std::map<std::string, GatewayPoint> loaded;
    std::map<std::pair<std::string, int>, std::string> addresses;
    for (const auto& item : items) {
        GatewayPoint point;
        point.id = item.at("id").get<std::string>();
        point.kind = item.at("kind").get<std::string>();
        point.address = item.at("address").get<int>();
        point.monitor_only = item.value("monitor_only", false);
        point.scale = item.value("scale", 1.0);
        if (!KINDS.count(point.kind)) throw std::runtime_error("Unknown point kind " + point.kind);
        auto table = is_command(point.kind) ? std::string("command") : point.kind;
        if (!addresses.emplace(std::make_pair(table, point.address), point.id).second)
            throw std::runtime_error("Two points use address " + std::to_string(point.address) + " (" + table + ")");
        loaded[point.id] = point;
    }
    std::lock_guard<std::mutex> guard(lock);
    points = std::move(loaded);
    by_address = std::move(addresses);
}

std::vector<std::string> PointTable::update(const Json& values) {
    std::vector<std::string> changed;
    std::lock_guard<std::mutex> guard(lock);
    for (const auto& item : values) {
        auto found = points.find(item.at("id").get<std::string>());
        if (found == points.end()) continue;
        auto& point = found->second;
        const auto& raw = item.at("value");
        double value = raw.is_boolean() ? (raw.get<bool>() ? 1.0 : 0.0) : raw.get<double>();
        int quality = item.value("quality", 0);
        bool source_ok = item.value("source_ok", true);
        if (value == point.value && quality == point.quality && source_ok == point.source_ok) continue;
        point.value = value;
        point.quality = quality;
        point.source_ok = source_ok;
        point.time_ms = item.value("time_ms", int64_t(0));
        changed.push_back(point.id);
    }
    return changed;
}

std::vector<GatewayPoint> PointTable::all() const {
    std::lock_guard<std::mutex> guard(lock);
    std::vector<GatewayPoint> result;
    for (const auto& entry : points) result.push_back(entry.second);
    return result;
}

std::optional<GatewayPoint> PointTable::find(const std::string& id) const {
    std::lock_guard<std::mutex> guard(lock);
    auto found = points.find(id);
    if (found == points.end()) return std::nullopt;
    return found->second;
}

std::optional<GatewayPoint> PointTable::at(const std::string& table, int address) const {
    std::lock_guard<std::mutex> guard(lock);
    auto found = by_address.find({table, address});
    if (found == by_address.end()) return std::nullopt;
    return points.at(found->second);
}
