#include "sbo_controls.hpp"
#include <atomic>
#include <chrono>
#include <thread>

namespace {
using Json = nlohmann::json;

void terminated(void* parameter, ControlObjectClient control) {
    auto error = ControlObjectClient_getLastApplError(control);
    bool success = error.error == CONTROL_ERROR_NO_ERROR && error.addCause == ADD_CAUSE_UNKNOWN;
    static_cast<std::atomic<int>*>(parameter)->store(success ? 1 : -1);
}

std::string read_state(IedConnection connection, const std::string& reference) {
    IedClientError error = IED_ERROR_OK;
    auto value = IedConnection_readObject(connection, &error,
                                           (reference + ".stVal").c_str(), IEC61850_FC_ST);
    if (!value || error != IED_ERROR_OK) {
        if (value) MmsValue_delete(value);
        return "read-failed";
    }
    char buffer[128];
    MmsValue_printToBuffer(value, buffer, sizeof(buffer));
    MmsValue_delete(value);
    return buffer;
}

std::string read_time(IedConnection connection, const std::string& reference) {
    IedClientError error = IED_ERROR_OK;
    auto value = IedConnection_readObject(connection, &error,
                                           (reference + ".t").c_str(), IEC61850_FC_ST);
    if (!value || error != IED_ERROR_OK) {
        if (value) MmsValue_delete(value);
        return "read-failed";
    }
    char buffer[128];
    MmsValue_printToBuffer(value, buffer, sizeof(buffer));
    MmsValue_delete(value);
    return buffer;
}

std::string wait_for_change(IedConnection connection, const std::string& reference,
                            const std::string& previous) {
    std::string current = previous;
    for (int index = 0; index < 50 && current == previous; ++index) {
        std::this_thread::sleep_for(std::chrono::milliseconds(20));
        current = read_state(connection, reference);
    }
    return current;
}

bool valid(ControlObjectClient control) {
    return control && ControlObjectClient_getCtlValType(control) == MMS_BOOLEAN
        && ControlObjectClient_getControlModel(control) == CONTROL_MODEL_SBO_ENHANCED;
}
}

Json exercise_sbo(IedConnection connection, const std::string& action,
                  const std::string& reference, bool target, int wait_ms) {
    auto first = ControlObjectClient_create(reference.c_str(), connection);
    if (!valid(first)) {
        if (first) ControlObjectClient_destroy(first);
        return {{"ok", false}, {"error", "Control is not enhanced SBO Boolean"}};
    }
    auto value = MmsValue_newBoolean(target);
    ControlObjectClient_useConstantT(first, true);
    if (action.rfind("sbo-compat", 0) == 0)
        ControlObjectClient_setOrigin(first, "CONCEN", 2);
    auto before = read_state(connection, reference);
    auto pre_t = action == "sbo-compat-noop" ? read_time(connection, reference) : "";
    bool selected = ControlObjectClient_selectWithValue(first, value);
    auto during = action == "sbo-compat" && selected
        ? wait_for_change(connection, reference, before)
        : read_state(connection, reference);
    Json result = {{"selected", selected}, {"before", before}, {"after_select", during}};
    if (action == "sbo-compat") {
        result["ok"] = selected && before != during;
    } else if (action == "sbo-compat-noop") {
        auto before_t = read_time(connection, reference);
        std::this_thread::sleep_for(std::chrono::milliseconds(350));
        auto after_noop = read_state(connection, reference);
        auto after_t = read_time(connection, reference);
        result.update({{"before_t", before_t}, {"after_select_t", after_t}, {"pre_t", pre_t}});
        result["ok"] = selected && before == after_noop && before_t == after_t
            && pre_t == before_t;
    } else if (action == "sbo-sequence") {
        auto repeated = ControlObjectClient_create(reference.c_str(), connection);
        if (repeated) ControlObjectClient_useConstantT(repeated, true);
        bool second = valid(repeated) && ControlObjectClient_selectWithValue(repeated, value);
        auto repeated_error = repeated ? ControlObjectClient_getLastApplError(repeated) : LastApplError{};
        std::atomic<int> complete{0};
        ControlObjectClient_setCommandTerminationHandler(first, terminated, &complete);
        bool operated = selected && ControlObjectClient_operate(first, value, 0);
        for (int index = 0; operated && !complete && index < 100; ++index)
            std::this_thread::sleep_for(std::chrono::milliseconds(20));
        result.update({{"repeated", second}, {"repeated_add_cause", int(repeated_error.addCause)},
                       {"operated", operated && complete == 1},
                       {"after_operate", read_state(connection, reference)}});
        result["ok"] = selected && before == during && !second
            && repeated_error.addCause == ADD_CAUSE_OBJECT_ALREADY_SELECTED
            && operated && complete == 1;
        if (repeated) ControlObjectClient_destroy(repeated);
    } else if (action == "sbo-cancel") {
        bool canceled = selected && ControlObjectClient_cancel(first);
        result.update({{"canceled", canceled}, {"ok", canceled}});
    } else if (action == "sbo-timeout") {
        std::this_thread::sleep_for(std::chrono::milliseconds(wait_ms));
        result["ok"] = selected;
    } else if (action == "sbo-fail") {
        auto different = MmsValue_newBoolean(!target);
        bool operated = selected && ControlObjectClient_operate(first, different, 0);
        MmsValue_delete(different);
        result.update({{"operated", operated}, {"ok", selected && !operated}});
    } else if (action == "sbo-disconnect") {
        result["ok"] = selected;
    } else {
        result.update({{"ok", false}, {"error", "Unknown SBO action"}});
    }
    MmsValue_delete(value);
    ControlObjectClient_destroy(first);
    return result;
}
