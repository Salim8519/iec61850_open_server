// studio_live: observe external IEDs and GOOSE traffic for Studio. It is a
// read-only client by construction: the operation table below has no control,
// setting or data write. Report-block reservation is the single exception.
#include "explore.hpp"
#include "goose_traffic.hpp"
#include "identify.hpp"
#include "license_guard.hpp"
#include "mms_session.hpp"
#include "npcap.hpp"
#include "reports.hpp"
#include "sv_traffic.hpp"
#include <functional>
#include <iostream>
#include <map>

int main() {
    protect_frame_output();
    MmsSession session;
    ReportAccess reports(session);
    GooseTraffic goose;
    SvTraffic sv;
    const std::map<std::string, std::function<Json(const Json&)>> operations = {
        {"hello", [](const Json&) {
            return Json{{"capabilities", {"live_discover", "live_reports", "live_poll",
                                          "goose_listen", "goose_inbox", "sv_listen", "read_only_client",
                                          "explore", "identify"}}};
        }},
        {"npcap_status", [](const Json&) { return npcap_status(); }},
        {"adapters", [](const Json&) { return Json{{"adapters", npcap_adapters()}}; }},
        {"goose_start", [&](const Json& request) { license::require_valid_license(); return goose.start(request); }},
        {"goose_stop", [&](const Json&) { goose.stop(); return Json(true); }},
        {"sv_start", [&](const Json& request) { license::require_valid_license(); return sv.start(request); }},
        {"sv_stop", [&](const Json&) { sv.stop(); return Json(true); }},
        {"connect", [&](const Json& request) { license::require_valid_license(); return session.connect(request); }},
        {"discover", [&](const Json&) { return session.discover(); }},
        {"explore", [&](const Json& request) { return explore_level(session, request); }},
        {"identify", [&](const Json& request) { license::require_valid_license(); return identify_server(request); }},
        {"read", [&](const Json& request) { return session.read(request); }},
        {"rcb_status", [&](const Json& request) { return reports.status(request); }},
        {"rcb_reserve", [&](const Json& request) { return reports.reserve(request); }},
        {"rcb_release", [&](const Json& request) { return reports.release(request); }},
        {"disconnect", [&](const Json&) { reports.release_all(); session.close(); return Json(true); }},
    };
    Json message;
    try {
        while (read_frame(message)) {
            auto id = message.value("id", "");
            try {
                if (message.value("version", 0) != 1) throw std::runtime_error("Protocol version must be 1");
                auto op = message.at("op").get<std::string>();
                if (op == "stop") {
                    goose.stop();
                    sv.stop();
                    reports.release_all();
                    session.close();
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
