#pragma once
#include "command_bridge.hpp"
#include "point_table.hpp"
#include <string>
extern "C" {
#include "iec60870_slave.h"
}

// Single and double commands (C_SC/C_DC, with or without time tag). Select is
// confirmed locally; execute asks the backend, which operates the simulated IED over
// MMS. Points of real IEDs are monitor-only and are always refused.
class Iec104Commands {
public:
    Iec104Commands(PointTable& table, CommandBridge& bridge) : table(table), bridge(bridge) {}
    static bool handles(int type_id);
    void handle(IMasterConnection connection, CS101_ASDU asdu, const std::string& peer);
private:
    PointTable& table;
    CommandBridge& bridge;
};
