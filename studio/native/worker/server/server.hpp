#pragma once
#include "ipc/framing.hpp"
extern "C" {
#include "iec61850_server.h"
#include "iec61850_dynamic_model.h"
#include "hal_time.h"
}
#include <memory>
#include <condition_variable>
#include <deque>
#include <functional>
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
    std::mutex task_lock;
    std::condition_variable task_ready;
    std::deque<std::function<void()>> tasks;
    std::thread task_thread;
    bool tasks_stopping = false;
    std::string goose_interface;
    std::thread goose_thread;
    Server();
    ~Server();
    Json start(const Json& config);
    DataAttribute* attribute(const std::string& reference);
    void defer(std::function<void()> task);
    void stop();
private:
    void start_tasks();
    void stop_tasks();
};
