#pragma once
#include "ipc/framing.hpp"
#include "access_log.hpp"
#include "event_loop.hpp"
extern "C" {
#include "iec61850_server.h"
#include "iec61850_dynamic_model.h"
#include "hal_time.h"
}
#include <memory>
#include <mutex>
#include <thread>
#include <string>
#include <vector>
#include <unordered_map>

struct ControlBinding;
struct Server {
    IedModel* model = nullptr;
    IedServer handle = nullptr;
    std::vector<std::unique_ptr<ControlBinding>> controls;
    std::unordered_map<DataAttribute*, uint64_t> revisions;
    std::string goose_interface;
    std::thread goose_thread;
    AccessLog access;
    EventLoop loop;
    Server();
    ~Server();
    Json start(const Json& config);
    DataAttribute* attribute(const std::string& reference);
    void stop();
};
