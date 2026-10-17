// One control action: select (when the model needs it), operate, then wait for the
// command termination of enhanced-security controls.
#include "control_session.hpp"
#include <atomic>
#include <chrono>
#include <memory>
#include <stdexcept>
#include <thread>

namespace {
struct Termination {
    std::atomic<int> state{0};  // 0 waiting, 1 success, -1 failed
    std::atomic<int> cause{0};
};

void terminated(void* parameter, ControlObjectClient control) {
    auto& done = *static_cast<Termination*>(parameter);
    auto error = ControlObjectClient_getLastApplError(control);
    done.cause = int(error.addCause);
    done.state = error.error == CONTROL_ERROR_NO_ERROR ? 1 : -1;
}

MmsValue* control_value(ControlObjectClient control, const Json& value) {
    auto type = ControlObjectClient_getCtlValType(control);
    if (type == MMS_BOOLEAN)
        return MmsValue_newBoolean(value.is_boolean() ? value.get<bool>() : value.get<int>() != 0);
    if (type == MMS_INTEGER) return MmsValue_newIntegerFromInt32(value.is_boolean() ? int(value.get<bool>()) : value.get<int>());
    throw std::runtime_error("This control expects a value type ScadaBench cannot send yet (only on/off and integer steps)");
}
}

ControlSession::~ControlSession() { close_all(); }

IedConnection ControlSession::connection(const std::string& ip, int port) {
    auto key = ip + ":" + std::to_string(port);
    auto found = connections.find(key);
    if (found != connections.end()) {
        if (IedConnection_getState(found->second) == IED_STATE_CONNECTED) return found->second;
        drop(ip, port);
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

void ControlSession::drop(const std::string& ip, int port) {
    auto key = ip + ":" + std::to_string(port);
    auto found = connections.find(key);
    if (found == connections.end()) return;
    IedConnection_close(found->second);
    IedConnection_destroy(found->second);
    connections.erase(found);
}

void ControlSession::close_all() {
    for (auto& entry : connections) {
        IedConnection_close(entry.second);
        IedConnection_destroy(entry.second);
    }
    connections.clear();
}

Json ControlSession::operate(const Json& request) {
    auto ip = request.at("ip").get<std::string>();
    int port = request.value("port", 102);
    auto reference = request.at("ref").get<std::string>();
    auto handle = connection(ip, port);
    std::unique_ptr<sControlObjectClient, decltype(&ControlObjectClient_destroy)> control(
        ControlObjectClient_create(reference.c_str(), handle), ControlObjectClient_destroy);
    if (!control) {
        if (IedConnection_getState(handle) != IED_STATE_CONNECTED) drop(ip, port);
        throw std::runtime_error(reference + " is not a controllable object on this device");
    }
    int model = ControlObjectClient_getControlModel(control.get());
    Json result = {{"ref", reference}, {"model", control_model_name(model)}, {"selected", false}, {"operated", false},
                   {"terminated", nullptr}, {"add_cause", 0}, {"add_cause_text", ""}, {"ok", false}};
    if (model == CONTROL_MODEL_STATUS_ONLY) {
        result["error"] = "The device marks this object as status-only; it cannot be operated";
        return result;
    }
    std::unique_ptr<MmsValue, decltype(&MmsValue_delete)> value(control_value(control.get(), request.at("value")),
                                                                MmsValue_delete);
    ControlObjectClient_setOrigin(control.get(), request.value("or_ident", std::string("ScadaBench")).c_str(),
                                  request.value("or_cat", CONTROL_ORCAT_REMOTE_CONTROL));
    ControlObjectClient_setInterlockCheck(control.get(), request.value("interlock_check", true));
    ControlObjectClient_setSynchroCheck(control.get(), request.value("synchro_check", false));
    ControlObjectClient_setTestMode(control.get(), request.value("test", false));
    Termination done;
    ControlObjectClient_setCommandTerminationHandler(control.get(), terminated, &done);
    auto failed = [&](const char* step) {
        auto error = ControlObjectClient_getLastApplError(control.get());
        result["add_cause"] = int(error.addCause);
        result["add_cause_text"] = add_cause_text(int(error.addCause));
        auto client_error = ControlObjectClient_getLastError(control.get());
        result["error"] = std::string(step) + " refused by the device: " + IedClientError_toString(client_error)
                          + (error.addCause ? std::string(" (") + add_cause_text(int(error.addCause)) + ")" : "");
        result["client_error"] = int(client_error);
        return result;
    };
    if (model == CONTROL_MODEL_SBO_NORMAL && !ControlObjectClient_select(control.get())) return failed("Select");
    if (model == CONTROL_MODEL_SBO_ENHANCED && !ControlObjectClient_selectWithValue(control.get(), value.get()))
        return failed("Select");
    result["selected"] = model == CONTROL_MODEL_SBO_NORMAL || model == CONTROL_MODEL_SBO_ENHANCED;
    if (!ControlObjectClient_operate(control.get(), value.get(), 0)) return failed("Operate");
    result["operated"] = true;
    bool enhanced = model == CONTROL_MODEL_DIRECT_ENHANCED || model == CONTROL_MODEL_SBO_ENHANCED;
    if (enhanced) {
        auto deadline = std::chrono::steady_clock::now() + std::chrono::milliseconds(request.value("termination_ms", 5000));
        while (done.state == 0 && std::chrono::steady_clock::now() < deadline)
            std::this_thread::sleep_for(std::chrono::milliseconds(10));
        result["terminated"] = done.state == 1;
        if (done.state == -1) {
            result["add_cause"] = int(done.cause);
            result["add_cause_text"] = add_cause_text(done.cause);
            result["error"] = "The device ended the command with a negative termination";
            return result;
        }
        if (done.state == 0) {
            result["error"] = "No command termination arrived in time; check the position on the device";
            return result;
        }
    }
    result["ok"] = true;
    return result;
}
