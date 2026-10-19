// GOOSE publishing for simulated IEDs. libiec61850 owns the GoCB state and frames;
// this module only points the stack at the Studio GOOSE engine and reports what it did.
// A worker never receives a network adapter (GOOSE-05, ADR-0073): the engine alone
// decides whether a frame may leave this PC.
#include "publishing.hpp"
#include <thread>

namespace {
void gocb_event(MmsGooseControlBlock control, int event, void* parameter) {
    // A client changed GoEna; the next frames go out on the following periodic runs.
    static_cast<Server*>(parameter)->loop.poll_for(20);
    auto ln = MmsGooseControlBlock_getLogicalNode(control);
    std::string name = MmsGooseControlBlock_getName(control);
    std::string node = ln ? ln->name : "";
    emit_frame({{"event", "goose_control"}, {"gocb", node + "$GO$" + name},
                {"enabled", event == IEC61850_GOCB_EVENT_ENABLE}});
}

int count_gocbs(IedModel* model) {
    int count = 0;
    for (auto block = model->gseCBs; block; block = block->sibling) ++count;
    return count;
}
}

Json prepare_goose(Server& server, const Json& config) {
    Json summary = {{"compiled", true}, {"requested", false}, {"active", false},
                    {"control_blocks", count_gocbs(server.model)}};
    auto settings = config.value("goose", Json::object());
    if (!settings.value("publish", false) || summary["control_blocks"] == 0) return summary;
    summary["requested"] = true;
    auto interface_id = settings.value("interface", "");
    if (interface_id.rfind("udp:", 0) != 0) {
        summary["reason"] = "GOOSE is sent only through the ScadaBench GOOSE engine";
        return summary;
    }
    server.goose_interface = interface_id;
    IedServer_setGooseInterfaceId(server.handle, server.goose_interface.c_str());
    IedServer_setGoCBHandler(server.handle, gocb_event, &server);
    summary["active"] = true;
    return summary;
}

void start_goose(Server& server) {
    if (server.goose_interface.empty()) return;
    auto handle = server.handle;
    auto loop = &server.loop;
    server.goose_thread = std::thread([handle, loop] {
        emit_frame({{"event", "goose_publishing"}, {"state", "starting"}});
        IedServer_enableGoosePublishing(handle);
        loop->poll_for(20);  // the first frames go out on the next periodic runs
        emit_frame({{"event", "goose_publishing"}, {"state", "active"}});
    });
}
