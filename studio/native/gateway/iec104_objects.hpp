#pragma once
#include "point_table.hpp"
#include <string>
extern "C" {
#include "cs101_information_objects.h"
}

// Build IEC 60870-5-104 information objects from gateway points. Interrogation
// answers use the types without time; spontaneous changes carry CP56Time2a.
InformationObject iec104_object(const GatewayPoint& point, bool with_time);
QualityDescriptor iec104_quality(const GatewayPoint& point);
std::string iec104_value_text(const GatewayPoint& point);
// Type names such as M_SP_TB_1 for the traffic log.
const char* iec104_type_name(int type_id);
