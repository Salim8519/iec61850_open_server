#include "server/server.hpp"
#include "server/timer_policy.hpp"
#include "updates/values.hpp"
#include "controls/controls.hpp"
#include "license_guard.hpp"
#include <cstdio>
#include <iostream>
#ifdef _WIN32
#include <fcntl.h>
#include <io.h>
#endif

int main() {
#ifdef _WIN32
    _setmode(_fileno(stdin), _O_BINARY);
    _setmode(_fileno(stdout), _O_BINARY);
#endif
    protect_frame_output();
    // GOOSE MinTime and control/report timers on time without raising the system timer
    // resolution; the 1 ms timer only where Windows has no high-resolution timers (ADR-0070).
    TimerPolicy timers;
    Server server;
    Json message;
    try {
        while (read_frame(message)) {
            auto id = message.value("id", "");
            try {
                if (message.value("version", 0) != 1) throw std::runtime_error("Protocol version must be 1");
                auto op = message.at("op").get<std::string>();
                Json result;
                if (op == "start") {
                    license::require_valid_license();
                    result = server.start(message);
                }
                else if (op == "read") result = read_value(server, message.at("ref"));
                else if (op == "read_versioned") result = read_versioned(server, message.at("ref"));
                else if (op == "update_guarded") result = update_batch(server, message.at("updates"), message.at("guard"));
                else if (op == "update_batch") result = update_batch(server, message.at("updates"));
                else if (op == "update") {
                    update_value(server, message.at("ref"), message.at("value"));
                    result = read_value(server, message.at("ref"));
                } else if (op == "block") {
                    bool found = false;
                    for (auto& binding : server.controls) {
                        if (binding->reference == message.at("ref")) {
                            binding->blocked = message.at("blocked").get<bool>(); found = true;
                        }
                    }
                    if (!found) throw std::runtime_error("Unknown control binding");
                    result = true;
                } else if (op == "stop") {
                    server.stop(); emit_frame({{"id", id}, {"ok", true}}); break;
                } else throw std::runtime_error("Unknown operation");
                emit_frame({{"id", id}, {"ok", true}, {"result", result}});
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
