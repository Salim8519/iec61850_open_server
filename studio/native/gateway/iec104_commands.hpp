#pragma once
#include "command_bridge.hpp"
#include "point_table.hpp"
#include <optional>
#include <string>
extern "C" {
#include "iec60870_slave.h"
}

// IEC 104 commands: single and double (C_SC/C_DC), and with typed commands (ADR-0107)
// regulating step (C_RC) and set points (C_SE_NA/NB/NC), each with or without time tag.
// Without typed commands a select or cancel is confirmed locally, as before; with them every
// step goes to the engine, whose routing table selects, operates or cancels in the IED.
// Points of real IEDs are monitor-only while real commands are off, and are refused.
class Iec104Commands {
public:
    Iec104Commands(PointTable& table, CommandBridge& bridge) : table(table), bridge(bridge) {}
    static bool handles(int type_id, bool typed);
    void handle(IMasterConnection connection, CS101_ASDU asdu, const std::string& peer);
    bool typed() const { return bridge.typed; }
private:
    PointTable& table;
    CommandBridge& bridge;
    std::optional<GatewayPoint> command_at(int ioa) const;
};
