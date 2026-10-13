// Independent client executable used to verify the worker over TCP/MMS.
#include <nlohmann/json.hpp>
#include <iostream>
#include <string>
#include <atomic>
#include <chrono>
#include <thread>
#include "reports.hpp"
#include "browse.hpp"
#include "sbo_controls.hpp"
extern "C" {
#include "iec61850_client.h"
}
using Json = nlohmann::json;

void terminated(void* parameter, ControlObjectClient control) {
    auto error = ControlObjectClient_getLastApplError(control);
    bool success = error.error == CONTROL_ERROR_NO_ERROR && error.addCause == ADD_CAUSE_UNKNOWN;
    static_cast<std::atomic<int>*>(parameter)->store(success ? 1 : -1);
}

int main(int argc, char** argv) {
    if (argc < 4 || (std::string(argv[3]) != "browse" && argc < 5)) {
        std::cerr << "usage: studio_probe IP PORT browse|read|control|reports|sbo-* [REFERENCE] [FC|VALUE]";
        return 2;
    }
    auto connection = IedConnection_create();
    IedClientError error;
    IedConnection_connect(connection, &error, argv[1], std::stoi(argv[2]));
    if (error != IED_ERROR_OK) {
        std::cout << Json({{"ok", false}, {"error", int(error)}}).dump();
        IedConnection_destroy(connection); return 1;
    }
    Json output;
    if (std::string(argv[3]) == "browse") {
        output = browse_server(connection);
    } else if (std::string(argv[3]) == "read") {
        auto fc = FunctionalConstraint_fromString(argc > 5 ? argv[5] : "ST");
        auto value = IedConnection_readObject(connection, &error, argv[4], fc);
        output = {{"ok", error == IED_ERROR_OK}, {"error", int(error)}};
        if (value) {
            char buffer[4096];
            MmsValue_printToBuffer(value, buffer, sizeof(buffer));
            output["value"] = buffer;
            MmsValue_delete(value);
        }
    } else if (std::string(argv[3]) == "control") {
        auto control = ControlObjectClient_create(argv[4], connection);
        bool ok = false;
        if (control && ControlObjectClient_getCtlValType(control) == MMS_BOOLEAN) {
            std::atomic<int> complete{0};
            ControlObjectClient_setCommandTerminationHandler(control, terminated, &complete);
            auto value = MmsValue_newBoolean(argc > 5 && std::string(argv[5]) == "true");
            auto model = ControlObjectClient_getControlModel(control);
            bool selected = true;
            if (model == CONTROL_MODEL_SBO_NORMAL) selected = ControlObjectClient_select(control);
            if (model == CONTROL_MODEL_SBO_ENHANCED) selected = ControlObjectClient_selectWithValue(control, value);
            ok = selected && ControlObjectClient_operate(control, value, 0);
            if (ok && (model == CONTROL_MODEL_SBO_ENHANCED || model == CONTROL_MODEL_DIRECT_ENHANCED)) {
                for (int i = 0; i < 100 && !complete; ++i)
                    std::this_thread::sleep_for(std::chrono::milliseconds(20));
                ok = complete == 1;
            }
            MmsValue_delete(value);
            ControlObjectClient_destroy(control);
        }
        output = {{"ok", ok}};
    } else if (std::string(argv[3]).rfind("sbo-", 0) == 0) {
        output = exercise_sbo(connection, argv[3], argv[4],
                              argc > 5 && std::string(argv[5]) == "true",
                              argc > 6 ? std::stoi(argv[6]) : 0);
    } else if (std::string(argv[3]) == "reports") {
        output = receive_reports(connection, argv[4]);
    } else output = {{"ok", false}, {"error", "Unknown probe operation"}};
    IedConnection_close(connection);
    IedConnection_destroy(connection);
    std::cout << output.dump() << std::endl;
    return output.value("ok", false) ? 0 : 1;
}
