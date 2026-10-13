#pragma once
#include "ipc/framing.hpp"
extern "C" {
#include "goose_receiver.h"
#include "goose_subscriber.h"
}
#include <atomic>
#include <map>
#include <memory>
#include <mutex>
#include <set>
#include <string>
#include <thread>

// Receives layer-2 GOOSE for the configured control blocks and reports what was
// actually seen on the wire. It never publishes and never changes any device.
struct GooseWatch {
    std::string key;
    std::string gocb_ref;
    uint32_t expected_conf_rev = 0;
    std::string expected_dataset;
    std::string expected_go_id;
    GooseSubscriber subscriber = nullptr;
    uint64_t frames = 0;
    uint32_t st_num = 0;
    uint32_t sq_num = 0;
    uint64_t last_rx_ms = 0;
    uint32_t tal_ms = 0;
    std::string state = "waiting";
    std::string problem;
};

class GooseTraffic {
public:
    ~GooseTraffic();
    Json start(const Json& request);
    void stop();
    void on_frame(GooseSubscriber subscriber, GooseWatch* watch);
    void on_observed(GooseSubscriber subscriber);
private:
    void supervise();
    GooseReceiver receiver = nullptr;
    std::map<std::string, std::unique_ptr<GooseWatch>> watches;
    std::set<std::string> unknown;
    std::mutex lock;
    std::thread supervisor;
    std::atomic<bool> running{false};
};
