// IEC 104 command ASDU -> one decoded command (ADR-0107). Decoding only: the engine routes it.
#include "iec104_command_decode.hpp"
#include "iec104_objects.hpp"
#include <cstdio>

namespace {
bool legacy_type(int type_id) {
    return type_id == C_SC_NA_1 || type_id == C_SC_TA_1 || type_id == C_DC_NA_1 || type_id == C_DC_TA_1;
}

std::string number_text(double value) {
    char buffer[48];
    std::snprintf(buffer, sizeof(buffer), "%.6g", value);
    return buffer;
}
}

bool iec104_command_type(int type_id, bool typed) {
    if (legacy_type(type_id)) return true;
    if (!typed) return false;
    switch (type_id) {
    case C_RC_NA_1: case C_RC_TA_1:
    case C_SE_NA_1: case C_SE_NB_1: case C_SE_NC_1: case C_SE_TA_1: case C_SE_TB_1: case C_SE_TC_1: return true;
    default: return false;
    }
}

Iec104Command iec104_decode_command(CS101_ASDU asdu) {
    Iec104Command command;
    int type_id = CS101_ASDU_getTypeID(asdu);
    auto object = CS101_ASDU_getElement(asdu, 0);
    if (!object) return command;
    command.valid = true;
    command.ioa = InformationObject_getObjectAddress(object);
    std::string text;
    switch (type_id) {
    case C_SC_NA_1: case C_SC_TA_1:
        command.kind = "command_single";
        command.select = SingleCommand_isSelect((SingleCommand)object);
        command.state = SingleCommand_getState((SingleCommand)object) ? 2 : 1;
        command.value = command.state == 2;
        text = command.state == 2 ? "ON" : "OFF";
        break;
    case C_DC_NA_1: case C_DC_TA_1:
        command.kind = "command_double";
        command.select = DoubleCommand_isSelect((DoubleCommand)object);
        command.state = DoubleCommand_getState((DoubleCommand)object);
        command.value = command.state == 2;
        text = command.state == 2 ? "ON" : command.state == 1 ? "OFF" : "invalid";
        break;
    case C_RC_NA_1: case C_RC_TA_1:
        command.kind = "command_step";
        command.select = StepCommand_isSelect((StepCommand)object);
        command.state = int(StepCommand_getState((StepCommand)object));
        command.value = command.state;
        text = command.state == 2 ? "raise" : command.state == 1 ? "lower" : "invalid step";
        break;
    case C_SE_NA_1: case C_SE_TA_1:
        command.kind = "command_setpoint", command.format = "normalized";
        command.select = SetpointCommandNormalized_isSelect((SetpointCommandNormalized)object);
        command.value = double(SetpointCommandNormalized_getValue((SetpointCommandNormalized)object));
        break;
    case C_SE_NB_1: case C_SE_TB_1:
        command.kind = "command_setpoint", command.format = "scaled";
        command.select = SetpointCommandScaled_isSelect((SetpointCommandScaled)object);
        command.value = SetpointCommandScaled_getValue((SetpointCommandScaled)object);
        break;
    default:
        command.kind = "command_setpoint", command.format = "float";
        command.select = SetpointCommandShort_isSelect((SetpointCommandShort)object);
        command.value = double(SetpointCommandShort_getValue((SetpointCommandShort)object));
        break;
    }
    InformationObject_destroy(object);
    if (command.kind == "command_setpoint") text = "= " + number_text(command.value.get<double>());
    command.base = std::string(iec104_type_name(type_id)) + " IOA " + std::to_string(command.ioa) + " " + text;
    command.what = command.base + (command.select ? " (select)" : " (execute)");
    return command;
}
