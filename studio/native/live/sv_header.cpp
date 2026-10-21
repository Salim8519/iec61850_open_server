// Passive SV header reader: APPID, MAC addresses, VLAN, simulate bit and svID.
#include "sv_header.hpp"
#include "npcap.hpp"
#include <cstdio>
#include <winsock2.h>
#include <windows.h>
#define HAVE_REMOTE
#include <pcap.h>

namespace {
std::string mac_text(const uint8_t* mac) {
    char text[18];
    std::snprintf(text, sizeof(text), "%02X-%02X-%02X-%02X-%02X-%02X", mac[0], mac[1], mac[2], mac[3], mac[4], mac[5]);
    return text;
}

// Read a BER length at position; returns -1 on malformed input.
int ber_length(const uint8_t* data, int size, int& position) {
    if (position >= size) return -1;
    int first = data[position++];
    if (first < 0x80) return first;
    int bytes = first & 0x7f;
    if (bytes < 1 || bytes > 2 || position + bytes > size) return -1;
    int length = 0;
    for (int i = 0; i < bytes; ++i) length = (length << 8) | data[position++];
    return length;
}

// Enter a constructed element with the given tag; returns its end or -1.
int enter(const uint8_t* data, int size, int& position, uint8_t tag) {
    if (position >= size || data[position++] != tag) return -1;
    int length = ber_length(data, size, position);
    if (length < 0 || position + length > size) return -1;
    return position + length;
}
}

std::string sv_key(uint16_t app_id, const std::string& dst_mac) {
    return std::to_string(app_id) + "/" + dst_mac;
}

bool parse_sv_header(const uint8_t* frame, int size, SvHeader& out) {
    if (size < 22) return false;
    int position = 12;
    out.dst_mac = mac_text(frame);
    out.src_mac = mac_text(frame + 6);
    out.vlan_id = -1;
    if (frame[12] == 0x81 && frame[13] == 0x00) {
        out.vlan_id = ((frame[14] & 0x0f) << 8) | frame[15];
        position = 16;
    }
    if (position + 10 > size || frame[position] != 0x88 || frame[position + 1] != 0xba) return false;
    position += 2;
    out.app_id = uint16_t((frame[position] << 8) | frame[position + 1]);
    int length = (frame[position + 2] << 8) | frame[position + 3];
    uint16_t reserved1 = uint16_t((frame[position + 4] << 8) | frame[position + 5]);
    out.simulate = (reserved1 & 0x8000) != 0;
    int end = position + length;
    if (length < 8 || end > size) return false;
    position += 8;
    if (enter(frame, end, position, 0x60) < 0) return false;          // savPdu
    if (position >= end || frame[position++] != 0x80) return false;   // noASDU
    int skip = ber_length(frame, end, position);
    if (skip < 0) return false;
    position += skip;
    if (enter(frame, end, position, 0xa2) < 0) return false;          // seqASDU
    if (enter(frame, end, position, 0x30) < 0) return false;          // first ASDU
    if (position >= end || frame[position++] != 0x80) return false;   // svID
    int id_length = ber_length(frame, end, position);
    if (id_length < 0 || position + id_length > end) return false;
    out.sv_id.assign(reinterpret_cast<const char*>(frame + position), id_length);
    return true;
}

SvHeaderReader::~SvHeaderReader() { stop(); }

bool SvHeaderReader::start(const std::string& interface_id, SVReceiver sv_receiver) {
    // Same device name, snap length, promiscuous mode and filter as libiec61850's Npcap HAL.
    std::string device;
    for (const auto& adapter : npcap_adapters())
        if (std::to_string(adapter["index"].get<int>()) == interface_id) device = adapter["name"].get<std::string>();
    char error[PCAP_ERRBUF_SIZE];
    if (device.empty() || !(capture = pcap_open_live(device.c_str(), 65536, PCAP_OPENFLAG_PROMISCUOUS, 1, error)))
        return false;
    struct bpf_program program;
    bool ready = pcap_setnonblock(capture, 1, error) == 0 && pcap_setmintocopy(capture, 0) == 0 &&
                 pcap_compile(capture, &program, "(ether proto 0x88ba) or (vlan and ether proto 0x88ba)", 1, 0) == 0;
    if (ready) {
        ready = pcap_setfilter(capture, &program) == 0;
        pcap_freecode(&program);
    }
    stop_event = CreateEventW(nullptr, TRUE, FALSE, nullptr);
    if (!ready || !stop_event || !pcap_getevent(capture)) {
        stop();
        return false;
    }
    receiver = sv_receiver;
    worker = std::thread([this] { run(); });
    return true;
}

// Every waiting header frame. Npcap clears the receive event before each read, so a frame
// that arrives after the last (empty) read sets it again and wakes the loop.
bool SvHeaderReader::drain_headers() {
    bool any = false;
    struct pcap_pkthdr* info = nullptr;
    const u_char* data = nullptr;
    while (pcap_next_ex(capture, &info, &data) > 0) {
        any = true;
        SvHeader header;
        int size = info->caplen > 1600 ? 1600 : int(info->caplen);  // the former HAL receive buffer
        if (!parse_sv_header(data, size, header)) continue;
        std::lock_guard<std::mutex> guard(lock);
        if (seen.size() < 1000 || seen.count(sv_key(header.app_id, header.dst_mac)))
            seen[sv_key(header.app_id, header.dst_mac)] = header;
    }
    return any;
}

void SvHeaderReader::run() {
    HANDLE events[2] = {pcap_getevent(capture), stop_event};
    for (;;) {
        bool traffic = drain_headers();
        // The receiver's own capture waits up to its 1 ms read timeout for a frame still on its
        // way, so both captures end each wake having read the same frames.
        if (receiver)
            while (SVReceiver_tick(receiver)) traffic = true;
        // One more look 20 ms after traffic stops; with no traffic the loop sleeps until a
        // frame or stop arrives (no idle wakes).
        DWORD woke = WaitForMultipleObjects(2, events, FALSE, traffic ? 20 : INFINITE);
        if (woke != WAIT_OBJECT_0 && woke != WAIT_TIMEOUT) break;
    }
}

std::map<std::string, SvHeader> SvHeaderReader::snapshot() {
    std::lock_guard<std::mutex> guard(lock);
    return seen;
}

void SvHeaderReader::stop() {
    if (stop_event) SetEvent(stop_event);
    if (worker.joinable()) worker.join();
    if (capture) {
        pcap_close(capture);
        capture = nullptr;
    }
    if (stop_event) {
        CloseHandle(stop_event);
        stop_event = nullptr;
    }
    receiver = nullptr;
    std::lock_guard<std::mutex> guard(lock);
    seen.clear();
}
