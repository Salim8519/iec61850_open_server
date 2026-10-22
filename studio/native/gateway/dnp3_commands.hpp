#pragma once
#include "command_bridge.hpp"
#include "point_table.hpp"
#include <opendnp3/outstation/ICommandHandler.h>
#include <string>

// DNP3 commands of the gateway simulator. CROB (g12v1: latch on/off, pulse on/off, trip /
// close) drives single, double and step command points; with typed commands (ADR-0107)
// analog outputs (g41 var 1/2/3) drive set points. Without typed commands a CROB select is
// checked locally and only the operate goes to the engine, as before. With them select,
// operate, direct operate and direct operate without answer each go to the engine.
class Dnp3Commands : public opendnp3::ICommandHandler {
public:
    Dnp3Commands(PointTable& table, CommandBridge& bridge) : table(table), bridge(bridge) {}
    void Begin() override {}
    void End() override {}

    opendnp3::CommandStatus Select(const opendnp3::ControlRelayOutputBlock& command, uint16_t index) override;
    opendnp3::CommandStatus Operate(const opendnp3::ControlRelayOutputBlock& command, uint16_t index,
                                    opendnp3::IUpdateHandler& handler, opendnp3::OperateType type) override;

    opendnp3::CommandStatus Select(const opendnp3::AnalogOutputInt16& command, uint16_t index) override {
        return analog(double(command.value), 2, index, nullptr, "select");
    }
    opendnp3::CommandStatus Operate(const opendnp3::AnalogOutputInt16& command, uint16_t index,
                                    opendnp3::IUpdateHandler& handler, opendnp3::OperateType type) override {
        return analog(double(command.value), 2, index, &handler, function(type));
    }
    opendnp3::CommandStatus Select(const opendnp3::AnalogOutputInt32& command, uint16_t index) override {
        return analog(double(command.value), 1, index, nullptr, "select");
    }
    opendnp3::CommandStatus Operate(const opendnp3::AnalogOutputInt32& command, uint16_t index,
                                    opendnp3::IUpdateHandler& handler, opendnp3::OperateType type) override {
        return analog(double(command.value), 1, index, &handler, function(type));
    }
    opendnp3::CommandStatus Select(const opendnp3::AnalogOutputFloat32& command, uint16_t index) override {
        return analog(double(command.value), 3, index, nullptr, "select");
    }
    opendnp3::CommandStatus Operate(const opendnp3::AnalogOutputFloat32& command, uint16_t index,
                                    opendnp3::IUpdateHandler& handler, opendnp3::OperateType type) override {
        return analog(double(command.value), 3, index, &handler, function(type));
    }
    opendnp3::CommandStatus Select(const opendnp3::AnalogOutputDouble64&, uint16_t) override {
        return opendnp3::CommandStatus::NOT_SUPPORTED;
    }
    opendnp3::CommandStatus Operate(const opendnp3::AnalogOutputDouble64&, uint16_t, opendnp3::IUpdateHandler&,
                                    opendnp3::OperateType) override {
        return opendnp3::CommandStatus::NOT_SUPPORTED;
    }

private:
    PointTable& table;
    CommandBridge& bridge;

    static const char* function(opendnp3::OperateType type);
    opendnp3::CommandStatus check(const opendnp3::ControlRelayOutputBlock& command, uint16_t index);
    opendnp3::CommandStatus analog(double value, int variation, uint16_t index, opendnp3::IUpdateHandler* handler,
                                   const char* step);
};
