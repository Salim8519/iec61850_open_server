// studio_master: ScadaBench as the SCADA master (ADR-0061). It connects to a real or
// simulated device (gateway, RTU or outstation) as IEC 104 controlling station, DNP3 master or Modbus client, reads
// every point and sends commands the backend asks for after the user confirmed them.
#include "client.hpp"
#include "license_guard.hpp"
#include "value_events.hpp"
#include <functional>
#include <iostream>
#include <map>
#include <thread>

int main() {
    protect_frame_output();
    std::unique_ptr<ProtocolClient> client;
    const std::map<std::string, std::function<std::unique_ptr<ProtocolClient>()>> factories = {
        {"iec104", [] { return make_client_104(); }},
        {"dnp3", [] { return make_client_dnp3(); }},
        {"modbus", [] { return make_client_modbus(); }},
    };
    auto connected = [&]() -> ProtocolClient& {
        if (!client) throw std::runtime_error("Not connected to the device; press Connect first");
        return *client;
    };
    // Find registers runs on its own thread and answers its request id when it ends, so
    // probe_stop (change 0178), polls and commands are still read meanwhile. A disconnect
    // first stops and waits for it, so the client never goes away under it.
    std::thread probing;
    auto end_probe = [&] {
        if (client) client->stop_probe();
        if (probing.joinable()) probing.join();
    };
    auto start_probe = [&](const std::string& id, const Json& request) {
        auto& target = connected();
        if (probing.joinable()) probing.join();
        probing = std::thread([&target, id, request] {
            try {
                emit_frame({{"id", id}, {"ok", true}, {"result", target.probe(request)}});
            } catch (const std::exception& error) {
                emit_frame({{"id", id}, {"ok", false}, {"error", error.what()}});
            }
        });
    };
    const std::map<std::string, std::function<Json(const Json&)>> operations = {
        {"hello", [](const Json&) { return Json{{"capabilities", {"master_iec104", "master_dnp3", "master_modbus"}}}; }},
        {"connect", [&](const Json& request) {
            license::require_valid_license();
            if (client) throw std::runtime_error("Already connected; disconnect first");
            auto found = factories.find(request.at("protocol").get<std::string>());
            if (found == factories.end()) throw std::runtime_error("This protocol is not available as master yet");
            auto candidate = found->second();
            auto opened = candidate->connect(request.at("settings"));
            client = std::move(candidate);
            return opened;
        }},
        {"interrogate", [&](const Json&) { return connected().interrogate(); }},
        {"poll", [&](const Json& request) { return connected().poll(request); }},
        {"probe_stop", [&](const Json&) {
            if (client) client->stop_probe();
            return Json(true);
        }},
        {"command", [&](const Json& request) { return connected().command(request); }},
        {"disconnect", [&](const Json&) {
            end_probe();
            if (client) client->disconnect();
            client.reset();
            return Json(true);
        }},
    };
    Json message;
    try {
        while (read_frame(message)) {
            auto id = message.value("id", "");
            try {
                if (message.value("version", 0) != 1) throw std::runtime_error("Protocol version must be 1");
                auto op = message.at("op").get<std::string>();
                if (op == "stop") {
                    end_probe();
                    if (client) client->disconnect();
                    client.reset();
                    emit_frame({{"id", id}, {"ok", true}});
                    break;
                }
                if (op == "probe") {
                    start_probe(id, message);
                    continue;
                }
                auto found = operations.find(op);
                if (found == operations.end()) throw std::runtime_error("Unknown operation " + op);
                emit_frame({{"id", id}, {"ok", true}, {"result", found->second(message)}});
            } catch (const std::exception& error) {
                emit_frame({{"id", id}, {"ok", false}, {"error", error.what()}});
            }
        }
    } catch (const std::exception& error) {
        end_probe();
        std::cerr << error.what() << std::endl;
        return 1;
    }
    end_probe();
    return 0;
}
