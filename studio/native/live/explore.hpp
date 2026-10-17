#pragma once
#include "mms_session.hpp"

// IED Explorer: browse one level of the connected server at a time, so large models
// open quickly. Read-only: only directory and type-description services are used.
Json explore_level(MmsSession& session, const Json& request);
