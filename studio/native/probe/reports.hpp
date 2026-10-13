#pragma once
#include <nlohmann/json.hpp>
extern "C" {
#include "iec61850_client.h"
}

nlohmann::json receive_reports(IedConnection connection, const char* reference);
