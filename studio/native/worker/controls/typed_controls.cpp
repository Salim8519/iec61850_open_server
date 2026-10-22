// Typed controls of a simulated IED (ADR-0104): the vendor stack runs the control model
// (direct/SBO, normal/enhanced, select timeout, cancel); these handlers decide the value, the
// simulated interlock/synchrocheck blocks and the feedback, and log who sent the command.
#include "typed_binding.hpp"
#include <algorithm>
#include <cctype>
#include <cstdio>
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

std::string peer_of(ControlAction action) {
    auto client = ControlAction_getClientConnection(action);
    auto peer = client ? ClientConnection_getPeerAddress(client) : nullptr;
    return peer ? peer : "";
}

// orIdent is an octet string: printable text as is, anything else as hex.
std::string originator(ControlAction action) {
    int size = 0;
    auto bytes = ControlAction_getOrIdent(action, &size);
    std::string text;
    bool printable = true;
    for (int i = 0; bytes && i < size; ++i) printable = printable && std::isprint(bytes[i]);
    for (int i = 0; bytes && i < size; ++i) {
        if (printable) { text.push_back(char(bytes[i])); continue; }
        char hex[3];
        std::snprintf(hex, sizeof(hex), "%02x", bytes[i]);
        text += hex;
    }
    return text;
}

Json origin_fields(ControlAction action, Json frame) {
    frame["or_cat"] = ControlAction_getOrCat(action);
    frame["or_ident"] = originator(action);
    frame["ctl_num"] = ControlAction_getCtlNum(action);
    return frame;
}

void selection(ControlAction, void* parameter, bool selected, SelectStateChangedReason reason) {
    auto binding = static_cast<TypedBinding*>(parameter);
    emit_frame({{"event", "control_selection"}, {"ref", binding->reference}, {"cdc", binding->cdc},
        {"state", selection_state(reason)}, {"selected", selected},
        {"model", binding->model}, {"sbo_timeout_ms", binding->sbo_timeout_ms}});
}

CheckHandlerResult check(ControlAction action, void* parameter, MmsValue* value, bool, bool interlock) {
    auto binding = static_cast<TypedBinding*>(parameter);
    if (binding->blocked) {
        ControlAction_setAddCause(action, ADD_CAUSE_BLOCKED_BY_PROCESS);
        return CONTROL_OBJECT_ACCESS_DENIED;
    }
    if (interlock && binding->interlock_blocked) {
        ControlAction_setAddCause(action, ADD_CAUSE_BLOCKED_BY_INTERLOCKING);
        return CONTROL_OBJECT_ACCESS_DENIED;
    }
    if (!value) return CONTROL_ACCEPTED;  // select without a value (SBO normal security)
    auto plan = plan_typed(*binding, value);
    if (plan.valid) return CONTROL_ACCEPTED;
    ControlAction_setAddCause(action, ControlAddCause(plan.add_cause));
    return CONTROL_VALUE_INVALID;
}

ControlHandlerResult wait_for_execution(ControlAction action, void* parameter, MmsValue*, bool, bool synchro) {
    auto binding = static_cast<TypedBinding*>(parameter);
    if (synchro && binding->synchro_blocked) {
        ControlAction_setAddCause(action, ADD_CAUSE_BLOCKED_BY_SYNCHROCHECK);
        return CONTROL_RESULT_FAILED;
    }
    return CONTROL_RESULT_OK;
}

ControlHandlerResult operate(ControlAction action, void* parameter, MmsValue* value, bool test) {
    auto binding = static_cast<TypedBinding*>(parameter);
    auto peer = peer_of(action);
    auto plan = plan_typed(*binding, value);
    if (binding->blocked || !plan.valid) {
        ControlAction_setAddCause(action, ControlAddCause(binding->blocked ? ADD_CAUSE_BLOCKED_BY_PROCESS : plan.add_cause));
        return CONTROL_RESULT_FAILED;
    }
    if (test) {
        emit_frame(origin_fields(action, {{"event", "control_test"}, {"ref", binding->reference},
            {"cdc", binding->cdc}, {"value", plan.ctl_value}, {"peer", peer}}));
        return CONTROL_RESULT_OK;
    }
    if (!peer.empty()) binding->server->access.write(peer, binding->reference);
    emit_frame(origin_fields(action, {{"event", "control_requested"}, {"ref", binding->reference},
        {"value", plan.ctl_value}, {"trigger", "Oper"}, {"peer", peer}, {"cdc", binding->cdc}}));
    uint64_t now = Hal_getTimeInMs();
    bool changed = false;
    IedServer_lockDataModel(binding->server->handle);
    auto result = apply_typed(*binding, plan, changed, now);
    IedServer_unlockDataModel(binding->server->handle);
    emit_frame(origin_fields(action, {{"event", "control_applied"}, {"ref", binding->reference},
        {"cdc", binding->cdc}, {"action", plan.action}, {"ctl_value", plan.ctl_value},
        {"status_ref", binding->status}, {"value", result}, {"changed", changed}, {"timestamp", now},
        {"trigger", "Oper"}, {"peer", peer}, {"typed", true}}));
    return CONTROL_RESULT_OK;
}

