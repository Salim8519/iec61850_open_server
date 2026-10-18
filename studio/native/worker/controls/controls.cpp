#include "controls.hpp"
#include <cstring>
#include <stdexcept>

// Standard IEC 61850 control only (ADR-0069): ctlVal true closes / sets, false opens /
// resets, and the switch moves on Oper. Select only reserves the object.
namespace {
const char* selection_state(SelectStateChangedReason reason) {
    switch (reason) {
    case SELECT_STATE_REASON_SELECTED: return "selected";
    case SELECT_STATE_REASON_CANCELED: return "canceled";
    case SELECT_STATE_REASON_TIMEOUT: return "timed_out";
    case SELECT_STATE_REASON_OPERATED: return "operated";
    case SELECT_STATE_REASON_OPERATE_FAILED: return "failed";
    case SELECT_STATE_REASON_DISCONNECTED: return "disconnected";
    }
    return "unknown";
}

std::string peer_of(ControlAction action) {
    auto client = ControlAction_getClientConnection(action);
    auto peer = client ? ClientConnection_getPeerAddress(client) : nullptr;
    return peer ? peer : "";
}

void apply_value(ControlBinding* binding, bool target, const std::string& peer) {
    if (!peer.empty()) binding->server->access.write(peer, binding->reference);
    emit_frame({{"event", "control_requested"}, {"ref", binding->reference},
        {"value", target}, {"trigger", "Oper"}, {"peer", peer}});
    auto server = binding->server->handle;
    uint64_t now = Hal_getTimeInMs();
    Json result;
    bool changed = true;
    IedServer_lockDataModel(server);
    if (binding->cdc == "DPC") {
        int position = target ? 2 : 1;
        auto current = IedServer_getAttributeValue(server, binding->feedback);
        changed = current && Dbpos_fromMmsValue(current) != static_cast<Dbpos>(position);
        if (changed) {
            IedServer_updateDbposValue(server, binding->feedback, static_cast<Dbpos>(position));
            if (binding->timestamp) IedServer_updateUTCTimeAttributeValue(server, binding->timestamp, now);
            ++binding->server->revisions[binding->feedback];
        }
        result = position;
    } else {
        auto current = IedServer_getAttributeValue(server, binding->feedback);
        bool previous = current ? MmsValue_getBoolean(current) : !target;
        changed = previous != target;
        if (changed) {
            IedServer_updateBooleanAttributeValue(server, binding->feedback, target);
            if (binding->timestamp) IedServer_updateUTCTimeAttributeValue(server, binding->timestamp, now);
            ++binding->server->revisions[binding->feedback];
        }
        result = target;
    }
    IedServer_unlockDataModel(server);
    emit_frame({{"event", "control_applied"}, {"ref", binding->reference},
        {"cdc", binding->cdc}, {"action", binding->cdc == "DPC" ?
            (target ? "close" : "open") : (target ? "set_true" : "set_false")},
        {"ctl_value", target}, {"status_ref", binding->status},
        {"value", result}, {"changed", changed}, {"timestamp", now}, {"trigger", "Oper"}, {"peer", peer}});
}

void selection(ControlAction, void* parameter, bool selected,
        SelectStateChangedReason reason) {
    auto binding = static_cast<ControlBinding*>(parameter);
    emit_frame({{"event", "control_selection"}, {"ref", binding->reference},
        {"state", selection_state(reason)}, {"selected", selected},
        {"model", binding->model}, {"sbo_timeout_ms", binding->sbo_timeout_ms}});
}

CheckHandlerResult check(ControlAction, void* parameter, MmsValue* value, bool, bool) {
    auto binding = static_cast<ControlBinding*>(parameter);
    if (binding->blocked) return CONTROL_OBJECT_ACCESS_DENIED;
    if (MmsValue_getType(value) != MMS_BOOLEAN) return CONTROL_VALUE_INVALID;
    return CONTROL_ACCEPTED;
}

ControlHandlerResult operate(ControlAction action, void* parameter, MmsValue* value, bool test) {
    auto binding = static_cast<ControlBinding*>(parameter);
    if (binding->blocked || MmsValue_getType(value) != MMS_BOOLEAN) return CONTROL_RESULT_FAILED;
    if (test) {
        emit_frame({{"event", "control_test"}, {"ref", binding->reference}});
        return CONTROL_RESULT_OK;
    }
    apply_value(binding, MmsValue_getBoolean(value), peer_of(action));
    return CONTROL_RESULT_OK;
}
}

void install_control(Server& server, const Json& config) {
    auto binding = std::make_unique<ControlBinding>();
    binding->server = &server;
    binding->reference = config.at("ref").get<std::string>();
    binding->status = config.at("status").get<std::string>();
    binding->cdc = config.at("cdc").get<std::string>();
    binding->model = config.value("model", 0);
    binding->sbo_timeout_ms = config.value("sbo_timeout_ms", 0);
    if (binding->cdc != "SPC" && binding->cdc != "DPC")
        throw std::runtime_error("Control CDC must be SPC or DPC");
    if (config.contains("compatibility"))
        throw std::runtime_error("Control compatibility profiles were removed; send standard controls only");
    binding->feedback = server.attribute(binding->status);
    auto feedback_type = MmsValue_getType(IedServer_getAttributeValue(server.handle, binding->feedback));
    if ((binding->cdc == "SPC" && feedback_type != MMS_BOOLEAN)
            || (binding->cdc == "DPC" && feedback_type != MMS_BIT_STRING))
        throw std::runtime_error("Control CDC does not match its feedback type");
    auto node = IedModel_getModelNodeByObjectReference(server.model, binding->reference.c_str());
    if (!node || node->modelType != DataObjectModelType) throw std::runtime_error("Invalid control object");
    auto time_ref = binding->status.substr(0, binding->status.rfind('.')) + ".t";
    auto time_node = IedModel_getModelNodeByObjectReference(server.model, time_ref.c_str());
    if (time_node && time_node->modelType == DataAttributeModelType)
        binding->timestamp = reinterpret_cast<DataAttribute*>(time_node);
    binding->blocked = config.value("blocked", false);
    IedServer_setPerformCheckHandler(server.handle, reinterpret_cast<DataObject*>(node), check, binding.get());
    IedServer_setControlHandler(server.handle, reinterpret_cast<DataObject*>(node), operate, binding.get());
    IedServer_setSelectStateChangedHandler(
        server.handle, reinterpret_cast<DataObject*>(node), selection, binding.get());
    server.controls.push_back(std::move(binding));
}
