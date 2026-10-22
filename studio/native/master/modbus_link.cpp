// Modbus TCP client link: MBAP framing over one blocking socket with timeouts.
#include "modbus_link.hpp"
#include "client.hpp"
#include <stdexcept>
#include <winsock2.h>
#include <ws2tcpip.h>

namespace {
SOCKET as_socket(std::uintptr_t handle) { return static_cast<SOCKET>(handle); }

void receive(SOCKET socket, uint8_t* buffer, int size, const std::string& peer) {
    int done = 0;
    while (done < size) {
        int count = recv(socket, reinterpret_cast<char*>(buffer + done), size - done, 0);
        if (count == 0) throw std::runtime_error("The Modbus server " + peer + " closed the connection");
        if (count < 0) throw std::runtime_error("No answer from the Modbus server " + peer + " in time");
        done += count;
    }
}
}

void ModbusLink::open(const std::string& host, int port, int timeout_ms) {
    WSADATA data;
    WSAStartup(MAKEWORD(2, 2), &data);
    peer = host + ":" + std::to_string(port);
    addrinfo hints{}, *found = nullptr;
    hints.ai_family = AF_INET;
    hints.ai_socktype = SOCK_STREAM;
    if (getaddrinfo(host.c_str(), std::to_string(port).c_str(), &hints, &found) != 0 || !found)
        throw std::runtime_error("The address " + host + " could not be found");
    SOCKET socket = ::socket(AF_INET, SOCK_STREAM, IPPROTO_TCP);
    u_long nonblocking = 1;
    ioctlsocket(socket, FIONBIO, &nonblocking);
    ::connect(socket, found->ai_addr, int(found->ai_addrlen));
    freeaddrinfo(found);
    fd_set writable, failed;
    FD_ZERO(&writable); FD_SET(socket, &writable);
    FD_ZERO(&failed); FD_SET(socket, &failed);
    timeval wait{timeout_ms / 1000, (timeout_ms % 1000) * 1000};
    if (select(0, nullptr, &writable, &failed, &wait) <= 0 || FD_ISSET(socket, &failed)) {
        int error = 0, size = sizeof error;
        bool refused = FD_ISSET(socket, &failed)
            && getsockopt(socket, SOL_SOCKET, SO_ERROR, reinterpret_cast<char*>(&error), &size) == 0
            && error == WSAECONNREFUSED;
        closesocket(socket);
        if (refused) throw std::runtime_error(refused_text(peer, "Modbus server"));
        throw std::runtime_error("No answer from " + peer + " within " + std::to_string(timeout_ms / 1000)
                                 + " s. Check the address and port, and that the Modbus server accepts this PC.");
    }
    nonblocking = 0;
    ioctlsocket(socket, FIONBIO, &nonblocking);
    DWORD timeout = DWORD(timeout_ms);
    setsockopt(socket, SOL_SOCKET, SO_RCVTIMEO, reinterpret_cast<const char*>(&timeout), sizeof timeout);
    handle = std::uintptr_t(socket);
}

std::vector<uint8_t> ModbusLink::request(uint8_t unit, const std::vector<uint8_t>& pdu) {
    if (!is_open()) throw std::runtime_error("Not connected to the Modbus server; press Connect first");
    uint16_t id = ++transaction;
    uint16_t length = uint16_t(pdu.size() + 1);
    std::vector<uint8_t> frame = {uint8_t(id >> 8), uint8_t(id), 0, 0, uint8_t(length >> 8), uint8_t(length), unit};
    frame.insert(frame.end(), pdu.begin(), pdu.end());
    if (send(as_socket(handle), reinterpret_cast<const char*>(frame.data()), int(frame.size()), 0) != int(frame.size())) {
        close();
        throw std::runtime_error("The connection to " + peer + " is broken");
    }
    try {
        while (true) {
            uint8_t header[7];
            receive(as_socket(handle), header, 7, peer);
            int size = (header[4] << 8 | header[5]) - 1;
            if (size < 1 || size > 253) throw std::runtime_error("The Modbus server " + peer + " sent a malformed frame");
            std::vector<uint8_t> answer(size);
            receive(as_socket(handle), answer.data(), size, peer);
            if ((header[0] << 8 | header[1]) == id) return answer;  // skip late answers to timed-out requests
        }
    } catch (const std::runtime_error&) {
        close();
        throw;
    }
}

void ModbusLink::close() {
    if (is_open()) closesocket(as_socket(handle));
    handle = invalid();
}

std::string modbus_exception_text(uint8_t code) {
    switch (code) {
    case 1: return "illegal function (the server does not support this request)";
    case 2: return "illegal data address (no such address)";
    case 3: return "illegal data value";
    case 4: return "server device failure";
    case 5: return "acknowledge (still working)";
    case 6: return "server device busy";
    case 10: return "gateway path unavailable";
    case 11: return "gateway target device failed to respond";
    default: return "exception " + std::to_string(code);
    }
}
