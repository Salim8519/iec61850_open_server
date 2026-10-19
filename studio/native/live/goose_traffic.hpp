#pragma once
#include "goose_inbox.hpp"
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

// Decodes the GOOSE frames the Studio GOOSE engine forwards (ADR-0073) for the
// configured control blocks and reports what was actually received, and whether it
// came from a simulated IED ("app") or the network ("wire"). It never publishes.
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
    GooseInbox inbox;
    std::mutex deliver_lock;  // one frame at a time through the receiver
    bool from_wire = false;   // origin of the frame being decoded, under deliver_lock
    uint8_t from_mac[6] = {}; // its source MAC; libiec61850 keeps it for observers only
    GooseReceiver receiver = nullptr;
    std::map<std::string, std::unique_ptr<GooseWatch>> watches;
    // Publishers seen on the wire but not in the project; listed once each, up to a limit.
    static constexpr size_t UNKNOWN_LIMIT = 500;
    std::set<std::string> unknown;
    bool unknown_capped = false;
    std::mutex lock;
    std::thread supervisor;
    std::atomic<bool> running{false};
};
