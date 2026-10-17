#pragma once
#include "command_bridge.hpp"
#include "point_table.hpp"
#include <memory>
#include <string>
#include <vector>

// One SCADA-facing protocol server of the gateway simulator. Values only flow from the
// point table to the master; commands go through the bridge to the backend.
class ProtocolServer {
public:
    virtual ~ProtocolServer() = default;
    // settings: protocol-specific listener options; returns what was actually opened.
    virtual Json start(const Json& settings) = 0;
    // Points whose value or quality changed, for spontaneous/event reporting.
    virtual void changed(const std::vector<GatewayPoint>& points) = 0;
    virtual void stop() = 0;
};

std::unique_ptr<ProtocolServer> make_iec104(PointTable& table, CommandBridge& bridge);
std::unique_ptr<ProtocolServer> make_modbus(PointTable& table, CommandBridge& bridge);
std::unique_ptr<ProtocolServer> make_dnp3(PointTable& table, CommandBridge& bridge);
