// IEC 104 command objects for ScadaBench as master (C_SC, C_DC, C_SE, with or without time tag).
#include "iec104_command_objects.hpp"
#include <stdexcept>
extern "C" {
#include "hal_time.h"
}

CommandObject iec104_command_object(const std::string& kind, int ioa, double value, bool select, bool time_tagged,
                                    int qualifier) {
    struct sCP56Time2a now;
    CP56Time2a_createFromMsTimestamp(&now, Hal_getTimeInMs());
    CommandObject result;
    if (kind == "single") {
        result.type = time_tagged ? C_SC_TA_1 : C_SC_NA_1;
        result.object = time_tagged
            ? (InformationObject)SingleCommandWithCP56Time2a_create(nullptr, ioa, value != 0, select, qualifier, &now)
            : (InformationObject)SingleCommand_create(nullptr, ioa, value != 0, select, qualifier);
    } else if (kind == "double") {
        int state = int(value);
        if (state != 1 && state != 2) throw std::runtime_error("A double command is 1 (OFF / open) or 2 (ON / close)");
        result.type = time_tagged ? C_DC_TA_1 : C_DC_NA_1;
        result.object = time_tagged
            ? (InformationObject)DoubleCommandWithCP56Time2a_create(nullptr, ioa, state, select, qualifier, &now)
            : (InformationObject)DoubleCommand_create(nullptr, ioa, state, select, qualifier);
    } else if (kind == "regulating") {
        int step = int(value);
        if (step != 1 && step != 2) throw std::runtime_error("A regulating step command is 1 (Lower) or 2 (Raise)");
        auto command = step == 2 ? IEC60870_STEP_HIGHER : IEC60870_STEP_LOWER;
        result.type = time_tagged ? C_RC_TA_1 : C_RC_NA_1;
        result.object = time_tagged
            ? (InformationObject)StepCommandWithCP56Time2a_create(nullptr, ioa, command, select, qualifier, &now)
            : (InformationObject)StepCommand_create(nullptr, ioa, command, select, qualifier);
    } else if (kind == "setpoint_float") {
        result.type = time_tagged ? C_SE_TC_1 : C_SE_NC_1;
        result.object = time_tagged
            ? (InformationObject)SetpointCommandShortWithCP56Time2a_create(nullptr, ioa, float(value), select, qualifier, &now)
            : (InformationObject)SetpointCommandShort_create(nullptr, ioa, float(value), select, qualifier);
    } else if (kind == "setpoint_scaled") {
        if (value < -32768 || value > 32767) throw std::runtime_error("A scaled setpoint is a whole number from -32768 to 32767");
        result.type = time_tagged ? C_SE_TB_1 : C_SE_NB_1;
        result.object = time_tagged
            ? (InformationObject)SetpointCommandScaledWithCP56Time2a_create(nullptr, ioa, int(value), select, qualifier, &now)
            : (InformationObject)SetpointCommandScaled_create(nullptr, ioa, int(value), select, qualifier);
    } else if (kind == "setpoint_normalized") {
        if (value < -1 || value > 1) throw std::runtime_error("A normalised setpoint is a number from -1 to +1");
        result.type = time_tagged ? C_SE_TA_1 : C_SE_NA_1;
        result.object = time_tagged
            ? (InformationObject)SetpointCommandNormalizedWithCP56Time2a_create(nullptr, ioa, float(value), select, qualifier, &now)
            : (InformationObject)SetpointCommandNormalized_create(nullptr, ioa, float(value), select, qualifier);
    } else {
        throw std::runtime_error("Unknown IEC 104 command kind " + kind);
    }
    return result;
}
