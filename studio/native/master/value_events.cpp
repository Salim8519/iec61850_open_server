// Master worker events: batched values and link state. Traffic lines reuse gateway/traffic.cpp.
#include "value_events.hpp"

void emit_values(const Json& values) {
    if (!values.empty()) emit_frame({{"event", "master_values"}, {"values", values}});
}

void emit_link(const std::string& state, const std::string& reason) {
    emit_frame({{"event", "master_link"}, {"state", state}, {"reason", reason}});
}

void emit_interrogation(const std::string& state) {
    emit_frame({{"event", "master_interrogation"}, {"state", state}});
}

std::string peer_name(const std::string& host, int port) {
    return host + ":" + std::to_string(port);
}
