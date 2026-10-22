// Select, Operate and Cancel as separate typed steps (see typed_session.hpp).
#include "typed_session.hpp"
#include "control_session.hpp"
#include <stdexcept>

namespace {
using Clock = std::chrono::steady_clock;

void terminated(void* parameter, ControlObjectClient control) {
    auto error = ControlObjectClient_getLastApplError(control);
    auto& done = *static_cast<TypedSession::Termination*>(parameter);
    std::lock_guard<std::mutex> guard(done.lock);
    done.cause = int(error.addCause);
    done.state = error.error == CONTROL_ERROR_NO_ERROR ? 1 : -1;
    done.arrived.notify_all();
}

bool sbo(int model) { return model == CONTROL_MODEL_SBO_NORMAL || model == CONTROL_MODEL_SBO_ENHANCED; }

Json answer(const std::string& reference, int model) {
    return {{"ref", reference}, {"model", typed_model_name(model)}, {"selected", false}, {"operated", false},
            {"canceled", false}, {"terminated", nullptr}, {"add_cause", 0}, {"add_cause_text", ""},
            {"expires_ms", 0}, {"ok", false}};
}

Json& refused(Json& result, ControlObjectClient control, const char* step) {
    auto error = ControlObjectClient_getLastApplError(control);
    auto client_error = ControlObjectClient_getLastError(control);
    result["add_cause"] = int(error.addCause);
    result["add_cause_text"] = add_cause_text(int(error.addCause));
    result["client_error"] = int(client_error);
    result["error"] = std::string(step) + " refused by the device: " + IedClientError_toString(client_error)
                      + (error.addCause ? std::string(" (") + add_cause_text(int(error.addCause)) + ")" : "");
    return result;
}

void configure(ControlObjectClient control, const Json& request) {
    ControlObjectClient_setOrigin(control, request.value("or_ident", std::string("ScadaBench")).c_str(),
                                  request.value("or_cat", int(CONTROL_ORCAT_REMOTE_CONTROL)));
    ControlObjectClient_setInterlockCheck(control, request.value("interlock_check", false));
    ControlObjectClient_setSynchroCheck(control, request.value("synchro_check", false));
    ControlObjectClient_setTestMode(control, request.value("test", false));
}

std::string endpoint_of(const Json& request) {
    return request.at("ip").get<std::string>() + ":" + std::to_string(request.value("port", 102));
}

using Value = std::unique_ptr<MmsValue, decltype(&MmsValue_delete)>;
}

TypedSession::~TypedSession() { close_all(); }

IedConnection TypedSession::connection(const std::string& ip, int port) {
    auto key = ip + ":" + std::to_string(port);
    auto found = connections.find(key);
    if (found != connections.end()) {
        if (IedConnection_getState(found->second) == IED_STATE_CONNECTED) return found->second;
        drop(key);
    }
    auto created = IedConnection_create();
    IedConnection_setConnectTimeout(created, 5000);
    IedConnection_setRequestTimeout(created, 10000);
    IedClientError error = IED_ERROR_OK;
    IedConnection_connect(created, &error, ip.c_str(), port);
    if (error != IED_ERROR_OK) {
        IedConnection_destroy(created);
        throw std::runtime_error("Could not connect to " + key + " (MMS error " + std::to_string(int(error)) + ")");
    }
    connections[key] = created;
    return created;
}

// Selections of an endpoint go before its association: their objects belong to it.
void TypedSession::drop(const std::string& endpoint) {
    for (auto it = selections.begin(); it != selections.end();) {
        if (it->second->endpoint == endpoint) {
            ControlObjectClient_destroy(it->second->handle);
            it = selections.erase(it);
        } else ++it;
    }
    auto found = connections.find(endpoint);
    if (found == connections.end()) return;
    IedConnection_close(found->second);
    IedConnection_destroy(found->second);
    connections.erase(found);
}

void TypedSession::forget(const std::string& key) {
    auto found = selections.find(key);
    if (found == selections.end()) return;
    ControlObjectClient_setCommandTerminationHandler(found->second->handle, nullptr, nullptr);
    ControlObjectClient_destroy(found->second->handle);
    selections.erase(found);
}

void TypedSession::close_all() {
    while (!connections.empty()) drop(connections.begin()->first);
    for (auto& entry : selections) ControlObjectClient_destroy(entry.second->handle);
    selections.clear();
}

std::unique_ptr<TypedSession::Control> TypedSession::open(const Json& request, IedConnection handle,
                                                          const std::string& endpoint) {
    auto reference = request.at("ref").get<std::string>();
    auto control = std::make_unique<Control>();
    control->handle = ControlObjectClient_create(reference.c_str(), handle);
    if (!control->handle) {
        if (IedConnection_getState(handle) != IED_STATE_CONNECTED) drop(endpoint);
        throw std::runtime_error(reference + " is not a controllable object on this device");
    }
    control->endpoint = endpoint;
    control->model = ControlObjectClient_getControlModel(control->handle);
    IedClientError error = IED_ERROR_OK;
    auto timeout = IedConnection_readUnsigned32Value(handle, &error, (reference + ".sboTimeout").c_str(), IEC61850_FC_CF);
    control->sbo_timeout_ms = error == IED_ERROR_OK && timeout > 0 ? int(timeout) : 30000;
    ControlObjectClient_setCommandTerminationHandler(control->handle, terminated, control->done.get());
    configure(control->handle, request);
    return control;
}

