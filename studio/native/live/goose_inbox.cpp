// Loopback inbox for frames forwarded by the Studio GOOSE engine (ADR-0073).
#include "goose_inbox.hpp"
#include <winsock2.h>
#include <ws2tcpip.h>
#include <mstcpip.h>
#include <stdexcept>
#include <vector>

#ifndef SIO_UDP_CONNRESET
#define SIO_UDP_CONNRESET _WSAIOW(IOC_VENDOR, 12)
#endif

GooseInbox::~GooseInbox() { close(); }

int GooseInbox::open(Handler on_frame) {
    if (running) throw std::runtime_error("GOOSE inbox already open");
    WSADATA data;
    if (WSAStartup(MAKEWORD(2, 2), &data) != 0) throw std::runtime_error("Windows sockets are unavailable");
    SOCKET s = ::socket(AF_INET, SOCK_DGRAM, IPPROTO_UDP);
    if (s == INVALID_SOCKET) {
        WSACleanup();
        throw std::runtime_error("Could not create the GOOSE inbox socket");
    }
    sockaddr_in local{};
    local.sin_family = AF_INET;
    local.sin_addr.s_addr = htonl(INADDR_LOOPBACK);
    int size = sizeof(local);
    if (bind(s, reinterpret_cast<sockaddr*>(&local), sizeof(local)) != 0 ||
        getsockname(s, reinterpret_cast<sockaddr*>(&local), &size) != 0) {
        closesocket(s);
        WSACleanup();
        throw std::runtime_error("Could not open the GOOSE inbox on 127.0.0.1");
    }
    BOOL off = FALSE;
    DWORD returned = 0;
    WSAIoctl(s, SIO_UDP_CONNRESET, &off, sizeof(off), nullptr, 0, &returned, nullptr, nullptr);
    int buffer = 4 << 20;
    setsockopt(s, SOL_SOCKET, SO_RCVBUF, reinterpret_cast<const char*>(&buffer), sizeof(buffer));
    socket_handle = static_cast<std::uintptr_t>(s);
    handler = std::move(on_frame);
    running = true;
    thread = std::thread([this] { run(); });
    return ntohs(local.sin_port);
}

void GooseInbox::run() {
    auto s = static_cast<SOCKET>(socket_handle);
    std::vector<uint8_t> buffer(65536);
    while (running) {
        fd_set ready;
        FD_ZERO(&ready);
        FD_SET(s, &ready);
        timeval wait{0, 50000};  // wake every 50 ms so close() never waits long
        if (select(0, &ready, nullptr, nullptr, &wait) <= 0) continue;
        int received = recvfrom(s, reinterpret_cast<char*>(buffer.data()), int(buffer.size()), 0, nullptr, nullptr);
        if (received < 2) continue;
        handler(buffer.data() + 1, received - 1, buffer[0] == 1);
    }
}

void GooseInbox::close() {
    if (!running) return;
    running = false;
    if (thread.joinable()) thread.join();
    closesocket(static_cast<SOCKET>(socket_handle));
    socket_handle = ~std::uintptr_t(0);
    WSACleanup();
}
