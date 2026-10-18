// Modbus range reads and value decoding for the master worker.
#include "modbus_reads.hpp"
#include <chrono>
#include <cstring>
#include <map>
#include <stdexcept>

namespace {
const std::map<std::string, uint8_t> FUNCTIONS = {
    {"coils", 1}, {"discrete_inputs", 2}, {"holding_registers", 3}, {"input_registers", 4}};
const std::map<std::string, std::string> GROUPS = {
    {"coils", "coil"}, {"discrete_inputs", "discrete_input"}, {"holding_registers", "holding_register"},
    {"input_registers", "input_register"}};

long long now_ms() {
    return std::chrono::duration_cast<std::chrono::milliseconds>(std::chrono::system_clock::now().time_since_epoch()).count();
}

uint16_t word(const std::vector<uint8_t>& data, int index) { return uint16_t(data[index * 2] << 8 | data[index * 2 + 1]); }

double decode(const std::vector<uint8_t>& data, int index, const std::string& type, bool little) {
    if (type == "uint16") return word(data, index);
    if (type == "int16") return int16_t(word(data, index));
    uint16_t first = word(data, index), second = word(data, index + 1);
    uint32_t raw = little ? (uint32_t(second) << 16 | first) : (uint32_t(first) << 16 | second);
    if (type == "uint32") return raw;
    if (type == "int32") return int32_t(raw);
    float value;
    std::memcpy(&value, &raw, sizeof value);
    return value;
}
}

int modbus_width(const std::string& data_type) {
    return data_type == "uint32" || data_type == "int32" || data_type == "float32" ? 2 : 1;
}

ModbusAnswer modbus_read(ModbusLink& link, uint8_t unit, const std::string& table, int start, int count) {
    auto function = FUNCTIONS.find(table);
    if (function == FUNCTIONS.end()) throw std::runtime_error("Unknown Modbus table " + table);
    auto answer = link.request(unit, {function->second, uint8_t(start >> 8), uint8_t(start), uint8_t(count >> 8),
                                      uint8_t(count)});
    ModbusAnswer result;
    if (answer.size() >= 2 && answer[0] == (function->second | 0x80)) {
        result.exception = answer[1];
        return result;
    }
    int expected = function->second <= 2 ? (count + 7) / 8 : count * 2;
    if (answer.size() < 2 || answer[0] != function->second || answer[1] != expected || int(answer.size()) != expected + 2)
        throw std::runtime_error("The Modbus server sent an answer that does not match the request");
    result.ok = true;
    result.data.assign(answer.begin() + 2, answer.end());
    return result;
}

Json modbus_range_values(ModbusLink& link, uint8_t unit, const Json& range, std::string& failure) {
    auto table = range.at("table").get<std::string>();
    auto type = range.value("data_type", std::string(table == "coils" || table == "discrete_inputs" ? "bool" : "uint16"));
    int start = range.at("start").get<int>(), count = range.at("count").get<int>();
    int width = type == "bool" ? 1 : modbus_width(type);
    auto answer = modbus_read(link, unit, table, start, count * width);
    failure = answer.ok ? "" : modbus_exception_text(answer.exception);
    Json values = Json::array();
    auto time = now_ms();
    bool little = range.value("word_order", std::string("big")) == "little";
    for (int index = 0; index < count; ++index) {
        Json value = nullptr;
        if (answer.ok) {
            value = type == "bool" ? Json((answer.data[index / 8] >> (index % 8)) & 1)
                                   : Json(decode(answer.data, index * width, type, little));
        }
        values.push_back({{"group", GROUPS.at(table)}, {"type", type}, {"address", start + index * width},
                          {"value", value}, {"quality", answer.ok ? 0 : 1}, {"time_ms", time}, {"cause", "polled"}});
    }
    return values;
}
