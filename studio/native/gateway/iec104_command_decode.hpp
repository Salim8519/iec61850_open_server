#pragma once
#include "ipc/framing.hpp"
#include <string>
extern "C" {
#include "iec60870_slave.h"
}

// Decoding of IEC 104 command ASDUs (ADR-0107): single, double, regulating step and the
// three set point types, each with or without a CP56Time2a time tag.
struct Iec104Command {
    bool valid = false;        // an information object was present
    int ioa = 0;
    bool select = false;       // S/E bit
    Json value;                // single/double: bool; step: state 0-3; set point: number
    int state = 0;             // single/double/step: 0-3 as the double-command state
    std::string kind;          // the gateway point kind this type id commands
    std::string format;        // set points: normalized, scaled or float
    std::string what;          // "C_SE_NC_1 IOA 5 = 12.5 (select)" for the traffic log
    std::string base;          // the same without the S/E words
};

// True for every type id handled with typed commands; legacy covers C_SC/C_DC only.
bool iec104_command_type(int type_id, bool typed);
Iec104Command iec104_decode_command(CS101_ASDU asdu);
