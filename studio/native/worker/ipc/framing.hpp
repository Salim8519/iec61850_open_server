#pragma once
#include <nlohmann/json.hpp>

using Json = nlohmann::json;
void protect_frame_output();
bool read_frame(Json& message);
void emit_frame(const Json& message);
