#pragma once
#include <nlohmann/json.hpp>
extern "C" {
#include "iec61850_client.h"
}

nlohmann::json exercise_sbo(IedConnection connection, const std::string& action,
                            const std::string& reference, bool target, int wait_ms,
                            const std::string& origin = "");