Json TypedSession::select(const Json& request) {
    auto endpoint = endpoint_of(request);
    auto key = endpoint + "|" + request.at("ref").get<std::string>();
    forget(key);  // a new Select replaces the previous one of this object
    auto handle = connection(request.at("ip").get<std::string>(), request.value("port", 102));
    auto control = open(request, handle, endpoint);
    std::unique_ptr<sControlObjectClient, decltype(&ControlObjectClient_destroy)> owner(control->handle, ControlObjectClient_destroy);
    auto result = answer(request.at("ref").get<std::string>(), control->model);
    result["sbo_timeout_ms"] = control->sbo_timeout_ms;
    if (control->model == CONTROL_MODEL_STATUS_ONLY) {
        result["error"] = "The device marks this object as status-only; it cannot be operated";
        return result;
    }
    if (!sbo(control->model)) {  // direct models need no select: the next Operate goes straight through
        result["ok"] = true;
        return result;
    }
    bool selected = false;
    if (control->model == CONTROL_MODEL_SBO_NORMAL) {
        selected = ControlObjectClient_select(control->handle);  // SBO normal selects without a value
    } else {
        Value value(typed_control_value(control->handle, request.at("value"), request.at("kind").get<std::string>()),
                    MmsValue_delete);
        selected = ControlObjectClient_selectWithValue(control->handle, value.get());
    }
    if (!selected) return refused(result, control->handle, "Select");
    control->selected = true;
    control->expires = Clock::now() + std::chrono::milliseconds(control->sbo_timeout_ms);
    result["selected"] = true;
    result["expires_ms"] = control->sbo_timeout_ms;
    result["ok"] = true;
    owner.release();
    selections[key] = std::move(control);
    return result;
}

Json TypedSession::operate(const Json& request) {
    auto endpoint = endpoint_of(request);
    auto reference = request.at("ref").get<std::string>();
    auto key = endpoint + "|" + reference;
    auto handle = connection(request.at("ip").get<std::string>(), request.value("port", 102));
    std::unique_ptr<Control> control;
    auto found = selections.find(key);
    bool expired = false;
    if (found != selections.end()) {
        expired = Clock::now() >= found->second->expires;
        if (!expired) { control = std::move(found->second); selections.erase(found); }
        else forget(key);
    }
    bool reused = control != nullptr;
    if (!control) control = open(request, handle, endpoint);
    else configure(control->handle, request);
    auto result = answer(reference, control->model);
    std::unique_ptr<sControlObjectClient, decltype(&ControlObjectClient_destroy)> owner(control->handle, ControlObjectClient_destroy);
    auto finish = [&](Json& value) { ControlObjectClient_setCommandTerminationHandler(control->handle, nullptr, nullptr); return value; };
    if (control->model == CONTROL_MODEL_STATUS_ONLY) {
        result["error"] = "The device marks this object as status-only; it cannot be operated";
        return finish(result);
    }
    Value value(typed_control_value(control->handle, request.at("value"), request.at("kind").get<std::string>()),
                MmsValue_delete);
    if (sbo(control->model) && !reused) {
        if (request.value("require_select", false)) {
            result["error_code"] = expired ? "select_expired" : "not_selected";
            result["error"] = expired ? "Select expired after " + std::to_string(control->sbo_timeout_ms / 1000) +
                                            " s. Press Select again." : "Press Select first.";
            return finish(result);
        }
        bool selected = control->model == CONTROL_MODEL_SBO_NORMAL ? ControlObjectClient_select(control->handle)
                                                                   : ControlObjectClient_selectWithValue(control->handle, value.get());
        if (!selected) return finish(refused(result, control->handle, "Select"));
    }
    result["selected"] = sbo(control->model);
    if (!ControlObjectClient_operate(control->handle, value.get(), 0)) return finish(refused(result, control->handle, "Operate"));
    result["operated"] = true;
    if (control->model == CONTROL_MODEL_DIRECT_ENHANCED || control->model == CONTROL_MODEL_SBO_ENHANCED) {
        auto& done = *control->done;
        std::unique_lock<std::mutex> guard(done.lock);
        done.arrived.wait_for(guard, std::chrono::milliseconds(request.value("termination_ms", 5000)),
                              [&] { return done.state != 0; });
        result["terminated"] = done.state == 1;
        if (done.state != 1) {
            result["add_cause"] = done.cause;
            result["add_cause_text"] = add_cause_text(done.cause);
            result["error"] = done.state == -1 ? "The device ended the command with a negative termination"
                                               : "No command termination arrived in time; check the value on the device";
            guard.unlock();
            return finish(result);
        }
    }
    result["ok"] = true;
    return finish(result);
}

Json TypedSession::cancel(const Json& request) {
    auto endpoint = endpoint_of(request);
    auto reference = request.at("ref").get<std::string>();
    auto key = endpoint + "|" + reference;
    auto found = selections.find(key);
    if (found == selections.end()) {
        auto result = answer(reference, 0);
        result["model"] = "unknown";
        result["ok"] = true;  // nothing selected here: nothing to cancel
        return result;
    }
    auto& control = *found->second;
    auto result = answer(reference, control.model);
    if (ControlObjectClient_cancel(control.handle)) {
        result["canceled"] = true;
        result["ok"] = true;
    } else {
        refused(result, control.handle, "Cancel");
    }
    forget(key);
    return result;
}
