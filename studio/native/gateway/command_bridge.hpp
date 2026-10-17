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
    void answer(const Json& request);
private:
    std::mutex lock;
    std::condition_variable changed;
    std::map<std::string, CommandAnswer> answers;
    unsigned long next = 0;
};
