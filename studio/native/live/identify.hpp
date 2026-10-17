#pragma once
#include "ipc/framing.hpp"

// Find IEDs: open a short association of its own, ask the MMS identity and the logical
// devices, and close again. It never reads data values or writes anything.
Json identify_server(const Json& request);
