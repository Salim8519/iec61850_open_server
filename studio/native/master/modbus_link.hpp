#pragma once
#include <cstdint>
#include <string>
#include <vector>

// One Modbus TCP connection from ScadaBench as Modbus client to a Modbus server (the device).
// request() sends one PDU (function code + data) in an MBAP frame and returns the answer PDU;
// it throws with plain words when the server does not answer in time or closes the link.
class ModbusLink {
public:
    ~ModbusLink() { close(); }
    void open(const std::string& host, int port, int timeout_ms, const std::string& local = "");
    std::vector<uint8_t> request(uint8_t unit, const std::vector<uint8_t>& pdu);
    void close();
    bool is_open() const { return handle != invalid(); }

private:
    static std::uintptr_t invalid() { return ~std::uintptr_t(0); }
    std::uintptr_t handle = invalid();
    uint16_t transaction = 0;
    std::string peer;
};

// Plain words for a Modbus exception code (1 illegal function, 2 illegal data address ...).
std::string modbus_exception_text(uint8_t code);