DataAttribute* optional_attribute(Server& server, const Json& config, const char* key) {
    if (!config.contains(key) || !config.at(key).is_string()) return nullptr;
    return server.attribute(config.at(key).get<std::string>());
}
}

void install_typed_control(Server& server, const Json& config) {
    auto binding = std::make_unique<TypedBinding>();
    binding->server = &server;
    binding->reference = config.at("ref").get<std::string>();
    binding->status = config.at("status").get<std::string>();
    binding->cdc = config.at("cdc").get<std::string>();
    binding->value_kind = config.at("value_kind").get<std::string>();
    binding->model = config.value("model", 0);
    binding->sbo_timeout_ms = config.value("sbo_timeout_ms", 0);
    static const std::vector<std::string> kinds = {"int", "enum", "float", "step"};
    if (std::find(kinds.begin(), kinds.end(), binding->value_kind) == kinds.end())
        throw std::runtime_error("Typed control value kind must be int, enum, float or step");
    binding->feedback = server.attribute(binding->status);
    auto feedback_type = MmsValue_getType(IedServer_getAttributeValue(server.handle, binding->feedback));
    bool float_feedback = binding->cdc == "APC" || binding->cdc == "BAC";
    if (float_feedback ? feedback_type != MMS_FLOAT : feedback_type != MMS_INTEGER)
        throw std::runtime_error("Typed control " + binding->reference + " does not match its feedback type");
    binding->trans_ind = optional_attribute(server, config, "trans_ind");
    auto node = IedModel_getModelNodeByObjectReference(server.model, binding->reference.c_str());
    if (!node || node->modelType != DataObjectModelType) throw std::runtime_error("Invalid control object");
    auto parent = binding->status.substr(0, binding->status.rfind('.'));
    if (binding->cdc == "APC" || binding->cdc == "BAC" || binding->cdc == "BSC" || binding->cdc == "ISC")
        parent = parent.substr(0, parent.rfind('.'));  // mxVal.f / valWTr.posVal: t is beside the structure
    auto time_node = IedModel_getModelNodeByObjectReference(server.model, (parent + ".t").c_str());
    if (time_node && time_node->modelType == DataAttributeModelType)
        binding->timestamp = reinterpret_cast<DataAttribute*>(time_node);
    if (config.contains("min") && config.contains("max") && config.at("min").is_number() && config.at("max").is_number()) {
        binding->has_limits = true;
        binding->minimum = config.at("min").get<double>();
        binding->maximum = config.at("max").get<double>();
    }
    if (config.contains("step") && config.at("step").is_number()) binding->step = config.at("step").get<double>();
    auto labels = config.value("labels", Json::object());  // a named copy: items() must not outlive it
    for (auto& entry : labels.items()) binding->allowed.push_back(entry.value().get<long long>());
    binding->blocked = config.value("blocked", false);
    binding->interlock_blocked = config.value("interlock_blocked", false);
    binding->synchro_blocked = config.value("synchro_blocked", false);
    auto object = reinterpret_cast<DataObject*>(node);
    IedServer_setPerformCheckHandler(server.handle, object, check, binding.get());
    IedServer_setWaitForExecutionHandler(server.handle, object, wait_for_execution, binding.get());
    IedServer_setControlHandler(server.handle, object, operate, binding.get());
    IedServer_setSelectStateChangedHandler(server.handle, object, selection, binding.get());
    server.typed_controls.push_back(std::move(binding));
}

// Simulated blocks of one typed control: {"ref", "blocked"?, "interlock_blocked"?, "synchro_blocked"?}.
void set_typed_checks(Server& server, const Json& request) {
    auto reference = request.at("ref").get<std::string>();
    for (auto& binding : server.typed_controls) {
        if (binding->reference != reference) continue;
        if (request.contains("blocked")) binding->blocked = request.at("blocked").get<bool>();
        if (request.contains("interlock_blocked")) binding->interlock_blocked = request.at("interlock_blocked").get<bool>();
        if (request.contains("synchro_blocked")) binding->synchro_blocked = request.at("synchro_blocked").get<bool>();
        return;
    }
    throw std::runtime_error("Unknown typed control " + reference);
}
