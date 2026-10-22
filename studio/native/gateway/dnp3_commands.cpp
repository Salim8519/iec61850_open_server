// DNP3 CROB and analog output commands via the engine (ADR-0057, ADR-0107).
#include "dnp3_commands.hpp"
#include "traffic.hpp"
#include <cstdio>

using namespace opendnp3;

namespace {
constexpr uint8_t ONLINE = 0x01;

bool target(const ControlRelayOutputBlock& command) {
    if (command.tcc == TripCloseCode::CLOSE) return true;
    if (command.tcc == TripCloseCode::TRIP) return false;
    return command.opType == OperationType::LATCH_ON || command.opType == OperationType::PULSE_ON;
}

const char* code(const ControlRelayOutputBlock& command) {
    if (command.tcc == TripCloseCode::CLOSE) return "close";
    if (command.tcc == TripCloseCode::TRIP) return "trip";
    switch (command.opType) {
    case OperationType::LATCH_ON: return "latch_on";
    case OperationType::LATCH_OFF: return "latch_off";
    case OperationType::PULSE_ON: return "pulse_on";
    case OperationType::PULSE_OFF: return "pulse_off";
    default: return "none";
    }
}

std::string name(const char* step) {
    std::string text = step;
    if (text == "direct") return "direct operate";
    if (text == "direct_no_ack") return "direct operate (no answer)";
    return text;
}
}

const char* Dnp3Commands::function(OperateType type) {
    if (type == OperateType::DirectOperate) return "direct";
    if (type == OperateType::DirectOperateNoAck) return "direct_no_ack";
    return "operate";
}

CommandStatus Dnp3Commands::check(const ControlRelayOutputBlock& command, uint16_t index) {
    auto point = table.at("command", index);
    if (!point) return CommandStatus::NOT_SUPPORTED;
    if (point->monitor_only) return CommandStatus::NOT_AUTHORIZED;
    if (command.opType == OperationType::NUL && command.tcc == TripCloseCode::NUL) return CommandStatus::FORMAT_ERROR;
    return CommandStatus::SUCCESS;
}

CommandStatus Dnp3Commands::Select(const ControlRelayOutputBlock& command, uint16_t index) {
    auto status = check(command, index);
    if (status == CommandStatus::SUCCESS && bridge.typed) {
        auto point = table.at("command", index);
        auto answer = bridge.ask_step(point->id, target(command), Json{{"function", "select"}, {"code", code(command)}},
                                      "dnp3", "master");
        traffic("rx", "master", "CROB select index " + std::to_string(index) + " " + code(command)
                + (answer.ok ? ": selected" : ": refused, " + answer.reason));
        return answer.ok ? CommandStatus::SUCCESS : CommandStatus::HARDWARE_ERROR;
    }
    traffic("rx", "master", "CROB select index " + std::to_string(index) + ": " + CommandStatusSpec::to_string(status));
    return status;
}

CommandStatus Dnp3Commands::Operate(const ControlRelayOutputBlock& command, uint16_t index, IUpdateHandler& handler,
                                    OperateType type) {
    auto status = check(command, index);
    if (status != CommandStatus::SUCCESS) {
        traffic("rx", "master", "CROB operate index " + std::to_string(index) + ": "
                + CommandStatusSpec::to_string(status));
        return status;
    }
    bool value = target(command);
    auto point = table.at("command", index);
    auto answer = bridge.typed
        ? bridge.ask_step(point->id, value, Json{{"function", function(type)}, {"code", code(command)}}, "dnp3", "master")
        : bridge.ask(point->id, value, "dnp3", "master");
    std::string how = bridge.typed ? " " + name(function(type)) + " " + code(command) : (value ? " CLOSE/ON" : " TRIP/OFF");
    traffic("rx", "master", "CROB operate index " + std::to_string(index) + how
            + (answer.ok ? ": operated" : ": failed, " + answer.reason));
    if (!answer.ok) return CommandStatus::HARDWARE_ERROR;
    handler.Update(BinaryOutputStatus(value, Flags(ONLINE)), index);
    return CommandStatus::SUCCESS;
}

CommandStatus Dnp3Commands::analog(double value, int variation, uint16_t index, IUpdateHandler* handler,
                                   const char* step) {
    char text[96];
    std::snprintf(text, sizeof(text), "Analog output g41v%d index %u = %.6g (%s)", variation, unsigned(index), value,
                  name(step).c_str());
    auto point = table.at("setpoint", index);
    if (!bridge.typed || !point) {
        traffic("rx", "master", std::string(text) + ": no set point at this index (NOT_SUPPORTED)");
        return CommandStatus::NOT_SUPPORTED;
    }
    if (point->monitor_only) {
        traffic("rx", "master", std::string(text) + ": real IED, and commands to real IEDs are off (NOT_AUTHORIZED)");
        return CommandStatus::NOT_AUTHORIZED;
    }
    auto answer = bridge.ask_step(point->id, value, Json{{"function", step}, {"variation", variation}}, "dnp3", "master");
    traffic("rx", "master", std::string(text) + (answer.ok ? ": done" : ": failed, " + answer.reason));
    if (!answer.ok) return CommandStatus::HARDWARE_ERROR;
    if (handler) handler->Update(AnalogOutputStatus(value, Flags(ONLINE)), index);
    return CommandStatus::SUCCESS;
}
