// IEC 60870-5-104 slave (controlled station) on lib60870: interrogation, spontaneous
// changes with time tags, clock sync, read and commands (single, double, step, set point).
#include "iec104_commands.hpp"
#include "iec104_objects.hpp"
#include "protocol.hpp"
#include "traffic.hpp"
#include <stdexcept>
extern "C" {
#include "cs104_slave.h"
#include "hal_time.h"
}

namespace {
std::string peer_of(IMasterConnection connection) {
    char buffer[64] = {0};
    IMasterConnection_getPeerAddress(connection, buffer, sizeof(buffer));
    return buffer;
}

class Iec104Server : public ProtocolServer {
public:
    Iec104Server(PointTable& table, CommandBridge& bridge) : table(table), commands(table, bridge) {}
    ~Iec104Server() override { stop(); }

    Json start(const Json& settings) override {
        common_address = settings.value("common_address", 1);
        if (common_address < 1 || common_address > 65534) throw std::runtime_error("Common address must be 1 to 65534");
        auto bind = settings.value("bind", std::string("127.0.0.1"));
        int port = settings.value("port", 2404);
        slave = CS104_Slave_create(settings.value("queue", 1000), settings.value("queue", 1000));
        CS104_Slave_setLocalAddress(slave, bind.c_str());
        CS104_Slave_setLocalPort(slave, port);
        CS104_Slave_setServerMode(slave, CS104_MODE_CONNECTION_IS_REDUNDANCY_GROUP);
        CS104_Slave_setMaxOpenConnections(slave, settings.value("max_connections", 4));
        auto apci = CS104_Slave_getConnectionParameters(slave);
        apci->t1 = settings.value("t1", apci->t1);
        apci->t2 = settings.value("t2", apci->t2);
        apci->t3 = settings.value("t3", apci->t3);
        apci->k = settings.value("k", apci->k);
        apci->w = settings.value("w", apci->w);
        parameters = CS104_Slave_getAppLayerParameters(slave);
        CS104_Slave_setConnectionRequestHandler(slave, &Iec104Server::on_request, this);
        CS104_Slave_setConnectionEventHandler(slave, &Iec104Server::on_event, this);
        CS104_Slave_setInterrogationHandler(slave, &Iec104Server::on_interrogation, this);
        CS104_Slave_setClockSyncHandler(slave, &Iec104Server::on_clock, this);
        CS104_Slave_setReadHandler(slave, &Iec104Server::on_read, this);
        CS104_Slave_setASDUHandler(slave, &Iec104Server::on_asdu, this);
        CS104_Slave_start(slave);
        if (!CS104_Slave_isRunning(slave)) {
            CS104_Slave_destroy(slave);
            slave = nullptr;
            throw std::runtime_error("Could not listen on " + bind + ":" + std::to_string(port)
                                     + ". Another program may use this port.");
        }
        return {{"protocol", "iec104"}, {"bind", bind}, {"port", port}, {"common_address", common_address},
                {"t1", apci->t1}, {"t2", apci->t2}, {"t3", apci->t3}, {"k", apci->k}, {"w", apci->w}};
    }

    void changed(const std::vector<GatewayPoint>& points) override {
        if (!slave) return;
        for (const auto& point : points) {
            if (is_command(point.kind)) continue;
            auto asdu = CS101_ASDU_create(parameters, false, CS101_COT_SPONTANEOUS, 0, common_address, false, false);
            auto object = iec104_object(point, true);
            CS101_ASDU_addInformationObject(asdu, object);
            traffic("tx", "all", std::string(iec104_type_name(CS101_ASDU_getTypeID(asdu))) + " spont IOA "
                    + std::to_string(point.address) + " = " + iec104_value_text(point));
            InformationObject_destroy(object);
            CS104_Slave_enqueueASDU(slave, asdu);
            CS101_ASDU_destroy(asdu);
        }
    }

    void stop() override {
        if (!slave) return;
        CS104_Slave_stop(slave);
        CS104_Slave_destroy(slave);
        slave = nullptr;
    }

private:
    PointTable& table;
    Iec104Commands commands;
    CS104_Slave slave = nullptr;
    CS101_AppLayerParameters parameters = nullptr;
    int common_address = 1;

    bool wrong_address(IMasterConnection connection, CS101_ASDU asdu) {
        int address = CS101_ASDU_getCA(asdu);
        if (address == common_address || address == 0xFFFF) return false;
        traffic("rx", peer_of(connection), std::string(iec104_type_name(CS101_ASDU_getTypeID(asdu)))
                + " for unknown common address " + std::to_string(address) + ": refused (COT 46)");
        CS101_ASDU_setCOT(asdu, CS101_COT_UNKNOWN_CA);
        CS101_ASDU_setNegative(asdu, true);
        IMasterConnection_sendASDU(connection, asdu);
        return true;
    }

