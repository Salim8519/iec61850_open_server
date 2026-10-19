// The canvas command, step for step the `control` operation of studio_probe (probe/main.cpp),
// so the Go engine's canvas operate sends the same PDUs as the Python engine's probe call.
// Only the wait for CommandTermination differs: the probe polls 100 x 20 ms; here the handler
// wakes the command at once, with the same 2 s limit.
#include "canvas_mux.hpp"
#include <chrono>
#include <condition_variable>
#include <mutex>

namespace {
struct Termination {
    std::mutex lock;
    std::condition_variable arrived;
    int state = 0;  // 0 waiting, 1 success, -1 failed
};

void terminated(void* parameter, ControlObjectClient control) {
    auto error = ControlObjectClient_getLastApplError(control);
    bool success = error.error == CONTROL_ERROR_NO_ERROR && error.addCause == ADD_CAUSE_UNKNOWN;
    auto& done = *static_cast<Termination*>(parameter);
    // Notify under the lock: once the waiter holds it again, this handler no longer touches `done`.
    std::lock_guard<std::mutex> guard(done.lock);
    done.state = success ? 1 : -1;
    done.arrived.notify_all();
}
}

Json canvas_command(IedConnection connection, const std::string& reference, bool requested) {
    auto control = ControlObjectClient_create(reference.c_str(), connection);
    bool ok = false;
    Termination done;
    if (control && ControlObjectClient_getCtlValType(control) == MMS_BOOLEAN) {
        ControlObjectClient_setCommandTerminationHandler(control, terminated, &done);
        auto value = MmsValue_newBoolean(requested);
        auto model = ControlObjectClient_getControlModel(control);
        bool selected = true;
        if (model == CONTROL_MODEL_SBO_NORMAL) selected = ControlObjectClient_select(control);
        if (model == CONTROL_MODEL_SBO_ENHANCED) selected = ControlObjectClient_selectWithValue(control, value);
        ok = selected && ControlObjectClient_operate(control, value, 0);
        if (ok && (model == CONTROL_MODEL_SBO_ENHANCED || model == CONTROL_MODEL_DIRECT_ENHANCED)) {
            std::unique_lock<std::mutex> guard(done.lock);
            done.arrived.wait_for(guard, std::chrono::milliseconds(2000), [&] { return done.state != 0; });
            ok = done.state == 1;
        }
        MmsValue_delete(value);
        // Detach the handler before the stack object goes away (a late termination is ignored).
        ControlObjectClient_setCommandTerminationHandler(control, nullptr, nullptr);
    }
    if (control) ControlObjectClient_destroy(control);
    return Json{{"ok", ok}};
}
