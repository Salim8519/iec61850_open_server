#pragma once
#include "ipc/framing.hpp"
#include <memory>
#include <stdexcept>
#include <string>

// ScadaBench as the SCADA master (ADR-0061): an IEC 104 controlling station, a DNP3 master
// or a Modbus TCP client toward a real gateway, RTU or outstation ("the device").
// Values and link changes go to the backend as events (value_events.hpp); requests block
// until the device answered or the request timed out.
class ProtocolClient {
public:
    virtual ~ProtocolClient() = default;
    // settings: host, port and protocol options; returns what was opened.
    virtual Json connect(const Json& settings) = 0;
    // Ask the device for every point: 104 general interrogation, DNP3 integrity poll.
    virtual Json interrogate() = 0;
    // Modbus only: read the given ranges once (the backend repeats it at the poll interval).
    virtual Json poll(const Json&) { throw std::runtime_error("Polling ranges is for Modbus TCP only"); }
    // Modbus only: find which addresses of one table answer, reading in blocks.
    virtual Json probe(const Json&) { throw std::runtime_error("Finding registers is for Modbus TCP only"); }
    // One command; returns the device's answers step by step.
    virtual Json command(const Json& request) = 0;
    virtual void disconnect() = 0;
};

std::unique_ptr<ProtocolClient> make_client_104();
std::unique_ptr<ProtocolClient> make_client_dnp3();
std::unique_ptr<ProtocolClient> make_client_modbus();
