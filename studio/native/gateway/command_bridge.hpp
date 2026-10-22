#pragma once
#include "ipc/framing.hpp"
#include <condition_variable>
#include <map>
#include <mutex>
#include <string>

// A SCADA command waits here while the backend operates the simulated IED over MMS.
// The protocol thread blocks for at most the timeout, so commands of one connection
// are handled one at a time, as in a real gateway.
struct CommandAnswer {
    bool ok = false;
    std::string reason;
};

class CommandBridge {
public:
    CommandAnswer ask(const std::string& point_id, bool value, const std::string& protocol,
                      const std::string& peer, int timeout_ms = 15000);
    // ADR-0107: a command step with what the outstation decoded ("wire": type, select bit,
    // cause, DNP3 function ...). The engine's routing table decides select / execute / cancel.
    CommandAnswer ask_step(const std::string& point_id, const Json& value, const Json& wire,
                           const std::string& protocol, const std::string& peer, int timeout_ms = 15000);
    void answer(const Json& request);
    // True when the engine asked for every command step ("typed_commands" in start); false
    // keeps the outstation's own select/cancel confirmation of the Python-era channel.
    bool typed = false;
private:
    CommandAnswer send(const Json& event, int timeout_ms);
    std::mutex lock;
    std::condition_variable changed;
    std::map<std::string, CommandAnswer> answers;
    unsigned long next = 0;
};
