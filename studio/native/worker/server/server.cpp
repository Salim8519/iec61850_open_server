#include "server.hpp"
#include "controls/controls.hpp"
#include "goose/publishing.hpp"
extern "C" {
#include "iec61850_config_file_parser.h"
}
#include <stdexcept>

namespace {
void connection(IedServer, ClientConnection, bool connected, void*) {
    emit_frame({{"event", "connection"}, {"connected", connected}});
}
void report(void*, ReportControlBlock* rcb, ClientConnection,
        IedServer_RCBEventType event, const char* parameter, MmsDataAccessError error) {
    emit_frame({{"event", "report_control"}, {"name", ReportControlBlock_getName(rcb)},
        {"kind", static_cast<int>(event)}, {"parameter", parameter ? parameter : ""},
        {"error", static_cast<int>(error)}});
}
}

Server::Server() = default;
Server::~Server() { stop(); }

void Server::start_tasks() {
    tasks_stopping = false;
    task_thread = std::thread([this] {
        while (true) {
            std::function<void()> task;
            {
                std::unique_lock<std::mutex> guard(task_lock);
                task_ready.wait(guard, [this] { return tasks_stopping || !tasks.empty(); });
                if (tasks.empty() && tasks_stopping) return;
                task = std::move(tasks.front());
                tasks.pop_front();
            }
            try { task(); }
            catch (const std::exception& error) {
                emit_frame({{"event", "control_compatibility_failed"}, {"message", error.what()}});
            }
        }
    });
}

void Server::defer(std::function<void()> task) {
    {
        std::lock_guard<std::mutex> guard(task_lock);
        if (tasks_stopping) return;
        tasks.push_back(std::move(task));
    }
    task_ready.notify_one();
}

void Server::stop_tasks() {
    {
        std::lock_guard<std::mutex> guard(task_lock);
        tasks_stopping = true;
    }
    task_ready.notify_all();
    if (task_thread.joinable()) task_thread.join();
    tasks.clear();
}

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
    start_tasks();
    auto ip = config.at("ip").get<std::string>();
    IedServer_setLocalIpAddress(handle, ip.c_str());
    IedServer_setServerIdentity(handle, "Substation Studio", "Laboratory simulator", "0.1");
    IedServer_setConnectionIndicationHandler(handle, connection, this);
    IedServer_setRCBEventHandler(handle, report, this);
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
    return {{"running", true}, {"ip", ip}, {"port", port}, {"goose", goose},
        {"capabilities", Json::array({"versioned_updates", "atomic_control_feedback",
                                      "control_selection_events", "concen_sbow_compatibility",
                                      "goose_aware"})}};
}

DataAttribute* Server::attribute(const std::string& reference) {
    if (!handle) throw std::runtime_error("IED is not running");
    auto node = IedModel_getModelNodeByObjectReference(model, reference.c_str());
    if (!node || node->modelType != DataAttributeModelType)
        throw std::runtime_error("Unknown data attribute: " + reference);
    return reinterpret_cast<DataAttribute*>(node);
}

void Server::stop() {
    stop_tasks();
    if (goose_thread.joinable()) goose_thread.join();
    goose_interface.clear();
    if (handle) { IedServer_stop(handle); IedServer_destroy(handle); handle = nullptr; }
    controls.clear();
    revisions.clear();
    if (model) { IedModel_destroy(model); model = nullptr; }
}
