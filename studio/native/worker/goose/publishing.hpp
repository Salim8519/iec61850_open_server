#pragma once
#include "server/server.hpp"

// Configure layer-2 GOOSE publishing before IedServer_start. Returns the GOOSE
// readiness summary that the start reply carries back to the coordinator.
Json prepare_goose(Server& server, const Json& config);
// Enable configured GOOSE control blocks off the request thread, because opening
// a pcap adapter can take several seconds. Progress is reported as events.
void start_goose(Server& server);
