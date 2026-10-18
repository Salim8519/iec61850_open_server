// Aggregates MMS access per client and emits `mms_access` at most once per second.
// Handlers only count; they never change what the MMS server answers (ADR-0066).
#include "access_log.hpp"
#include "ipc/framing.hpp"
extern "C" {
#include "iec61850_server.h"
}
#include <chrono>
#include <cstring>

AccessLog::~AccessLog() { stop(); }

void AccessLog::start() {
    if (running.exchange(true)) return;
    flusher = std::thread([this] {
        std::unique_lock<std::mutex> guard(lock);
        while (running) {
            wake.wait_for(guard, std::chrono::milliseconds(1000));
            guard.unlock();
            flush();
            guard.lock();
        }
    });
}

void AccessLog::stop() {
    if (!running.exchange(false)) return;
    wake.notify_all();
    if (flusher.joinable()) flusher.join();
    std::lock_guard<std::mutex> guard(lock);
    clients.clear();
    dirty = false;
}

AccessLog::Client* AccessLog::client(const std::string& peer) {
    auto found = clients.find(peer);
    if (found != clients.end()) return &found->second;
    if (clients.size() >= MAX_CLIENTS) return nullptr;
    return &clients[peer];
}

void AccessLog::connection(const std::string& peer, bool connected) {
    std::lock_guard<std::mutex> guard(lock);
    auto entry = client(peer);
    if (!entry) return;
    entry->connected = connected;
    if (!connected) entry->rcbs.clear();
    dirty = true;
}

void AccessLog::read(const char* domain, const char* item, const char* peer) {
    // MMS item "LN$FC$DO$DA": the logical node is everything before the first '$'.
    const char* end = std::strchr(item, '$');
    std::string node = std::string(domain) + "/" + (end ? std::string(item, end - item) : std::string(item));
    std::lock_guard<std::mutex> guard(lock);
    auto entry = client(peer);
    if (!entry) return;
    entry->total_reads++;
    if (entry->reads.size() < MAX_NODES || entry->reads.count(node)) entry->reads[node]++;
    else entry->reads["(other)"]++;
    dirty = true;
}

void AccessLog::write(const std::string& peer, const std::string& object) {
    std::lock_guard<std::mutex> guard(lock);
    auto entry = client(peer);
    if (!entry) return;
    if (entry->writes.size() < MAX_NODES || entry->writes.count(object)) entry->writes[object]++;
    dirty = true;
}

void AccessLog::report(const std::string& peer, const std::string& rcb, int event) {
    if (event != RCB_EVENT_ENABLE && event != RCB_EVENT_DISABLE && event != RCB_EVENT_UNRESERVED) return;
    std::lock_guard<std::mutex> guard(lock);
    auto entry = client(peer);
    if (!entry) return;
    if (event == RCB_EVENT_ENABLE) entry->rcbs.insert(rcb);
    else entry->rcbs.erase(rcb);
    dirty = true;
}

void AccessLog::flush() {
    Json summary = Json::array();
    {
        std::lock_guard<std::mutex> guard(lock);
        if (!dirty) return;
        for (auto it = clients.begin(); it != clients.end();) {
            auto& entry = it->second;
            summary.push_back({{"peer", it->first}, {"connected", entry.connected},
                               {"reads", entry.reads}, {"total_reads", entry.total_reads},
                               {"writes", entry.writes}, {"rcbs", entry.rcbs}});
            entry.reads.clear();
            entry.writes.clear();
            it = entry.connected ? std::next(it) : clients.erase(it);
        }
        dirty = false;
    }
    emit_frame({{"event", "mms_access"}, {"clients", summary}});
}
