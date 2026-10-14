// studio_sv_test_publisher: TEST-ONLY Sampled Values source for protocol tests.
// It is never packaged. It publishes one 9-2LE style stream (4 currents, 4 voltages)
// with the vendor SV publisher, or as raw frames when the Ed2 simulate bit is needed
// (libiec61850 1.6.1 cannot set that bit on layer-2 frames).
#include "ipc/framing.hpp"
#include "npcap.hpp"
extern "C" {
#include "hal_ethernet.h"
#include "sv_publisher.h"
}
#include <atomic>
#include <chrono>
#include <cmath>
#include <cstdio>
#include <stdexcept>
#include <thread>
#include <vector>

namespace {
struct Settings {
    std::string interface_id, sv_id, mac;
    uint16_t app_id = 0x4000, smp_rate = 80;
    uint32_t conf_rev = 1;
    int nominal = 50, skip_every = 0, smp_synch = 2;
    double amps = 1.0, volts = 63.5;
    bool invalid = false, simulate = false;
};

void parse_mac(const std::string& text, uint8_t* mac) {
    unsigned int p[6];
    if (std::sscanf(text.c_str(), "%x-%x-%x-%x-%x-%x", &p[0], &p[1], &p[2], &p[3], &p[4], &p[5]) != 6)
        throw std::runtime_error("Invalid MAC " + text);
    for (int i = 0; i < 6; ++i) mac[i] = uint8_t(p[i]);
}

// Eight INT32 + quality pairs: IA, IB, IC, IN (1 mA), VA, VB, VC, VN (10 mV).
std::vector<int32_t> sample(const Settings& s, int index) {
    const double pi = 3.14159265358979323846;
    double phase = 2.0 * pi * index / s.smp_rate;
    std::vector<int32_t> out;
    for (int kind = 0; kind < 2; ++kind) {
        double peak = (kind == 0 ? s.amps * 1000.0 : s.volts * 100.0) * std::sqrt(2.0);
        for (int p = 0; p < 3; ++p) out.push_back(int32_t(std::lround(peak * std::cos(phase - p * 2.0 * pi / 3.0))));
        out.push_back(0);
    }
    return out;
}

void put(std::vector<uint8_t>& b, std::initializer_list<int> bytes) { for (int x : bytes) b.push_back(uint8_t(x)); }

// Raw frame with reserved1 bit 15 set: header plus one ASDU in standard tag order.
std::vector<uint8_t> raw_frame(const Settings& s, const uint8_t* src, int count, const std::vector<int32_t>& values) {
    std::vector<uint8_t> asdu;
    put(asdu, {0x80, int(s.sv_id.size())});
    asdu.insert(asdu.end(), s.sv_id.begin(), s.sv_id.end());
    put(asdu, {0x82, 2, count >> 8, count & 0xff, 0x83, 4, int(s.conf_rev >> 24), int((s.conf_rev >> 16) & 0xff),
               int((s.conf_rev >> 8) & 0xff), int(s.conf_rev & 0xff), 0x85, 1, s.smp_synch, 0x87, 64});
    for (int32_t v : values) {
        uint32_t u = uint32_t(v);
        put(asdu, {int(u >> 24), int((u >> 16) & 0xff), int((u >> 8) & 0xff), int(u & 0xff), 0, 0, 0, 0});
    }
    std::vector<uint8_t> pdu;
    put(pdu, {0x60, int(asdu.size() + 7), 0x80, 1, 1, 0xa2, int(asdu.size() + 2), 0x30, int(asdu.size())});
    pdu.insert(pdu.end(), asdu.begin(), asdu.end());
    uint8_t dst[6];
    parse_mac(s.mac, dst);
    std::vector<uint8_t> frame(dst, dst + 6);
    frame.insert(frame.end(), src, src + 6);
    int length = int(pdu.size()) + 8;
    put(frame, {0x88, 0xba, s.app_id >> 8, s.app_id & 0xff, length >> 8, length & 0xff, 0x80, 0, 0, 0});
    frame.insert(frame.end(), pdu.begin(), pdu.end());
    return frame;
}

void publish(const Settings& s, std::atomic<bool>& running) {
    int per_second = s.smp_rate * s.nominal;
    CommParameters parameters{4, 0, s.app_id, {}};
    parse_mac(s.mac, parameters.dstAddress);
    SVPublisher publisher = nullptr;
    SVPublisher_ASDU asdu = nullptr;
    EthernetSocket socket = nullptr;
    uint8_t src[6] = {0};
    std::vector<int> offsets;
    if (s.simulate) {
        socket = Ethernet_createSocket(s.interface_id.c_str(), nullptr);
        Ethernet_getInterfaceMACAddress(s.interface_id.c_str(), src);
    } else {
        publisher = SVPublisher_create(&parameters, s.interface_id.c_str());
        if (!publisher) throw std::runtime_error("Could not open the adapter for publishing");
        asdu = SVPublisher_addASDU(publisher, s.sv_id.c_str(), nullptr, s.conf_rev);
        for (int i = 0; i < 8; ++i) {
            offsets.push_back(SVPublisher_ASDU_addINT32(asdu));
            offsets.push_back(SVPublisher_ASDU_addQuality(asdu));
        }
        SVPublisher_ASDU_setSmpCntWrap(asdu, uint16_t(per_second));
        SVPublisher_setupComplete(publisher);
        // smpSynch writes into the encoded buffer, which exists only after setup.
        SVPublisher_ASDU_setSmpSynch(asdu, uint16_t(s.smp_synch));
    }
    if (!publisher && !socket) throw std::runtime_error("Could not open the adapter for publishing");
    auto next = std::chrono::steady_clock::now();
    for (long long n = 0; running; ++n) {
        int count = int(n % per_second);
        auto values = sample(s, count);
        bool dropped = s.skip_every > 0 && n % s.skip_every == s.skip_every - 1;
        if (!dropped && socket) {
            auto frame = raw_frame(s, src, count, values);
            Ethernet_sendPacket(socket, frame.data(), int(frame.size()));
        } else if (!dropped) {
            for (int i = 0; i < 8; ++i) {
                SVPublisher_ASDU_setINT32(asdu, offsets[i * 2], values[i]);
                SVPublisher_ASDU_setQuality(asdu, offsets[i * 2 + 1], Quality(s.invalid ? QUALITY_VALIDITY_INVALID : 0));
            }
            SVPublisher_ASDU_setSmpCnt(asdu, uint16_t(count));
            SVPublisher_publish(publisher);
        }
        if (n % 20 == 19) {
            next += std::chrono::microseconds(20 * 1000000 / per_second);
            std::this_thread::sleep_until(next);
        }
    }
    if (publisher) SVPublisher_destroy(publisher);
    if (socket) Ethernet_destroySocket(socket);
}
}

