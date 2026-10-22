// studio_gateway: the gateway simulator worker. It serves gateway points to a SCADA
// master over IEC 60870-5-104, Modbus TCP or DNP3. It has no IEC 61850 client: values
// arrive from the backend, and every command goes back to the backend, which only
// operates simulated IEDs (ADR-0057).
#include "license_guard.hpp"
#include "protocol.hpp"
#include <functional>
#include <iostream>
#include <map>

int main() {
    protect_frame_output();
    PointTable table;
    CommandBridge bridge;
    std::unique_ptr<ProtocolServer> server;
    std::string protocol;
    const std::map<std::string, std::function<std::unique_ptr<ProtocolServer>()>> factories = {
        {"iec104", [&] { return make_iec104(table, bridge); }},
        {"modbus", [&] { return make_modbus(table, bridge); }},
        {"dnp3", [&] { return make_dnp3(table, bridge); }},
    };
    const std::map<std::string, std::function<Json(const Json&)>> operations = {
        {"hello", [](const Json&) {
            return Json{{"capabilities", {"gateway_iec104", "gateway_modbus", "gateway_dnp3", "commands_via_backend",
                                             "typed_commands"}}};
        }},
        {"start", [&](const Json& request) {
            license::require_valid_license();
            if (server) throw std::runtime_error("The gateway is already running");
            auto found = factories.find(request.at("protocol").get<std::string>());
            if (found == factories.end()) throw std::runtime_error("Unknown gateway protocol");
            table.load(request.at("points"));
            bridge.typed = request.value("typed_commands", false);  // ADR-0107; absent = the Python-era channel
            auto candidate = found->second();
            auto opened = candidate->start(request.value("settings", Json::object()));
            server = std::move(candidate);
            protocol = found->first;
            return opened;
        }},
        {"update", [&](const Json& request) {
            auto changed = table.update(request.at("values"));
            if (server && !changed.empty()) {
                std::vector<GatewayPoint> points;
                for (const auto& id : changed) points.push_back(*table.find(id));
                server->changed(points);
            }
            return Json{{"changed", changed.size()}};
        }},
        {"command_result", [&](const Json& request) { bridge.answer(request); return Json(true); }},
        {"halt", [&](const Json&) {
            if (server) server->stop();
            server.reset();
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
                    if (server) server->stop();
                    emit_frame({{"id", id}, {"ok", true}});
                    break;
                }
                auto found = operations.find(op);
                if (found == operations.end()) throw std::runtime_error("Unknown operation " + op);
                emit_frame({{"id", id}, {"ok", true}, {"result", found->second(message)}});
            } catch (const std::exception& error) {
                emit_frame({{"id", id}, {"ok", false}, {"error", error.what()}});
            }
        }
    } catch (const std::exception& error) {
        std::cerr << error.what() << std::endl;
        return 1;
    }
    return 0;
}
