// Connect, discover and read an external IED without changing it.
#include "mms_session.hpp"
#include "browse.hpp"
#include "mms_json.hpp"
#include <stdexcept>

namespace {
void state_changed(void*, IedConnection, IedConnectionState state) {
    const char* names[] = {"closed", "connecting", "connected", "closing"};
    int index = static_cast<int>(state);
    emit_frame({{"event", "live_connection"}, {"state", index >= 0 && index < 4 ? names[index] : "unknown"}});
}
}

MmsSession::~MmsSession() { close(); }

Json MmsSession::connect(const Json& request) {
    std::lock_guard<std::mutex> guard(io);
    if (connection) throw std::runtime_error("Already connected");
    auto ip = request.at("ip").get<std::string>();
    int port = request.value("port", 102);
    connection = IedConnection_create();
    IedConnection_setConnectTimeout(connection, request.value("timeout_ms", 5000u));
    IedConnection_setRequestTimeout(connection, request.value("request_timeout_ms", 5000u));
    IedConnection_installStateChangedHandler(connection, state_changed, nullptr);
    IedClientError error = IED_ERROR_OK;
    IedConnection_connect(connection, &error, ip.c_str(), port);
    if (error != IED_ERROR_OK) {
        IedConnection_destroy(connection);
        connection = nullptr;
        throw std::runtime_error("Could not connect to " + ip + ":" + std::to_string(port)
                                 + " (MMS error " + std::to_string(int(error)) + ")");
    }
    return {{"connected", true}, {"ip", ip}, {"port", port}};
}

Json MmsSession::discover() {
    std::lock_guard<std::mutex> guard(io);
    if (!connection) throw std::runtime_error("Not connected");
    auto result = browse_server(connection);
    if (!result.value("ok", false)) throw std::runtime_error(result.value("error", "Discovery failed"));
    return result;
}

Json MmsSession::read(const Json& request) {
    std::lock_guard<std::mutex> guard(io);
    if (!connection) throw std::runtime_error("Not connected");
    auto refs = request.at("refs");
    if (refs.size() > 2000) throw std::runtime_error("At most 2000 references per read");
    Json values = Json::array();
    for (const auto& item : refs) {
        auto ref = item.at("ref").get<std::string>();
        auto fc_name = item.at("fc").get<std::string>();
        auto fc = FunctionalConstraint_fromString(fc_name.c_str());
        IedClientError error = IED_ERROR_OK;
        auto value = IedConnection_readObject(connection, &error, ref.c_str(), fc);
        if (error != IED_ERROR_OK || !value) {
            values.push_back({{"ref", ref}, {"error", int(error)}});
            if (error == IED_ERROR_NOT_CONNECTED || error == IED_ERROR_CONNECTION_LOST) break;
            continue;
        }
        auto type = MmsValue_getType(value);
        if (type == MMS_DATA_ACCESS_ERROR) {
            values.push_back({{"ref", ref}, {"error", "access_error"},
                              {"access_error", int(MmsValue_getDataAccessError(value))}});
        } else if (type == MMS_STRUCTURE || type == MMS_ARRAY) {
            Json flat = Json::object();
            mms_flatten(value, nullptr, ref, flat);
            values.push_back({{"ref", ref}, {"structure", flat}});
        } else {
            values.push_back({{"ref", ref}, {"value", mms_scalar_json(value)}});
        }
        MmsValue_delete(value);
    }
    return {{"values", values}};
}

void MmsSession::close() {
    std::lock_guard<std::mutex> guard(io);
    if (connection) {
        IedConnection_close(connection);
        IedConnection_destroy(connection);
        connection = nullptr;
    }
}