int main() {
    protect_frame_output();
    std::atomic<bool> running{false};
    std::thread worker;
    Json message;
    while (read_frame(message)) {
        auto id = message.value("id", "");
        try {
            auto op = message.at("op").get<std::string>();
            if (op == "start") {
                Settings s;
                if (!npcap_available()) throw std::runtime_error("Npcap is not installed");
                s.interface_id = npcap_interface_for(message.at("adapter_guid").get<std::string>());
                s.mac = message.at("mac"); s.app_id = message.at("app_id"); s.sv_id = message.at("sv_id");
                s.conf_rev = message.value("conf_rev", 1u); s.smp_rate = message.value("smp_rate", uint16_t(80));
                s.skip_every = message.value("skip_every", 0); s.smp_synch = message.value("smp_synch", 2);
                s.amps = message.value("amps", 1.0); s.volts = message.value("volts", 63.5);
                s.invalid = message.value("invalid", false); s.simulate = message.value("simulate", false);
                running = true;
                worker = std::thread([s, &running] { publish(s, running); });
                emit_frame({{"id", id}, {"ok", true}, {"result", true}});
            } else if (op == "stop") {
                running = false;
                if (worker.joinable()) worker.join();
                emit_frame({{"id", id}, {"ok", true}});
                break;
            } else {
                throw std::runtime_error("Unknown operation " + op);
            }
        } catch (const std::exception& error) {
            emit_frame({{"id", id}, {"ok", false}, {"error", error.what()}});
        }
    }
    running = false;
    if (worker.joinable()) worker.join();
    return 0;
}
