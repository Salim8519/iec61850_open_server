#include "event_loop.hpp"
#include "schedule.h"
extern "C" {
#include "hal_time.h"
}
#include <algorithm>
#include <chrono>
#include <stdexcept>

EventLoop::~EventLoop() { stop(); }

void EventLoop::start(IedServer handle) {
    if (running) return;
    wake = StudioWake_create();
    if (!wake) throw std::runtime_error("Unable to create the IED server wake-up socket");
    server = handle;
    StudioEthernet_setSendObserver([](void* self) { static_cast<EventLoop*>(self)->frame_sent(); }, this);
    running = true;
    thread = std::thread([this] { run(); });
}

void EventLoop::stop() {
    if (!running.exchange(false)) return;
    StudioWake_signal(wake);
    if (thread.joinable()) thread.join();
    StudioEthernet_setSendObserver(nullptr, nullptr);
    StudioWake_destroy(wake);
    wake = nullptr;
    server = nullptr;
}

void EventLoop::kick() {
    if (running && std::this_thread::get_id() != loop_thread.load()) StudioWake_signal(wake);
}

void EventLoop::poll_for(uint32_t milliseconds) {
    {
        std::lock_guard<std::mutex> guard(lock);
        poll_until = std::max(poll_until, Hal_getTimeInMs() + milliseconds);
    }
    kick();
}

// libiec61850 schedules the next frame of a GOOSE control block at send time + MinTime
// or + MaxTime (mms_goose.c), measured just before this observer runs. Waking one
// millisecond either side of each candidate keeps retransmissions on time.
void EventLoop::frame_sent() {
    uint32_t intervals[16];
    int count = studio_goose_intervals(server, intervals, 16);
    uint64_t now = Hal_getTimeInMs();
    {
        std::lock_guard<std::mutex> guard(lock);
        for (int i = 0; i < count; ++i) {
            for (uint64_t offset = 0; offset < 3; ++offset)
                goose_due.insert(now + intervals[i] + offset - 1);
        }
    }
    kick();
}

uint64_t EventLoop::local_deadline(uint64_t now) {
    if (StudioSocket_takeBackpressure()) poll_for(20);  // vendor retries partial sends per tick
    std::lock_guard<std::mutex> guard(lock);
    goose_due.erase(goose_due.begin(), goose_due.upper_bound(last_run));
    uint64_t next = goose_due.empty() ? STUDIO_NO_DEADLINE : *goose_due.begin();
    if (now < poll_until) next = std::min(next, now + 1);
    return next;
}

void EventLoop::run() {
    loop_thread.store(std::this_thread::get_id());
    uint64_t left_by_periodic = STUDIO_NO_DEADLINE;
    while (running) {
        uint64_t now = Hal_getTimeInMs();
        uint64_t server_next = studio_server_deadline(server, now);
        uint64_t local_next = local_deadline(now);
        uint64_t next = std::min(server_next, local_next);
        unsigned int timeout = static_cast<unsigned int>(
            next <= now ? 0 : std::min<uint64_t>(next - now, LONGEST_SLEEP_MS));
        // A deadline the last periodic run left unchanged is one the vendor waits out itself
        // (strict select timeout, model locked by an update): retry each millisecond like its
        // own loop instead of spinning.
        if (timeout == 0 && server_next <= now && server_next == left_by_periodic && local_next > now)
            timeout = 1;
        StudioSocket_beginWait(wake);
        int ready = IedServer_waitReady(server, timeout);
        StudioSocket_beginPoll();
        {
            std::lock_guard<std::mutex> guard(lock);
            last_run = Hal_getTimeInMs();
        }
        // As the vendor loop: requests first, then periodic tasks, so an operate or a select
        // with value runs in the same pass. The second pass's connection ticks send the reports
        // the periodic tasks queued without waiting a millisecond for the next pass.
        IedServer_processIncomingData(server);
        IedServer_performPeriodicTasks(server);
        left_by_periodic = studio_server_deadline(server, Hal_getTimeInMs());
        IedServer_processIncomingData(server);
        StudioSocket_endLoopMode();
        if (ready < 0 && running) std::this_thread::sleep_for(std::chrono::milliseconds(timeout));
    }
}
