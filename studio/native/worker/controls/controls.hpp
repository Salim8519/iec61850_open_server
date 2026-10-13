#pragma once
#include "server/server.hpp"
#include <atomic>
#include <mutex>

struct ControlBinding {
    Server* server;
    std::string reference;
    std::string status;
    DataAttribute* feedback = nullptr;
    DataAttribute* timestamp = nullptr;
    std::string cdc;
    int model = 0;
    int sbo_timeout_ms = 0;
    std::string compatibility = "standard";
    std::mutex compatibility_lock;
    ClientConnection pending_client = nullptr;
    int pending_ctl_num = -1;
    bool pending_raw_target = false;
    bool pending_compatibility = false;
    bool pending_applied = false;
    std::atomic<bool> blocked{false};
};
void install_control(Server& server, const Json& config);
