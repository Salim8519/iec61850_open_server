// studio_control: operates controllable objects of real or simulated IEDs for the Tools
// place (IED Explorer and gateway pass-through). One command at a time; associations
// stay open between commands so SCADA commands are answered quickly.
#include "canvas_mux.hpp"
#include "control_session.hpp"
#include "license_guard.hpp"
#include <functional>
#include <iostream>
#include <map>

int main() {
    protect_frame_output();
    ControlSession session;
    CanvasMux canvas;
    const std::map<std::string, std::function<Json(const Json&)>> operations = {
        {"hello", [](const Json&) {
             return Json{{"capabilities", {"control_operate", "control_models_all", "control_mux_v1"}}};
         }},
        {"operate", [&](const Json& request) { license::require_valid_license(); return session.operate(request); }},
        {"close_all", [&](const Json&) { session.close_all(); return Json(true); }},
    };
    Json message;
    try {
        while (read_frame(message)) {
            auto id = message.value("id", "");
            try {
                if (message.value("version", 0) != 1) throw std::runtime_error("Protocol version must be 1");
                auto op = message.at("op").get<std::string>();
                if (op == "canvas_operate") {  // answered by the endpoint's lane, by request id
                    canvas.submit(message);
                    continue;
                }
                if (op == "stop") {
                    canvas.stop();
                    session.close_all();
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
