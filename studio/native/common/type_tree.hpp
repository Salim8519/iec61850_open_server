#pragma once
#include "browse.hpp"
#include <string>
#include <vector>

nlohmann::json discover_leaves(IedConnection connection, const std::string& reference,
                              const std::vector<std::string>& variables);
