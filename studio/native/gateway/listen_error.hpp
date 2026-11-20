// Why the gateway could not listen, in words a user can act on (change 0227). Before, every
// failure said "Another program may use this port", also when the address had left this PC (for
// example a Lab-card address another ScadaBench copy took off), and users looked for a program
// that did not exist.
#pragma once
#include <string>

// The message for a listen on bind:port that failed with this socket error code (WSAGetLastError).
std::string listen_failure(const std::string& bind, int port, int code);

// Binds a probe socket to bind:port as lib60870's IEC 104 listener does (SO_REUSEADDR) and returns
// the socket error code, 0 when the bind works now. lib60870 keeps its own error to itself.
int listen_probe(const std::string& bind, int port);
