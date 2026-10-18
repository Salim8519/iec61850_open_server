#pragma once
#include "ipc/framing.hpp"
#include <string>

// Events from the master worker to the backend. A value is one JSON object:
// {group, type, address, value, quality, time_ms, cause}; quality keeps the protocol's raw
// bits so the backend words them in one place.
void emit_values(const Json& values);
void emit_link(const std::string& state, const std::string& reason);
// A general interrogation or integrity poll finished (all values have been sent).
void emit_interrogation(const std::string& state);
std::string peer_name(const std::string& host, int port);
