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
struct TypedBinding;
struct Server {
    IedModel* model = nullptr;
    IedServer handle = nullptr;
    std::vector<std::unique_ptr<ControlBinding>> controls;
    std::vector<std::unique_ptr<TypedBinding>> typed_controls;  // ADR-0104
    std::unordered_map<DataAttribute*, uint64_t> revisions;
    // Resolved object references (IPC thread only; the model is fixed while running).
    std::unordered_map<std::string, DataAttribute*> attributes;
    std::string goose_interface;
    std::thread goose_thread;
    AccessLog access;
    EventLoop loop;
    // One TimeQuality for every timestamp this IED writes (ADR-0115): leap seconds known,
    // clock not synchronised (the PC clock is no IEC 61850 time source), 10-bit accuracy.
    static constexpr uint8_t time_quality = 0x80 | 0x20 | 10;
    uint64_t started_at = 0;  // Run start (ms since 1970): every "t" holds it until it changes
    Server();
    ~Server();
    Json start(const Json& config);
    DataAttribute* attribute(const std::string& reference);
    void stop();
};
