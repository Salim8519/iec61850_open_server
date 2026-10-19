// Endpoint lanes of the multiplexed canvas mode (see canvas_mux.hpp).
#include "canvas_mux.hpp"
#include "license_guard.hpp"
#include <chrono>
#include <stdexcept>

namespace {
struct Association {
    IedConnection handle = nullptr;
    std::chrono::steady_clock::time_point idle_until;
    void close() {
        if (!handle) return;
        IedConnection_close(handle);
        IedConnection_destroy(handle);
        handle = nullptr;
    }
};

// Same defaults as studio_probe: IedConnection_create with its own connect/request timeouts.
IedConnection open(const std::string& ip, int port) {
    auto connection = IedConnection_create();
    IedClientError error = IED_ERROR_OK;
    IedConnection_connect(connection, &error, ip.c_str(), port);
    if (error != IED_ERROR_OK) {
        IedConnection_destroy(connection);
        throw std::runtime_error("Could not connect to " + ip + ":" + std::to_string(port) +
                                 " (MMS error " + std::to_string(int(error)) + ")");
    }
    return connection;
}

void reply(const Json& request, const Json& result) {
    emit_frame({{"id", request.value("id", "")}, {"ok", true}, {"result", result}});
}

void refuse(const Json& request, const std::string& error) {
    emit_frame({{"id", request.value("id", "")}, {"ok", false}, {"error", error}});
}
}

CanvasMux::~CanvasMux() { stop(); }

void CanvasMux::submit(const Json& request) {
    auto ip = request.at("ip").get<std::string>();
    int port = request.value("port", 102);
    auto key = ip + ":" + std::to_string(port);
    std::lock_guard<std::mutex> guard(lanes_lock);
    auto& lane = lanes[key];
    if (!lane) {
        lane = std::make_unique<Lane>();
        lane->thread = std::thread(&CanvasMux::run, std::ref(*lane), ip, port);
    }
    {
        std::lock_guard<std::mutex> jobs(lane->lock);
        lane->jobs.push_back(request);
    }
    lane->wake.notify_one();
}

void CanvasMux::stop() {
    std::lock_guard<std::mutex> guard(lanes_lock);
    for (auto& entry : lanes) {
        {
            std::lock_guard<std::mutex> jobs(entry.second->lock);
            entry.second->stopping = true;
        }
        entry.second->wake.notify_one();
    }
    for (auto& entry : lanes)
        if (entry.second->thread.joinable()) entry.second->thread.join();
    lanes.clear();
}

void CanvasMux::run(Lane& lane, std::string ip, int port) {
    Association association;
    for (;;) {
        Json request;
        {
            std::unique_lock<std::mutex> guard(lane.lock);
            while (lane.jobs.empty() && !lane.stopping) {
                if (association.handle) {
                    if (lane.wake.wait_until(guard, association.idle_until) == std::cv_status::timeout &&
                        lane.jobs.empty()) {
                        guard.unlock();
                        association.close();
                        guard.lock();
                    }
                } else {
                    lane.wake.wait(guard);
                }
            }
            if (lane.jobs.empty()) break;  // stopping and drained
            request = std::move(lane.jobs.front());
            lane.jobs.pop_front();
        }
        try {
            license::require_valid_license();
            if (association.handle && IedConnection_getState(association.handle) != IED_STATE_CONNECTED)
                association.close();
            if (!association.handle) association.handle = open(ip, port);
            auto result = canvas_command(association.handle, request.at("ref").get<std::string>(),
                                         request.at("value").get<bool>());
            // keep_ms 0: release before replying, as the probe process ended before Python went on,
            // so the IED's disconnect event comes before the feedback writes (same event order).
            int keep_ms = request.value("keep_ms", 0);
            if (keep_ms <= 0) association.close();
            else association.idle_until = std::chrono::steady_clock::now() + std::chrono::milliseconds(keep_ms);
            reply(request, result);
        } catch (const std::exception& error) {
            association.close();
            refuse(request, error.what());
        }
    }
    association.close();
}
