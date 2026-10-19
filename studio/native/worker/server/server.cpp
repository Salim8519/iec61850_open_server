#include "server.hpp"
#include "controls/controls.hpp"
#include "goose/publishing.hpp"
#include "read_hook.h"
extern "C" {
#include "iec61850_config_file_parser.h"
}
#include <stdexcept>

namespace {
// Peer "ip:port" of an MMS client; empty for events the server raises itself (ADR-0066).
std::string peer_of(ClientConnection client) {
    auto peer = client ? ClientConnection_getPeerAddress(client) : nullptr;
    return peer ? peer : "";
}
void connection(IedServer, ClientConnection client, bool connected, void* parameter) {
    auto peer = peer_of(client);
    static_cast<Server*>(parameter)->access.connection(peer, connected);
    emit_frame({{"event", "connection"}, {"connected", connected}, {"peer", peer}});
}
void report(void* parameter, ReportControlBlock* rcb, ClientConnection client,
        IedServer_RCBEventType event, const char* name, MmsDataAccessError error) {
    auto peer = peer_of(client);
    auto parent = ReportControlBlock_getParent(rcb);
    std::string block = std::string(parent ? parent->name : "") + "." + ReportControlBlock_getName(rcb);
    if (!peer.empty()) static_cast<Server*>(parameter)->access.report(peer, block, static_cast<int>(event));
    emit_frame({{"event", "report_control"}, {"name", ReportControlBlock_getName(rcb)},
        {"kind", static_cast<int>(event)}, {"parameter", name ? name : ""},
        {"error", static_cast<int>(error)}, {"peer", peer}});
}
void observed_read(void* context, const char* domain, const char* item, const char* peer) {
    static_cast<AccessLog*>(context)->read(domain, item, peer);
}
}

Server::Server() = default;
Server::~Server() { stop(); }

Json Server::start(const Json& config) {
    if (handle) throw std::runtime_error("IED already started");
    auto path = config.at("cfg").get<std::string>();
    model = ConfigFileParser_createModelFromConfigFileEx(path.c_str());
    if (!model) throw std::runtime_error("Native model parser rejected configuration");
    auto options = IedServerConfig_create();
    IedServerConfig_setReportBufferSize(options, 1024 * 1024);
    IedServerConfig_enableFileService(options, false);
    IedServerConfig_enableLogService(options, false);
    IedServerConfig_setMaxMmsConnections(options, 8);
    IedServerConfig_setEdition(options, config.value("edition", 2) == 1 ?
        IEC_61850_EDITION_1 : IEC_61850_EDITION_2);
    handle = IedServer_createWithConfig(model, nullptr, options);
    IedServerConfig_destroy(options);
    if (!handle) throw std::runtime_error("Unable to construct MMS server");
    auto ip = config.at("ip").get<std::string>();
    IedServer_setLocalIpAddress(handle, ip.c_str());
    // What clients and Find IEDs see: the product name and that this IED is simulated (change 0089).
    IedServer_setServerIdentity(handle, "ScadaBench", "Simulated IED", "1");
    IedServer_setConnectionIndicationHandler(handle, connection, this);
    IedServer_setRCBEventHandler(handle, report, this);
    bool observing = studio_observe_reads(handle, observed_read, &access);
    for (const auto& entry : config.value("controls", Json::array())) {
        install_control(*this, entry);
    }
    auto goose = prepare_goose(*this, config);
    int port = config.value("port", 102);
    IedServer_start(handle, port);
    if (!IedServer_isRunning(handle)) {
        stop();
        throw std::runtime_error("MMS bind failed: " + ip + ":" + std::to_string(port));
    }
    start_goose(*this);
    access.start();
    auto capabilities = Json::array({"versioned_updates", "atomic_control_feedback",
                                     "control_selection_events", "standard_controls", "goose_aware", "goose_bus"});
    if (observing) capabilities.push_back("mms_access");
    return {{"running", true}, {"ip", ip}, {"port", port}, {"goose", goose}, {"capabilities", capabilities}};
}

DataAttribute* Server::attribute(const std::string& reference) {
    if (!handle) throw std::runtime_error("IED is not running");
    auto node = IedModel_getModelNodeByObjectReference(model, reference.c_str());
    if (!node || node->modelType != DataAttributeModelType)
        throw std::runtime_error("Unknown data attribute: " + reference);
    return reinterpret_cast<DataAttribute*>(node);
}

void Server::stop() {
    access.stop();
    if (goose_thread.joinable()) goose_thread.join();
    goose_interface.clear();
    if (handle) { IedServer_stop(handle); IedServer_destroy(handle); handle = nullptr; }
    controls.clear();
    revisions.clear();
    if (model) { IedModel_destroy(model); model = nullptr; }
}
