#pragma once
#include "ipc/framing.hpp"
#include <string>
extern "C" {
#include "cs101_information_objects.h"
}

// Monitoring ASDUs (M_xx) from the device as value records for the backend:
// {group, type, address, value, quality, time_ms, cause}. Other ASDUs return an empty array.
Json iec104_values(CS101_ASDU asdu);
// Type names such as M_DP_TB_1 and cause names such as "spontaneous" for logs and tables.
const char* iec104_name(int type_id);
std::string iec104_cause(int cot);
