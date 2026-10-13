#pragma once
#include <nlohmann/json.hpp>
extern "C" {
#include "iec61850_client.h"
}

nlohmann::json browse_server(IedConnection connection);
