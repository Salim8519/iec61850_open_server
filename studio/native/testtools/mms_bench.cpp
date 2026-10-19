// Test-only MMS load client (never packaged): measures per-request latency of a simulated
// IED while other MMS clients are connected. Used for the workstream G evidence (multi-client
// latency, control round trip). Each client is an independent libiec61850 connection.
//
// usage: studio_test_mms_bench IP PORT read REF FC ACTIVE IDLE COUNT
//        studio_test_mms_bench IP PORT control REF - ACTIVE IDLE COUNT
// ACTIVE clients issue COUNT requests back to back; IDLE clients only stay connected.
// control: each active client operates REF (Boolean ctlVal, toggled) and, for enhanced
// security, waits for the CommandTermination; the time covers select, operate and CmdTerm.
#include <nlohmann/json.hpp>
#include <algorithm>
#include <atomic>
#include <chrono>
#include <condition_variable>
#include <iostream>
#include <mutex>
#include <string>
#include <thread>
#include <vector>
extern "C" {
#include "iec61850_client.h"
}
using Json = nlohmann::json;
using Clock = std::chrono::steady_clock;

namespace {
struct Termination {
    std::mutex lock;
    std::condition_variable done;
    int state = 0;
};

void terminated(void* parameter, ControlObjectClient control) {
    auto termination = static_cast<Termination*>(parameter);
    auto error = ControlObjectClient_getLastApplError(control);
    std::lock_guard<std::mutex> guard(termination->lock);
    termination->state = error.error == CONTROL_ERROR_NO_ERROR ? 1 : -1;
    termination->done.notify_all();
}

IedConnection connect(const char* ip, int port) {
    auto connection = IedConnection_create();
    IedClientError error;
    IedConnection_connect(connection, &error, ip, port);
    if (error != IED_ERROR_OK) {
        IedConnection_destroy(connection);
        return nullptr;
    }
    return connection;
}

bool operate_once(ControlObjectClient control, bool target, Termination& termination) {
    auto value = MmsValue_newBoolean(target);
    auto model = ControlObjectClient_getControlModel(control);
    {
        std::lock_guard<std::mutex> guard(termination.lock);
        termination.state = 0;
    }
    bool ok = true;
    if (model == CONTROL_MODEL_SBO_NORMAL) ok = ControlObjectClient_select(control);
    if (model == CONTROL_MODEL_SBO_ENHANCED) ok = ControlObjectClient_selectWithValue(control, value);
    ok = ok && ControlObjectClient_operate(control, value, 0);
    if (ok && (model == CONTROL_MODEL_SBO_ENHANCED || model == CONTROL_MODEL_DIRECT_ENHANCED)) {
        std::unique_lock<std::mutex> guard(termination.lock);
        ok = termination.done.wait_for(guard, std::chrono::seconds(2), [&] { return termination.state != 0; })
            && termination.state == 1;
    }
    MmsValue_delete(value);
    return ok;
}

Json summary(std::vector<double> samples) {
    if (samples.empty()) return {{"count", 0}};
    std::sort(samples.begin(), samples.end());
    auto at = [&](double q) { return samples[std::min(samples.size() - 1, size_t(q * samples.size()))]; };
    double total = 0;
    for (double sample : samples) total += sample;
    return {{"count", samples.size()}, {"mean_ms", total / samples.size()}, {"p50_ms", at(0.50)},
            {"p95_ms", at(0.95)}, {"p99_ms", at(0.99)}, {"max_ms", samples.back()}};
}
}

int main(int argc, char** argv) {
    if (argc < 9) {
        std::cerr << "usage: studio_test_mms_bench IP PORT read|control REF FC ACTIVE IDLE COUNT\n";
        return 2;
    }
    const char* ip = argv[1];
    int port = std::stoi(argv[2]);
    std::string mode = argv[3], reference = argv[4];
    auto fc = FunctionalConstraint_fromString(argv[5]);
    int active = std::stoi(argv[6]), idle = std::stoi(argv[7]), count = std::stoi(argv[8]);
    std::vector<IedConnection> idlers;
    for (int i = 0; i < idle; ++i) {
        auto connection = connect(ip, port);
        if (!connection) { std::cout << Json({{"ok", false}, {"error", "idle connect failed"}}).dump(); return 1; }
        idlers.push_back(connection);
    }
    std::vector<std::vector<double>> latencies(active);
    std::atomic<int> ready{0}, failures{0};
    std::atomic<bool> go{false};
    std::vector<std::thread> threads;
    for (int index = 0; index < active; ++index) {
        threads.emplace_back([&, index] {
            auto connection = connect(ip, port);
            if (!connection) { failures++; ready++; return; }
            ControlObjectClient control = nullptr;
            Termination termination;
            if (mode == "control") {
                control = ControlObjectClient_create(reference.c_str(), connection);
                if (control) ControlObjectClient_setCommandTerminationHandler(control, terminated, &termination);
            }
            ready++;
            while (!go) std::this_thread::yield();
            for (int i = 0; i < count; ++i) {
                IedClientError error = IED_ERROR_OK;
                auto start = Clock::now();
                bool ok;
                if (mode == "control") {
                    ok = control && operate_once(control, i % 2 == 0, termination);
                } else {
                    auto value = IedConnection_readObject(connection, &error, reference.c_str(), fc);
                    ok = value && error == IED_ERROR_OK;
                    if (value) MmsValue_delete(value);
                }
                std::chrono::duration<double, std::milli> took = Clock::now() - start;
                if (!ok) failures++;
                latencies[index].push_back(took.count());
            }
            if (control) ControlObjectClient_destroy(control);
            IedConnection_close(connection);
            IedConnection_destroy(connection);
        });
    }
    while (ready < active) std::this_thread::sleep_for(std::chrono::milliseconds(1));
    auto start = Clock::now();
    go = true;
    for (auto& thread : threads) thread.join();
    std::chrono::duration<double, std::milli> wall = Clock::now() - start;
    for (auto connection : idlers) { IedConnection_close(connection); IedConnection_destroy(connection); }
    std::vector<double> all;
    Json clients = Json::array();
    for (auto& samples : latencies) {
        all.insert(all.end(), samples.begin(), samples.end());
        clients.push_back(summary(samples));
    }
    std::cout << Json({{"ok", failures == 0}, {"failures", failures.load()}, {"mode", mode},
                       {"active", active}, {"idle", idle}, {"wall_ms", wall.count()},
                       {"all", summary(all)}, {"clients", clients}}).dump() << std::endl;
    return failures == 0 ? 0 : 1;
}
