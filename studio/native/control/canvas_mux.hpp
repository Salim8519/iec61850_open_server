#pragma once
#include "ipc/framing.hpp"
#include <condition_variable>
#include <deque>
#include <map>
#include <memory>
#include <mutex>
#include <string>
#include <thread>
extern "C" {
#include "iec61850_client.h"
}

// Canvas commands of the Go engine ("control_mux_v1"): one lane (thread and association)
// per endpoint, so commands to different IEDs run in parallel and each reply carries its
// request id. The command on the wire is exactly what `studio_probe IP PORT control REF
// true|false` sends: no originator set, interlock/synchro/test off, select or
// select-with-value by the device's ctlModel, up to 100 x 20 ms for CommandTermination,
// and the reply is the probe's {"ok": bool}. `keep_ms` 0 closes the association after each
// command like the probe process did; a larger value keeps it for the next command.
class CanvasMux {
public:
    ~CanvasMux();
    void submit(const Json& request);
    void stop();
private:
    struct Lane {
        std::mutex lock;
        std::condition_variable wake;
        std::deque<Json> jobs;
        bool stopping = false;
        std::thread thread;
    };
    std::mutex lanes_lock;
    std::map<std::string, std::unique_ptr<Lane>> lanes;
    static void run(Lane& lane, std::string ip, int port);
};

// Probe-identical canvas command on an open association: {"ok": bool}.
Json canvas_command(IedConnection connection, const std::string& reference, bool value);
