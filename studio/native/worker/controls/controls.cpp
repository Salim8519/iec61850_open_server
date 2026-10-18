#include "controls.hpp"
#include <cstring>
#include <stdexcept>

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

std::string client_origin(ControlAction action) {
    int size = 0;
    auto value = ControlAction_getOrIdent(action, &size);
    return value && size > 0 ? std::string(reinterpret_cast<char*>(value), size) : "";
}

bool compatibility_applies(ControlBinding* binding, ControlAction action) {
    return binding->compatibility == "concen_inverted_sbow"
        && client_origin(action) == "CONCEN";
}

std::string peer_of(ControlAction action) {
    auto client = ControlAction_getClientConnection(action);
    auto peer = client ? ClientConnection_getPeerAddress(client) : nullptr;
    return peer ? peer : "";
}

void apply_value(ControlBinding* binding, bool raw_target, bool target,
        const char* trigger, const std::string& peer) {
    if (!peer.empty()) binding->server->access.write(peer, binding->reference);
    Json requested = {{"event", "control_requested"}, {"ref", binding->reference},
        {"value", raw_target}, {"trigger", trigger}, {"peer", peer}};
    if (raw_target != target) {
        requested["adapted_value"] = target;
        requested["compatibility"] = binding->compatibility;
    }
    emit_frame(requested);
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
    Json applied = {{"event", "control_applied"}, {"ref", binding->reference},
        {"cdc", binding->cdc}, {"action", binding->cdc == "DPC" ?
            (target ? "close" : "open") : (target ? "set_true" : "set_false")},
        {"ctl_value", target}, {"raw_ctl_value", raw_target}, {"status_ref", binding->status},
        {"value", result}, {"changed", changed}, {"timestamp", now}, {"trigger", trigger}, {"peer", peer}};
    if (raw_target != target) applied["compatibility"] = binding->compatibility;
    emit_frame(applied);
}

void selection(ControlAction, void* parameter, bool selected,
        SelectStateChangedReason reason) {
    auto binding = static_cast<ControlBinding*>(parameter);
    bool apply = false;
    bool raw_target = false;
    bool auto_applied = false;
    std::string peer;
    {
        std::lock_guard<std::mutex> guard(binding->compatibility_lock);
        if (selected && reason == SELECT_STATE_REASON_SELECTED
                && binding->pending_compatibility) {
            apply = true;
            raw_target = binding->pending_raw_target;
            binding->pending_applied = true;
            auto client_peer = binding->pending_client ? ClientConnection_getPeerAddress(binding->pending_client) : nullptr;
            peer = client_peer ? client_peer : "";
        }
        auto_applied = binding->pending_applied;
    }
    if (apply) {
        binding->server->defer([binding, raw_target, peer] {
            apply_value(binding, raw_target, !raw_target, "SBOw", peer);
            emit_frame({{"event", "control_compatibility_applied"},
                {"ref", binding->reference}, {"profile", binding->compatibility},
                {"client_origin", "CONCEN"}, {"trigger", "SBOw"},
                {"requested_value", raw_target}, {"executed_value", !raw_target},
                {"requested_action", raw_target ? "close" : "open"},
                {"executed_action", raw_target ? "open" : "close"}});
        });
    }
    Json event = {{"event", "control_selection"}, {"ref", binding->reference},
        {"state", selection_state(reason)}, {"selected", selected},
        {"model", binding->model}, {"sbo_timeout_ms", binding->sbo_timeout_ms}};
    if (auto_applied) {
        event["auto_applied"] = true;
        event["compatibility"] = binding->compatibility;
    }
    emit_frame(event);
    if (!selected) {
        std::lock_guard<std::mutex> guard(binding->compatibility_lock);
        binding->pending_client = nullptr;
        binding->pending_ctl_num = -1;
        binding->pending_compatibility = false;
        binding->pending_applied = false;
    }
}

CheckHandlerResult check(ControlAction action, void* parameter, MmsValue* value, bool test, bool) {
    auto binding = static_cast<ControlBinding*>(parameter);
    if (binding->blocked) return CONTROL_OBJECT_ACCESS_DENIED;
    if (MmsValue_getType(value) != MMS_BOOLEAN) return CONTROL_VALUE_INVALID;
    if (!test && ControlAction_isSelect(action) && compatibility_applies(binding, action)) {
        std::lock_guard<std::mutex> guard(binding->compatibility_lock);
        binding->pending_client = ControlAction_getClientConnection(action);
        binding->pending_ctl_num = ControlAction_getCtlNum(action);
        binding->pending_raw_target = MmsValue_getBoolean(value);
        binding->pending_compatibility = true;
        binding->pending_applied = false;
    }
    return CONTROL_ACCEPTED;
}

ControlHandlerResult operate(ControlAction action, void* parameter, MmsValue* value, bool test) {
    auto binding = static_cast<ControlBinding*>(parameter);
    if (binding->blocked || MmsValue_getType(value) != MMS_BOOLEAN) return CONTROL_RESULT_FAILED;
    if (test) {
        emit_frame({{"event", "control_test"}, {"ref", binding->reference}});
        return CONTROL_RESULT_OK;
    }
    bool raw_target = MmsValue_getBoolean(value);
    bool adapted = compatibility_applies(binding, action);
    bool duplicate = false;
    if (adapted) {
        std::lock_guard<std::mutex> guard(binding->compatibility_lock);
        duplicate = binding->pending_applied
            && binding->pending_client == ControlAction_getClientConnection(action)
            && binding->pending_ctl_num == ControlAction_getCtlNum(action)
            && binding->pending_raw_target == raw_target;
    }
    if (duplicate) {
        emit_frame({{"event", "control_compatibility_oper_confirmed"},
            {"ref", binding->reference}, {"profile", binding->compatibility},
            {"client_origin", "CONCEN"}, {"raw_ctl_value", raw_target}});
        return CONTROL_RESULT_OK;
    }
    apply_value(binding, raw_target, adapted ? !raw_target : raw_target, "Oper", peer_of(action));
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
    binding->compatibility = config.value("compatibility", "standard");
    if (binding->cdc != "SPC" && binding->cdc != "DPC")
        throw std::runtime_error("Control CDC must be SPC or DPC");
    if (binding->compatibility != "standard"
            && !(binding->compatibility == "concen_inverted_sbow"
                && binding->cdc == "DPC" && binding->model == 4))
        throw std::runtime_error("Invalid control compatibility profile");
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
