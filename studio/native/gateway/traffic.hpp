#pragma once
#include <string>

// Decoded one-line summaries of protocol traffic for the traffic log. At most
// TRAFFIC_LIMIT lines per second are sent; the rest are counted as skipped.
constexpr int TRAFFIC_LIMIT = 200;
void traffic(const std::string& direction, const std::string& peer, const std::string& text);
void connection_event(const std::string& state, const std::string& peer);
