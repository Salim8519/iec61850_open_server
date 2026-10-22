// Modbus FC6 / FC16 register writes to set points (ADR-0107). Decoding only: each set point
// value goes to the engine, which operates the linked APC/INC control.
#include "modbus_pdu.hpp"
#include <cstdio>
#include <cstring>

namespace {
constexpr uint8_t ILLEGAL_FUNCTION = 1, ILLEGAL_ADDRESS = 2, ILLEGAL_VALUE = 3, DEVICE_FAILURE = 4;

int width(const GatewayPoint& point) { return point.format == "int16" || point.format == "uint16" ? 1 : 2; }

double decode(const GatewayPoint& point, uint16_t first, uint16_t second) {
    if (point.format == "int16") return double(int16_t(first));
    if (point.format == "uint16") return double(first);
    uint16_t high = point.low_word_first ? second : first, low = point.low_word_first ? first : second;
    uint32_t bits = (uint32_t(high) << 16) | low;
    if (point.format == "int32") return double(int32_t(bits));
    float value;
    std::memcpy(&value, &bits, 4);
    return double(value);
}

std::string number_text(double value) {
    char buffer[48];
    std::snprintf(buffer, sizeof(buffer), "%.6g", value);
    return buffer;
}

struct Write {
    GatewayPoint point;
    double value;
    int offset;
};
}

std::vector<uint8_t> ModbusApplication::write_registers(uint8_t function, int start,
                                                        const std::vector<uint16_t>& values, const std::string& peer,
                                                        std::string& summary) {
    bool any = false;
    for (const auto& point : table.all()) any = any || point.kind == "command_setpoint";
    if (!any) {  // as before ADR-0107: registers are read-only without set points
        summary = "FC" + std::to_string(function) + " is not supported by this gateway";
        return modbus_exception(function, ILLEGAL_FUNCTION);
    }
    int count = int(values.size());
    summary = "FC" + std::to_string(function) + " write register " + std::to_string(start)
              + (count > 1 ? ".." + std::to_string(start + count - 1) : "");
    std::vector<Write> writes;
    for (int offset = 0; offset < count;) {
        int at = start + offset;
        auto point = table.at("setpoint", at);
        if (!point) {
            summary += ": no set point starts at register " + std::to_string(at) + " (exception 2)";
            return modbus_exception(function, ILLEGAL_ADDRESS);
        }
        if (offset + width(*point) > count) {
            summary += ": the set point at register " + std::to_string(at)
                       + " has 2 registers; write both with FC16 (exception 3)";
            return modbus_exception(function, ILLEGAL_VALUE);
        }
        if (point->monitor_only) {
            summary += ": real IED, and commands to real IEDs are off in the gateway settings (exception 1)";
            return modbus_exception(function, ILLEGAL_FUNCTION);
        }
        writes.push_back({*point, decode(*point, values[offset], width(*point) == 2 ? values[offset + 1] : 0), offset});
        offset += width(*point);
    }
    for (const auto& write : writes) {
        auto answer = bridge.ask_step(write.point.id, write.value, Json{{"function", int(function)}}, "modbus", peer);
        if (!answer.ok) {
            summary += " = " + number_text(write.value) + ": failed, " + answer.reason + " (exception 4)";
            return modbus_exception(function, DEVICE_FAILURE);
        }
        std::lock_guard<std::mutex> guard(coil_lock);
        for (int index = 0; index < width(write.point); ++index)
            holding[start + write.offset + index] = values[write.offset + index];
        summary += " = " + number_text(write.value);
    }
    summary += ": operated";
    return {function};
}
