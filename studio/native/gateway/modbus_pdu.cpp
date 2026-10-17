// Modbus function codes 1-5 and 15 over the gateway point table.
#include "modbus_pdu.hpp"
#include <cmath>
#include <cstring>
#include <limits>

namespace {
constexpr uint8_t ILLEGAL_FUNCTION = 1, ILLEGAL_ADDRESS = 2, ILLEGAL_VALUE = 3, DEVICE_FAILURE = 4;

int word(const std::vector<uint8_t>& data, size_t at) { return (data[at] << 8) | data[at + 1]; }

void put_word(std::vector<uint8_t>& out, uint16_t value) {
    out.push_back(uint8_t(value >> 8));
    out.push_back(uint8_t(value & 0xFF));
}

std::string range(int start, int count) {
    return std::to_string(start) + (count > 1 ? ".." + std::to_string(start + count - 1) : "");
}
}

std::vector<uint8_t> modbus_exception(uint8_t function, uint8_t code) {
    return {uint8_t(function | 0x80), code};
}

std::vector<uint8_t> ModbusApplication::handle(const std::vector<uint8_t>& request, const std::string& peer,
                                               std::string& summary) {
    if (request.empty()) return modbus_exception(0, ILLEGAL_FUNCTION);
    uint8_t function = request[0];
    if ((function >= 1 && function <= 6 && request.size() < 5) || (function == 15 && request.size() < 6)) {
        summary = "FC" + std::to_string(function) + " too short";
        return modbus_exception(function, ILLEGAL_VALUE);
    }
    switch (function) {
    case 1: case 2: return read_bits(function, word(request, 1), word(request, 3), summary);
    case 3: case 4: return read_registers(function, word(request, 1), word(request, 3), summary);
    case 5: {
        int value = word(request, 3);
        if (value != 0xFF00 && value != 0x0000) {
            summary = "FC5 coil " + std::to_string(word(request, 1)) + ": value must be ON (FF00) or OFF (0000)";
            return modbus_exception(function, ILLEGAL_VALUE);
        }
        auto response = write_coils(function, word(request, 1), {value == 0xFF00}, peer, summary);
        if (response.size() == 2) return response;
        return std::vector<uint8_t>(request.begin(), request.begin() + 5);
    }
    case 15: {
        int start = word(request, 1), count = word(request, 3);
        if (count < 1 || count > 1968 || request.size() < size_t(6 + (count + 7) / 8)) {
            summary = "FC15 bad length";
            return modbus_exception(function, ILLEGAL_VALUE);
        }
        std::vector<bool> values;
        for (int index = 0; index < count; ++index) values.push_back((request[6 + index / 8] >> (index % 8)) & 1);
        auto response = write_coils(function, start, values, peer, summary);
        if (response.size() == 2) return response;
        return std::vector<uint8_t>(request.begin(), request.begin() + 5);
    }
    default:
        summary = "FC" + std::to_string(function) + " is not supported by this gateway";
        return modbus_exception(function, ILLEGAL_FUNCTION);
    }
}

std::vector<uint8_t> ModbusApplication::read_bits(uint8_t function, int start, int count, std::string& summary) {
    std::string name = function == 1 ? "FC1 read coils " : "FC2 read discrete inputs ";
    summary = name + range(start, count);
    if (count < 1 || count > 2000) return modbus_exception(function, ILLEGAL_VALUE);
    std::vector<uint8_t> out = {function, uint8_t((count + 7) / 8)};
    out.resize(2 + out[1], 0);
    bool any = false;
    for (int index = 0; index < count; ++index) {
        bool bit = false;
        if (function == 2) {
            auto point = table.at("single", start + index);
            if (point) { any = true; bit = point->value != 0.0; }
        } else {
            auto point = table.at("command", start + index);
            if (point) {
                any = true;
                std::lock_guard<std::mutex> guard(coil_lock);
                bit = coil_state[start + index];
            }
        }
        if (bit) out[2 + index / 8] |= uint8_t(1 << (index % 8));
    }
    if (!any) {
        summary += ": nothing mapped there (exception 2)";
        return modbus_exception(function, ILLEGAL_ADDRESS);
    }
    return out;
}

std::vector<uint8_t> ModbusApplication::read_registers(uint8_t function, int start, int count, std::string& summary) {
    summary = std::string(function == 3 ? "FC3 read holding registers " : "FC4 read input registers ")
              + range(start, count);
    if (count < 1 || count > 125) return modbus_exception(function, ILLEGAL_VALUE);
    std::map<int, uint16_t> registers;
    for (const auto& point : table.all()) {
        if (point.kind == "double") {
            registers[point.address] = uint16_t(point.value);
        } else if (point.kind == "float") {
            float value = quality_invalid(point) ? std::numeric_limits<float>::quiet_NaN()
                                                 : float(point.value * point.scale);
            uint32_t bits;
            std::memcpy(&bits, &value, 4);
            registers[point.address] = uint16_t(bits >> 16);
            registers[point.address + 1] = uint16_t(bits & 0xFFFF);
        } else if (point.kind == "integer") {
            auto value = uint32_t(int32_t(std::lround(point.value * point.scale)));
            registers[point.address] = uint16_t(value >> 16);
            registers[point.address + 1] = uint16_t(value & 0xFFFF);
        }
    }
    std::vector<uint8_t> out = {function, uint8_t(count * 2)};
    bool any = false;
    for (int index = 0; index < count; ++index) {
        auto found = registers.find(start + index);
        any = any || found != registers.end();
        put_word(out, found == registers.end() ? 0 : found->second);
    }
    if (!any) {
        summary += ": nothing mapped there (exception 2)";
        return modbus_exception(function, ILLEGAL_ADDRESS);
    }
    return out;
}

std::vector<uint8_t> ModbusApplication::write_coils(uint8_t function, int start, const std::vector<bool>& values,
                                                    const std::string& peer, std::string& summary) {
    summary = "FC" + std::to_string(function) + " write coil " + range(start, int(values.size()));
    for (size_t index = 0; index < values.size(); ++index) {
        auto point = table.at("command", start + int(index));
        if (!point) {
            summary += ": no command at coil " + std::to_string(start + index) + " (exception 2)";
            return modbus_exception(function, ILLEGAL_ADDRESS);
        }
        if (point->monitor_only) {
            summary += ": real IED, commands are never sent (exception 1)";
            return modbus_exception(function, ILLEGAL_FUNCTION);
        }
    }
    for (size_t index = 0; index < values.size(); ++index) {
        int coil = start + int(index);
        auto point = table.at("command", coil);
        auto answer = bridge.ask(point->id, values[index], "modbus", peer);
        if (!answer.ok) {
            summary += " " + std::string(values[index] ? "ON" : "OFF") + ": failed, " + answer.reason + " (exception 4)";
            return modbus_exception(function, DEVICE_FAILURE);
        }
        std::lock_guard<std::mutex> guard(coil_lock);
        coil_state[coil] = values[index];
    }
    summary += std::string(" ") + (values.size() == 1 ? (values[0] ? "ON" : "OFF") : "") + ": operated";
    return {function};
}
