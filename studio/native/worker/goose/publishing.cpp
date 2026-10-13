// GOOSE publishing for simulated IEDs. libiec61850 owns the GoCB state and frames;
// this module only selects the adapter and reports what the stack did.
#include "publishing.hpp"
#include "npcap.hpp"
#include <thread>

namespace {
void gocb_event(MmsGooseControlBlock control, int event, void*) {
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
    auto status = npcap_status();
    summary["npcap"] = status;
    if (!status["available"].get<bool>()) {
        summary["reason"] = "Npcap is not installed, so GOOSE cannot be sent";
        return summary;
    }
    try {
        server.goose_interface = npcap_interface_for(settings.at("adapter_guid").get<std::string>());
    } catch (const std::exception& error) {
        summary["reason"] = error.what();
        return summary;
    }
    IedServer_setGooseInterfaceId(server.handle, server.goose_interface.c_str());
    IedServer_setGoCBHandler(server.handle, gocb_event, nullptr);
    summary["active"] = true;
    return summary;
}

void start_goose(Server& server) {
    if (server.goose_interface.empty()) return;
    auto handle = server.handle;
    server.goose_thread = std::thread([handle] {
        emit_frame({{"event", "goose_publishing"}, {"state", "starting"}});
        IedServer_enableGoosePublishing(handle);
        emit_frame({{"event", "goose_publishing"}, {"state", "active"}});
    });
}
