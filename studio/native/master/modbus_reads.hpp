#pragma once
#include "ipc/framing.hpp"
#include "modbus_link.hpp"
#include <string>

// Modbus reads by ScadaBench as Modbus client. A range is {table, start, count, data_type,
// word_order}: table is coils (FC1), discrete_inputs (FC2), holding_registers (FC3) or
// input_registers (FC4); data_type bool, uint16, int16, uint32, int32 or float32.
struct ModbusAnswer {
    bool ok = false;
    uint8_t exception = 0;  // the server's exception code when ok is false
    std::vector<uint8_t> data;  // the bytes after the byte count
};

// Reads count bits or registers starting at start (no decoding).
ModbusAnswer modbus_read(ModbusLink& link, uint8_t unit, const std::string& table, int start, int count);
// Reads one range and turns it into master_values records; a refused range gives records
// with quality 1 ("no answer") and the reason in failure.
Json modbus_range_values(ModbusLink& link, uint8_t unit, const Json& range, std::string& failure);
// Registers per value of a data type (1 or 2).
int modbus_width(const std::string& data_type);
