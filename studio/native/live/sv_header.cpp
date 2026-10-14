// Passive SV header reader: APPID, MAC addresses, VLAN, simulate bit and svID.
#include "sv_header.hpp"
#include <chrono>
#include <cstdio>
#include <vector>

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

void SvHeaderReader::start(const std::string& interface_id) {
    socket = Ethernet_createSocket(interface_id.c_str(), nullptr);
    if (!socket) return;
    Ethernet_setProtocolFilter(socket, 0x88ba);
    running = true;
    worker = std::thread([this] { run(); });
}

void SvHeaderReader::run() {
    std::vector<uint8_t> buffer(1600);
    while (running) {
        int size = Ethernet_receivePacket(socket, buffer.data(), int(buffer.size()));
        if (size <= 0) {
            std::this_thread::sleep_for(std::chrono::milliseconds(1));
            continue;
        }
        SvHeader header;
        if (!parse_sv_header(buffer.data(), size, header)) continue;
        std::lock_guard<std::mutex> guard(lock);
        if (seen.size() < 1000 || seen.count(sv_key(header.app_id, header.dst_mac)))
            seen[sv_key(header.app_id, header.dst_mac)] = header;
    }
}

std::map<std::string, SvHeader> SvHeaderReader::snapshot() {
    std::lock_guard<std::mutex> guard(lock);
    return seen;
}

void SvHeaderReader::stop() {
    running = false;
    if (worker.joinable()) worker.join();
    if (socket) {
        Ethernet_destroySocket(socket);
        socket = nullptr;
    }
    std::lock_guard<std::mutex> guard(lock);
    seen.clear();
}