    static bool on_request(void*, const char* address) {
        traffic("rx", address, "TCP connection request accepted");
        return true;
    }

    static void on_event(void*, IMasterConnection connection, CS104_PeerConnectionEvent event) {
        static const char* names[] = {"connected", "disconnected", "started (STARTDT)", "stopped (STOPDT)"};
        auto peer = peer_of(connection);
        connection_event(event == CS104_CON_EVENT_CONNECTION_CLOSED ? "closed" : "open", peer);
        if (event >= 0 && event <= 3) traffic("rx", peer, std::string("Master ") + names[event]);
    }

    static bool on_interrogation(void* parameter, IMasterConnection connection, CS101_ASDU asdu, uint8_t qoi) {
        auto& self = *static_cast<Iec104Server*>(parameter);
        auto peer = peer_of(connection);
        if (self.wrong_address(connection, asdu)) return true;
        if (qoi != IEC60870_QOI_STATION) {
            traffic("rx", peer, "C_IC_NA_1 group " + std::to_string(qoi - 20) + ": not configured, refused");
            IMasterConnection_sendACT_CON(connection, asdu, true);
            return true;
        }
        IMasterConnection_sendACT_CON(connection, asdu, false);
        auto parameters = IMasterConnection_getApplicationLayerParameters(connection);
        int sent = 0;
        for (const char* kind : {"single", "double", "float", "integer", "step_position"}) {
            CS101_ASDU answer = nullptr;
            for (const auto& point : self.table.all()) {
                if (point.kind != kind) continue;
                auto object = iec104_object(point, false);
                if (!answer) answer = CS101_ASDU_create(parameters, false, CS101_COT_INTERROGATED_BY_STATION, 0,
                                                        self.common_address, false, false);
                if (!CS101_ASDU_addInformationObject(answer, object)) {
                    IMasterConnection_sendASDU(connection, answer);
                    CS101_ASDU_destroy(answer);
                    answer = CS101_ASDU_create(parameters, false, CS101_COT_INTERROGATED_BY_STATION, 0,
                                               self.common_address, false, false);
                    CS101_ASDU_addInformationObject(answer, object);
                }
                InformationObject_destroy(object);
                ++sent;
            }
            if (answer) {
                IMasterConnection_sendASDU(connection, answer);
                CS101_ASDU_destroy(answer);
            }
        }
        IMasterConnection_sendACT_TERM(connection, asdu);
        traffic("rx", peer, "C_IC_NA_1 station interrogation: answered with " + std::to_string(sent) + " points");
        return true;
    }

    static bool on_clock(void*, IMasterConnection connection, CS101_ASDU, CP56Time2a time) {
        traffic("rx", peer_of(connection), "C_CS_NA_1 clock sync accepted (the PC clock is not changed)");
        CP56Time2a_setFromMsTimestamp(time, Hal_getTimeInMs());
        return true;
    }

    static bool on_read(void* parameter, IMasterConnection connection, CS101_ASDU asdu, int ioa) {
        auto& self = *static_cast<Iec104Server*>(parameter);
        if (self.wrong_address(connection, asdu)) return true;
        for (const auto& point : self.table.all()) {
            if (point.address != ioa || is_command(point.kind)) continue;
            auto answer = CS101_ASDU_create(IMasterConnection_getApplicationLayerParameters(connection), false,
                                            CS101_COT_REQUEST, 0, self.common_address, false, false);
            auto object = iec104_object(point, false);
            CS101_ASDU_addInformationObject(answer, object);
            InformationObject_destroy(object);
            IMasterConnection_sendASDU(connection, answer);
            CS101_ASDU_destroy(answer);
            traffic("rx", peer_of(connection), "C_RD_NA_1 IOA " + std::to_string(ioa) + " = " + iec104_value_text(point));
            return true;
        }
        traffic("rx", peer_of(connection), "C_RD_NA_1 IOA " + std::to_string(ioa) + ": unknown IOA (COT 47)");
        CS101_ASDU_setCOT(asdu, CS101_COT_UNKNOWN_IOA);
        CS101_ASDU_setNegative(asdu, true);
        IMasterConnection_sendASDU(connection, asdu);
        return true;
    }

    static bool on_asdu(void* parameter, IMasterConnection connection, CS101_ASDU asdu) {
        auto& self = *static_cast<Iec104Server*>(parameter);
        if (!Iec104Commands::handles(CS101_ASDU_getTypeID(asdu), self.commands.typed())) return false;
        if (self.wrong_address(connection, asdu)) return true;
        self.commands.handle(connection, asdu, peer_of(connection));
        return true;
    }
};
}

std::unique_ptr<ProtocolServer> make_iec104(PointTable& table, CommandBridge& bridge) {
    return std::make_unique<Iec104Server>(table, bridge);
}
