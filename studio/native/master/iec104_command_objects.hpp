#pragma once
#include "ipc/framing.hpp"
#include <string>
extern "C" {
#include "cs101_information_objects.h"
}

// Build one IEC 104 command object sent by ScadaBench as master. kind: single, double,
// regulating (step), setpoint_float, setpoint_scaled or setpoint_normalized. With time_tagged
// the CP56Time2a variant (C_SC_TA_1 ...) is used, which some devices require. qualifier is the
// QU field (0 no additional definition, 1 short pulse, 2 long pulse, 3 persistent) 0-31.
struct CommandObject {
    InformationObject object = nullptr;
    int type = 0;
};
CommandObject iec104_command_object(const std::string& kind, int ioa, double value, bool select, bool time_tagged,
                                    int qualifier);
