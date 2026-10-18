#pragma once
#include "server/server.hpp"
#include <atomic>

struct ControlBinding {
    Server* server;
    std::string reference;
    std::string status;
    DataAttribute* feedback = nullptr;
    DataAttribute* timestamp = nullptr;
    std::string cdc;
    int model = 0;
    int sbo_timeout_ms = 0;
    std::atomic<bool> blocked{false};
};
void install_control(Server& server, const Json& config);
