#pragma once
#include <nlohmann/json.hpp>
extern "C" {
#include "iec61850_client.h"
}

#include <string>
#include <vector>

// Whole model (live acquisition and probe), or one level at a time (IED Explorer).
nlohmann::json browse_server(IedConnection connection);
// Logical devices of the server when device is empty, else logical nodes of that device.
std::vector<std::string> browse_names(IedConnection connection, const std::string& device);
// Leaves, data objects, datasets and report blocks of one logical node "LD/LN".
nlohmann::json browse_node(IedConnection connection, const std::string& reference);
