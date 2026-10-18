// ScadaBench as Modbus TCP client toward a Modbus server (the device). Modbus has no
// discovery: the user lists ranges, or "Find registers" probes a bounded address span.
#include "client.hpp"
#include "modbus_reads.hpp"
#include "traffic.hpp"
#include "value_events.hpp"
#include <chrono>
#include <cstring>
#include <thread>

namespace {
constexpr int MAX_PROBE = 10000;    // addresses per "Find registers"
constexpr int PROBE_GAP_MS = 100;   // at most 10 requests per second
constexpr int SPLIT_BELOW = 16;     // a refused small block is checked address by address

class ClientModbus : public ProtocolClient {
public:
    Json connect(const Json& settings) override {
        auto host = settings.at("host").get<std::string>();
        int port = settings.value("port", 502);
        unit = uint8_t(settings.value("unit_id", 1));
        peer = peer_name(host, port);
        emit_link("connecting", "Connecting to " + peer);
        link.open(host, port, settings.value("timeout_ms", 3000));
        emit_link("open", "Connected to " + peer);
        return {{"host", host}, {"port", port}, {"unit_id", unit}};
    }

    Json interrogate() override {
        throw std::runtime_error("Modbus has no interrogation: ScadaBench reads the ranges you list, or use Find registers");
    }

    Json poll(const Json& request) override {
        Json all = Json::array(), failed = Json::array();
        int number = 0;
        for (const auto& range : request.at("ranges")) {
            ++number;
            std::string failure;
            auto values = guarded([&] { return modbus_range_values(link, unit, range, failure); });
            if (!failure.empty()) failed.push_back({{"range", number}, {"text", failure}});
            for (auto& value : values) all.push_back(std::move(value));
        }
        emit_values(all);
        return {{"values", all.size()}, {"failed", failed}};
    }

    Json probe(const Json& request) override {
        auto table = request.at("table").get<std::string>();
        int start = request.at("start").get<int>(), end = request.at("end").get<int>();
        if (start < 0 || end > 65535 || end < start || end - start + 1 > MAX_PROBE)
            throw std::runtime_error("Find registers checks 1 to 10000 addresses from 0 to 65535");
        int block = table == "coils" || table == "discrete_inputs" ? 2000 : 125;
        traffic("info", peer, "Find registers: " + table + " " + std::to_string(start) + " to " + std::to_string(end));
        found = Json::array();
        requests = 0;
        for (int first = start; first <= end; first += block) scan(table, first, std::min(block, end - first + 1));
        traffic("info", peer, "Find registers done: " + std::to_string(found.size()) + " ranges answer, "
                                  + std::to_string(requests) + " requests");
        return {{"found", found}, {"requests", requests}};
    }

    Json command(const Json& request) override {
        auto kind = request.at("kind").get<std::string>();
        int address = request.at("address").get<int>();
        double value = request.at("value").get<double>();
        std::vector<uint8_t> pdu;
        if (kind == "coil") {
            pdu = {5, uint8_t(address >> 8), uint8_t(address), uint8_t(value ? 0xFF : 0), 0};
        } else if (kind == "register") {
            auto raw = uint16_t(int(value));
            pdu = {6, uint8_t(address >> 8), uint8_t(address), uint8_t(raw >> 8), uint8_t(raw)};
        } else if (kind == "registers_int32" || kind == "registers_float") {
            uint32_t raw;
            if (kind == "registers_float") { float number = float(value); std::memcpy(&raw, &number, sizeof raw); }
            else raw = uint32_t(int32_t(value));
            uint16_t high = uint16_t(raw >> 16), low = uint16_t(raw);
            if (request.value("word_order", std::string("big")) == "little") std::swap(high, low);
            pdu = {16, uint8_t(address >> 8), uint8_t(address), 0, 2, 4, uint8_t(high >> 8), uint8_t(high),
                   uint8_t(low >> 8), uint8_t(low)};
        } else {
            throw std::runtime_error("Unknown Modbus write " + kind);
        }
        traffic("tx", peer, "FC" + std::to_string(pdu[0]) + " write " + kind + " address " + std::to_string(address));
        auto answer = guarded([&] { return link.request(unit, pdu); });
        bool ok = !answer.empty() && answer[0] == pdu[0];
        std::string text = ok ? "the device confirmed the write" : answer.size() >= 2 ? "refused: " + modbus_exception_text(answer[1]) : "no answer";
        traffic("rx", peer, "FC" + std::to_string(pdu[0]) + " " + text);
        return {{"ok", ok}, {"steps", Json::array({Json{{"step", "write"}, {"ok", ok}, {"text", text}}})}};
    }

    void disconnect() override {
        if (!link.is_open()) return;
        link.close();
        emit_link("closed", "Disconnected by you");
    }

private:
    ModbusLink link;
    uint8_t unit = 1;
    std::string peer;
    Json found = Json::array();
    int requests = 0;

    template<class Task> auto guarded(Task task) -> decltype(task()) {
        try {
            return task();
        } catch (const std::runtime_error& error) {
            if (!link.is_open()) emit_link("closed", error.what());
            throw;
        }
    }

    // Read the block; when the server refuses an address, split it until the answering parts
    // are known. Small refused blocks are read one address at a time, so a probe never needs
    // many more requests than addresses.
    void scan(const std::string& table, int first, int count) {
        if (requests > 0) std::this_thread::sleep_for(std::chrono::milliseconds(PROBE_GAP_MS));
        ++requests;
        emit_frame({{"event", "master_probe"}, {"address", first}, {"requests", requests}});
        auto answer = guarded([&] { return modbus_read(link, unit, table, first, count); });
        if (answer.ok) {
            if (!found.empty() && found.back()["start"].get<int>() + found.back()["count"].get<int>() == first)
                found.back()["count"] = found.back()["count"].get<int>() + count;
            else
                found.push_back({{"table", table}, {"start", first}, {"count", count}});
        } else if (answer.exception == 1) {
            throw std::runtime_error("The server does not support reading " + table);
        } else if (count > SPLIT_BELOW) {
            scan(table, first, count / 2);
            scan(table, first + count / 2, count - count / 2);
        } else if (count > 1) {
            for (int address = first; address < first + count; ++address) scan(table, address, 1);
        }
    }
};
}

std::unique_ptr<ProtocolClient> make_client_modbus() { return std::make_unique<ClientModbus>(); }
