#pragma once
// Receives IEC 61850-9-2 Sampled Values for the streams in the SCD and reports, at
// most twice a second per stream, what was actually seen. It never publishes.
#include "ipc/framing.hpp"
#include "sv_header.hpp"
#include "sv_math.hpp"
extern "C" {
#include "sv_subscriber.h"
}
#include <atomic>
#include <condition_variable>
#include <map>
#include <memory>
#include <mutex>
#include <set>
#include <string>
#include <thread>
#include <vector>

struct SvWatch {
    std::string key, mac, layout;
    uint16_t app_id = 0;
    int per_cycle = 0;
    std::vector<std::string> channels;
    std::vector<double> scale;
    SVSubscriber subscriber = nullptr;
    SampleCounter counter;
    std::vector<ChannelCycle> cycles;
    uint64_t asdus = 0, window_asdus = 0, last_rx_ms = 0, window_start_ms = 0, invalid = 0;
    double rate = 0;
    std::string sv_id, dataset, state = "waiting";
    uint32_t conf_rev = 0;
    int smp_rate = -1, smp_mod = -1, smp_synch = -1, data_size = 0;
    std::vector<long long> raw;
};

class SvTraffic {
public:
    static constexpr size_t STREAM_LIMIT = 16;
    static constexpr size_t UNKNOWN_LIMIT = 500;
    ~SvTraffic();
    Json start(const Json& request);
    void stop();
    void on_asdu(SvWatch* watch, SVSubscriber_ASDU asdu);
private:
    void supervise();
    Json report(SvWatch& watch, uint64_t now, const std::map<std::string, SvHeader>& headers);
    SVReceiver receiver = nullptr;
    SvHeaderReader headers;
    std::map<std::string, std::unique_ptr<SvWatch>> watches;
    std::set<std::string> unknown;
    bool unknown_capped = false;
    std::mutex lock;
    std::thread supervisor;
    std::atomic<bool> running{false};
    std::mutex wake_lock;           // stop() wakes the supervisor's 500 ms wait at once
    std::condition_variable wake;
};
