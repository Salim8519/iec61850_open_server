// Event-driven server loop of a simulated IED (workstream G). It replaces libiec61850's
// single-threaded server thread, which polled every 1 ms and blocked 10 ms on every idle
// MMS client per request. One thread still runs all MMS handling, connection ticks and
// periodic tasks, as before; it now sleeps until a socket is ready, a deadline of the
// vendor's own control/report/GOOSE schedule is due, or another thread signals it.
#pragma once
extern "C" {
#include "iec61850_server.h"
#include "studio_hal.h"
}
#include <atomic>
#include <cstdint>
#include <mutex>
#include <set>
#include <thread>

class EventLoop {
public:
    ~EventLoop();
    void start(IedServer server);
    void stop();
    // A data model update from another thread may have queued reports or GOOSE frames.
    void kick();
    // Keep the vendor's 1 ms tick for a while, after a change whose deadline is private.
    void poll_for(uint32_t milliseconds);

private:
    static constexpr uint32_t LONGEST_SLEEP_MS = 50;
    void run();
    uint64_t local_deadline(uint64_t now);  // GOOSE retransmissions and poll windows
    void frame_sent();
    IedServer server = nullptr;
    StudioWake wake = nullptr;
    std::thread thread;
    std::atomic<std::thread::id> loop_thread{};
    std::atomic<bool> running{false};
    std::mutex lock;                  // guards the members below
    std::multiset<uint64_t> goose_due;  // GOOSE retransmission candidates (Hal_getTimeInMs)
    uint64_t poll_until = 0;
    uint64_t last_run = 0;
};
