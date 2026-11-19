// Hand a SCADA command to the backend and wait for the MMS result.
#include "command_bridge.hpp"
#include <chrono>

CommandAnswer CommandBridge::ask(const std::string& point_id, bool value, const std::string& protocol,
                                 const std::string& peer, int timeout_ms) {
    return send({{"point_id", point_id}, {"value", value}, {"protocol", protocol}, {"peer", peer}}, timeout_ms);
}

CommandAnswer CommandBridge::ask_step(const std::string& point_id, const Json& value, const Json& wire,
                                      const std::string& protocol, const std::string& peer, int timeout_ms) {
    return send({{"point_id", point_id}, {"value", value}, {"protocol", protocol}, {"peer", peer}, {"wire", wire}},
                timeout_ms);
}

CommandAnswer CommandBridge::send(const Json& event, int timeout_ms) {
    const char* refused = "the gateway is stopping";
    std::unique_lock<std::mutex> guard(lock);
    if (stopping) return {false, refused};
    auto token = "cmd-" + std::to_string(++next);
    Json frame = event;
    frame["event"] = "gw_command";
    frame["token"] = token;
    emit_frame(frame);
    waiting.insert(token);
    changed.wait_for(guard, std::chrono::milliseconds(timeout_ms),
                     [&] { return answers.count(token) > 0 || stopping; });
    waiting.erase(token);
    auto found = answers.find(token);
    if (found == answers.end())
        return {false, stopping ? refused : "no answer from ScadaBench within the command timeout"};
    auto result = found->second;
    answers.erase(found);
    return result;
}

void CommandBridge::answer(const Json& request) {
    {
        std::lock_guard<std::mutex> guard(lock);
        auto token = request.at("token").get<std::string>();
        if (!waiting.count(token)) return;  // late: its waiter timed out or was cancelled
        answers[token] = {request.value("ok", false), request.value("reason", "")};
    }
    changed.notify_all();
}

void CommandBridge::cancel() {
    {
        std::lock_guard<std::mutex> guard(lock);
        stopping = true;
    }
    changed.notify_all();
}

void CommandBridge::resume() {
    std::lock_guard<std::mutex> guard(lock);
    stopping = false;
    answers.clear();
}
