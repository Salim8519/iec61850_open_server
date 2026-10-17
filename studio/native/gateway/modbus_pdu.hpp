#pragma once
#include "command_bridge.hpp"
#include "point_table.hpp"
#include <cstdint>
#include <map>
#include <mutex>
#include <string>
#include <vector>

// Modbus application layer of the gateway simulator (spec gateway-simulator.md):
// discrete inputs (FC2) = single points; input registers (FC4) and the same view as
// holding registers (FC3) = double points (1 register) and float/integer points
// (2 registers, high word first); coils (FC1/5/15) = commands.
// An invalid float is served as NaN, because Modbus has no quality.
class ModbusApplication {
public:
    ModbusApplication(PointTable& table, CommandBridge& bridge) : table(table), bridge(bridge) {}
    // Returns the response PDU (function code + data, or exception) and a log line.
    std::vector<uint8_t> handle(const std::vector<uint8_t>& request, const std::string& peer, std::string& summary);
private:
    PointTable& table;
    CommandBridge& bridge;
    std::mutex coil_lock;
    std::map<int, bool> coil_state;  // last command accepted per coil

    std::vector<uint8_t> read_bits(uint8_t function, int start, int count, std::string& summary);
    std::vector<uint8_t> read_registers(uint8_t function, int start, int count, std::string& summary);
    std::vector<uint8_t> write_coils(uint8_t function, int start, const std::vector<bool>& values,
                                     const std::string& peer, std::string& summary);
};

std::vector<uint8_t> modbus_exception(uint8_t function, uint8_t code);
