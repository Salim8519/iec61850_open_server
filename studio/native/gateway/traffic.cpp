// Rate-limited traffic and connection events toward the backend.
#include "traffic.hpp"
#include "ipc/framing.hpp"
#include <chrono>
#include <mutex>

namespace {
std::mutex lock;
long long window = 0;
int sent = 0;
int skipped = 0;
}

void traffic(const std::string& direction, const std::string& peer, const std::string& text) {
    auto now = std::chrono::duration_cast<std::chrono::seconds>(
        std::chrono::steady_clock::now().time_since_epoch()).count();
    int dropped = 0;
    {
        std::lock_guard<std::mutex> guard(lock);
        if (now != window) {
            window = now;
            dropped = skipped;
            sent = 0;
            skipped = 0;
        }
        if (sent >= TRAFFIC_LIMIT) {
            ++skipped;
            return;
        }
        ++sent;
    }
    Json event = {{"event", "gw_traffic"}, {"dir", direction}, {"peer", peer}, {"text", text}};
    if (dropped) event["skipped_before"] = dropped;
    emit_frame(event);
}

void connection_event(const std::string& state, const std::string& peer) {
    emit_frame({{"event", "gw_client"}, {"state", state}, {"peer", peer}});
}
