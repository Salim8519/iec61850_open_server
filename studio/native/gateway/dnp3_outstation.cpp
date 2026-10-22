// DNP3 outstation over TCP on opendnp3: binary, double-bit and analog inputs with
// events, binary output status for commands, and CROB commands via the backend; with typed
// commands also tap positions (analog input) and set points (analog output) (ADR-0107).
#include "dnp3_commands.hpp"
#include "protocol.hpp"
#include "traffic.hpp"
#include <opendnp3/DNP3Manager.h>
#include <opendnp3/channel/IChannelListener.h>
#include <opendnp3/logging/LogLevels.h>
#include <opendnp3/outstation/DefaultOutstationApplication.h>
#include <opendnp3/outstation/UpdateBuilder.h>
#include <algorithm>
#include <stdexcept>

using namespace opendnp3;

namespace {
// DNP3 flag bits.
constexpr uint8_t ONLINE = 0x01, COMM_LOST = 0x04, LOCAL_FORCED = 0x10, OVER_RANGE = 0x20;

uint8_t flags_of(const GatewayPoint& point) {
    uint8_t flags = quality_invalid(point) ? 0 : ONLINE;
    if (!point.source_ok) flags |= COMM_LOST;
    if (point.quality & 1024) flags |= LOCAL_FORCED;
    if ((point.quality & 4) && (point.kind == "float" || point.kind == "integer")) flags |= OVER_RANGE;
    return flags;
}

DNPTime time_of(const GatewayPoint& point) {
    return point.time_ms > 0 ? DNPTime(uint64_t(point.time_ms)) : DNPTime();
}

class TrafficLog : public ILogHandler {
public:
    void log(ModuleId, const char*, LogLevel level, char const*, char const* message) override {
        if (level == flags::APP_HEADER_RX || level == flags::APP_OBJECT_RX) traffic("rx", "master", message);
        else if (level == flags::APP_HEADER_TX || level == flags::APP_OBJECT_TX) traffic("tx", "master", message);
        else if (level == flags::ERR || level == flags::WARN) traffic("rx", "dnp3", std::string("warning: ") + message);
    }
};

class ChannelEvents : public IChannelListener {
public:
    void OnStateChange(ChannelState state) override {
        if (state == ChannelState::OPEN) connection_event("open", "master");
        if (state == ChannelState::CLOSED) connection_event("closed", "master");
    }
};

class Dnp3Outstation : public ProtocolServer {
public:
    Dnp3Outstation(PointTable& table, CommandBridge& bridge)
        : table(table), commands(std::make_shared<Dnp3Commands>(table, bridge)) {}
    ~Dnp3Outstation() override { stop(); }

    Json start(const Json& settings) override {
        auto bind = settings.value("bind", std::string("127.0.0.1"));
        int port = settings.value("port", 20000);
        int local = settings.value("outstation_address", 10), remote = settings.value("master_address", 1);
        if (local < 0 || local > 65519 || remote < 0 || remote > 65519)
            throw std::runtime_error("DNP3 addresses must be 0 to 65519");
        DatabaseConfig database;
        for (const auto& point : table.all()) {
            auto index = uint16_t(point.address);
            if (point.kind == "single") {
                database.binary_input[index].svariation = StaticBinaryVariation::Group1Var2;
                database.binary_input[index].evariation = EventBinaryVariation::Group2Var2;
            } else if (point.kind == "double") {
                database.double_binary[index].evariation = EventDoubleBinaryVariation::Group4Var2;
            } else if (point.kind == "float" || point.kind == "integer") {
                database.analog_input[index].clazz = PointClass::Class2;
                database.analog_input[index].svariation = point.kind == "float" ? StaticAnalogVariation::Group30Var5
                                                                                 : StaticAnalogVariation::Group30Var1;
                database.analog_input[index].evariation = point.kind == "float" ? EventAnalogVariation::Group32Var7
                                                                                : EventAnalogVariation::Group32Var3;
            } else if (point.kind == "step_position") {
                database.analog_input[index].clazz = PointClass::Class2;
                database.analog_input[index].svariation = StaticAnalogVariation::Group30Var1;
                database.analog_input[index].evariation = EventAnalogVariation::Group32Var3;
            } else if (point.kind == "command_setpoint") {
                database.analog_output_status[index].clazz = PointClass::Class0;
                database.analog_output_status[index].svariation = point.format == "int16"
                    ? StaticAnalogOutputStatusVariation::Group40Var2
                    : point.format == "int32" ? StaticAnalogOutputStatusVariation::Group40Var1
                                              : StaticAnalogOutputStatusVariation::Group40Var3;
            } else {
                database.binary_output_status[index].clazz = PointClass::Class0;
            }
        }
        manager = std::make_unique<DNP3Manager>(1, std::make_shared<TrafficLog>());
        auto levels = levels::NORMAL | flags::APP_HEADER_RX | flags::APP_HEADER_TX;
        try {
            channel = manager->AddTCPServer("gateway", levels, ServerAcceptMode::CloseExisting, IPEndpoint(bind, uint16_t(port)),
                                            std::make_shared<ChannelEvents>());
        } catch (const std::exception& error) {
            manager.reset();
            throw std::runtime_error("Could not listen on " + bind + ":" + std::to_string(port) + ": " + error.what());
        }
        OutstationStackConfig config(database);
        config.outstation.eventBufferConfig = EventBufferConfig::AllTypes(1000);
        config.outstation.params.allowUnsolicited = settings.value("unsolicited", true);
        config.link.LocalAddr = uint16_t(local);
        config.link.RemoteAddr = uint16_t(remote);
        outstation = channel->AddOutstation("outstation", commands, DefaultOutstationApplication::Create(), config);
        changed(table.all());
        outstation->Enable();
        return {{"protocol", "dnp3"}, {"bind", bind}, {"port", port}, {"outstation_address", local},
                {"master_address", remote}, {"unsolicited", config.outstation.params.allowUnsolicited}};
    }

    void changed(const std::vector<GatewayPoint>& points) override {
        if (!outstation) return;
        UpdateBuilder builder;
        for (const auto& point : points) {
            auto index = uint16_t(point.address);
            Flags flags(flags_of(point));
            if (point.kind == "single") builder.Update(Binary(point.value != 0.0, flags, time_of(point)), index);
            else if (point.kind == "double")
                builder.Update(DoubleBitBinary(DoubleBit(std::clamp(int(point.value), 0, 3)), flags, time_of(point)), index);
            else if (point.kind == "float" || point.kind == "integer" || point.kind == "step_position")
                builder.Update(Analog(point.value * point.scale, flags, time_of(point)), index);
        }
        outstation->Apply(builder.Build());
    }

    void stop() override {
        if (!manager) return;
        outstation.reset();
        channel.reset();
        manager->Shutdown();
        manager.reset();
    }

private:
    PointTable& table;
    std::shared_ptr<Dnp3Commands> commands;
    std::unique_ptr<DNP3Manager> manager;
    std::shared_ptr<IChannel> channel;
    std::shared_ptr<IOutstation> outstation;
};
}

std::unique_ptr<ProtocolServer> make_dnp3(PointTable& table, CommandBridge& bridge) {
    return std::make_unique<Dnp3Outstation>(table, bridge);
}
