// Hand a SCADA command to the backend and wait for the MMS result.
#include "command_bridge.hpp"
#include <chrono>

CommandAnswer CommandBridge::ask(const std::string& point_id, bool value, const std::string& protocol,
                                 const std::string& peer, int timeout_ms) {
    std::unique_lock<std::mutex> guard(lock);
    auto token = "cmd-" + std::to_string(++next);
    emit_frame({{"event", "gw_command"}, {"token", token}, {"point_id", point_id}, {"value", value},
                {"protocol", protocol}, {"peer", peer}});
    bool done = changed.wait_for(guard, std::chrono::milliseconds(timeout_ms),
                                 [&] { return answers.count(token) > 0; });
    if (!done) return {false, "no answer from ScadaBench within the command timeout"};
    auto result = answers[token];
    answers.erase(token);
    return result;
}

void CommandBridge::answer(const Json& request) {
    {
        std::lock_guard<std::mutex> guard(lock);
        answers[request.at("token").get<std::string>()] = {request.value("ok", false), request.value("reason", "")};
    }
    changed.notify_all();
}
