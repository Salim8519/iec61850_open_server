#pragma once
// Passive reader of Sampled Values frame headers on the lab adapter.
// libiec61850 1.6.1 decodes SV for known streams but exposes neither unknown
// streams nor the IEC 61850-9-2 Ed2 simulate bit (reserved1, bit 15). This reader
// only looks at those header fields and the first svID; it never sends anything.
extern "C" {
#include "hal_ethernet.h"
}
#include <atomic>
#include <cstdint>
#include <map>
#include <mutex>
#include <string>
#include <thread>

struct SvHeader {
    uint16_t app_id = 0;
    std::string dst_mac, src_mac, sv_id;
    int vlan_id = -1;
    bool simulate = false;
};

// Parse one Ethernet frame; false when it is not a well-formed SV frame.
bool parse_sv_header(const uint8_t* frame, int size, SvHeader& out);

class SvHeaderReader {
public:
    ~SvHeaderReader();
    void start(const std::string& interface_id);
    void stop();
    // Latest header per "APPID/destination MAC" key.
    std::map<std::string, SvHeader> snapshot();
private:
    void run();
    EthernetSocket socket = nullptr;
    std::thread worker;
    std::atomic<bool> running{false};
    std::mutex lock;
    std::map<std::string, SvHeader> seen;
};

std::string sv_key(uint16_t app_id, const std::string& dst_mac);
