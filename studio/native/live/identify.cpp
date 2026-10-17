// MMS Identify plus the logical-device list of one server, on a temporary association.
#include "identify.hpp"
#include "browse.hpp"
#include <chrono>
#include <memory>
#include <stdexcept>
extern "C" {
#include "iec61850_client.h"
}

namespace {
std::string text(const char* value) { return value ? value : ""; }
}

Json identify_server(const Json& request) {
    auto ip = request.at("ip").get<std::string>();
    int port = request.value("port", 102);
    std::unique_ptr<sIedConnection, decltype(&IedConnection_destroy)> connection(IedConnection_create(),
                                                                               IedConnection_destroy);
    IedConnection_setConnectTimeout(connection.get(), request.value("timeout_ms", 3000u));
    IedConnection_setRequestTimeout(connection.get(), request.value("timeout_ms", 3000u));
    auto started = std::chrono::steady_clock::now();
    IedClientError error = IED_ERROR_OK;
    IedConnection_connect(connection.get(), &error, ip.c_str(), port);
    if (error != IED_ERROR_OK)
        return {{"ip", ip}, {"port", port}, {"identified", false}, {"reason", "no IEC 61850 association (MMS error "
                + std::to_string(int(error)) + ")"}};
    auto elapsed = std::chrono::duration_cast<std::chrono::milliseconds>(std::chrono::steady_clock::now() - started);
    Json result = {{"ip", ip}, {"port", port}, {"identified", true}, {"connect_ms", elapsed.count()},
                   {"vendor", ""}, {"model", ""}, {"revision", ""}, {"devices", Json::array()}};
    MmsError mms_error = MMS_ERROR_NONE;
    auto identity = MmsConnection_identify(IedConnection_getMmsConnection(connection.get()), &mms_error);
    if (identity) {
        result["vendor"] = text(identity->vendorName);
        result["model"] = text(identity->modelName);
        result["revision"] = text(identity->revision);
        MmsServerIdentity_destroy(identity);
    } else {
        result["identity_note"] = "the device does not answer MMS Identify (error " + std::to_string(int(mms_error)) + ")";
    }
    try {
        result["devices"] = browse_names(connection.get(), "");
    } catch (const std::exception& reason) {
        result["devices_note"] = reason.what();
    }
    IedConnection_close(connection.get());
    return result;
}
