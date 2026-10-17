#pragma once
#include "ipc/framing.hpp"
#include <map>
#include <string>
extern "C" {
#include "iec61850_client.h"
}

// IEC 61850 control client of ScadaBench (studio_control). It keeps one association per
// IED open, follows the device's own control model (direct or select-before-operate,
// normal or enhanced security) and reports the IED's AddCause when a command is refused.
// It is the only Studio program that operates real or simulated IEDs for the tools.
class ControlSession {
public:
    ~ControlSession();
    Json operate(const Json& request);
    void close_all();
private:
    std::map<std::string, IedConnection> connections;
    IedConnection connection(const std::string& ip, int port);
    void drop(const std::string& ip, int port);
};

const char* add_cause_text(int cause);
const char* control_model_name(int model);
