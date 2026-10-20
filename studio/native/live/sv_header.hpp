#pragma once
// Passive reader of Sampled Values frame headers on the lab adapter.
// libiec61850 1.6.1 decodes SV for known streams but exposes neither unknown
// streams nor the IEC 61850-9-2 Ed2 simulate bit (reserved1, bit 15). This reader
// only looks at those header fields and the first svID; it never sends anything.
extern "C" {
#include "sv_subscriber.h"
}
#include <cstdint>
#include <map>
#include <mutex>
#include <string>
#include <thread>

struct pcap;
struct SvHeader {
    uint16_t app_id = 0;
    std::string dst_mac, src_mac, sv_id;
    int vlan_id = -1;
    bool simulate = false;
};

// Parse one Ethernet frame; false when it is not a well-formed SV frame.
bool parse_sv_header(const uint8_t* frame, int size, SvHeader& out);

// Own Npcap capture of SV frames (same adapter and filter as libiec61850's receiver). Its
// thread sleeps on Npcap's receive event and a stop event, so an idle adapter costs no wakes;
// it also drives the libiec61850 SV receiver, started threadless, on every wake.
class SvHeaderReader {
public:
    ~SvHeaderReader();
    // interface_id: libiec61850's numeric Npcap adapter index. False when the capture cannot open.
    bool start(const std::string& interface_id, SVReceiver receiver);
    void stop();
    // Latest header per "APPID/destination MAC" key.
    std::map<std::string, SvHeader> snapshot();
private:
    void run();
    bool drain_headers();
    pcap* capture = nullptr;
    void* stop_event = nullptr;
    SVReceiver receiver = nullptr;
    std::thread worker;
    std::mutex lock;
    std::map<std::string, SvHeader> seen;
};

std::string sv_key(uint16_t app_id, const std::string& dst_mac);
