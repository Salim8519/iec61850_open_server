#pragma once
#include <nlohmann/json.hpp>
extern "C" {
#include "mms_value.h"
#include "mms_type_spec.h"
}
#include <string>

// Convert one received scalar MMS value into typed JSON. Bit strings carry both
// bit orders because only the engineering type decides how Studio reads them.
nlohmann::json mms_scalar_json(const MmsValue* value);
// Flatten a received (possibly structured) value into reference -> scalar pairs,
// using the server-provided type description for member names.
void mms_flatten(const MmsValue* value, MmsVariableSpecification* spec,
                 const std::string& reference, nlohmann::json& output, int depth = 0);
