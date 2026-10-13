#pragma once
#include "server/server.hpp"
Json read_value(Server& server, const std::string& reference);
void update_value(Server& server, const std::string& reference, const Json& value);
Json update_batch(Server& server, const Json& updates, const Json& guard = nullptr);
Json read_versioned(Server& server, const std::string& reference);
