#pragma once
#include "ipc/framing.hpp"
#include <chrono>
#include <condition_variable>
#include <map>
#include <memory>
#include <mutex>
#include <string>
extern "C" {
#include "iec61850_client.h"
}

// Typed commands of the Go engine (ADR-0104): Select, Operate and Cancel as separate steps
// with a typed ctlVal (bool, int, enum, float or Tcmd step) and the command options (test,
// interlock/synchro check, originator). A select is kept per endpoint and reference until it
// is operated, canceled or its sboTimeout passes. It has its own associations, so the
// canvas lanes and the Tools `operate` keep exactly their wire behaviour.
class TypedSession {
public:
    ~TypedSession();
    Json select(const Json& request);
    Json operate(const Json& request);
    Json cancel(const Json& request);
    void close_all();

    struct Termination {
        std::mutex lock;
        std::condition_variable arrived;
        int state = 0;  // 0 waiting, 1 success, -1 failed
        int cause = 0;
    };
    struct Control {
        ControlObjectClient handle = nullptr;
        std::unique_ptr<Termination> done = std::make_unique<Termination>();
        std::string endpoint;
        int model = 0;
        int sbo_timeout_ms = 0;
        bool selected = false;
        std::chrono::steady_clock::time_point expires;
    };
private:
    std::map<std::string, IedConnection> connections;
    std::map<std::string, std::unique_ptr<Control>> selections;  // "ip:port|ref"
    IedConnection connection(const std::string& ip, int port);
    void drop(const std::string& endpoint);
    std::unique_ptr<Control> open(const Json& request, IedConnection handle, const std::string& endpoint);
    void forget(const std::string& key);
};

// The ctlVal of one typed command, checked against the type the device declares.
MmsValue* typed_control_value(ControlObjectClient control, const Json& value, const std::string& kind);
const char* typed_model_name(int model);
