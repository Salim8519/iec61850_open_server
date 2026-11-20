// Listen failures named by their cause (change 0227); see listen_error.hpp.
#include "listen_error.hpp"
#ifdef _WIN32
#include <winsock2.h>
#include <ws2tcpip.h>
#else
#include <arpa/inet.h>
#include <cerrno>
#include <netinet/in.h>
#include <sys/socket.h>
#include <unistd.h>
#define WSAEADDRINUSE EADDRINUSE
#define WSAEADDRNOTAVAIL EADDRNOTAVAIL
#define WSAEACCES EACCES
#define INVALID_SOCKET (-1)
#define closesocket close
#endif

std::string listen_failure(const std::string& bind, int port, int code) {
    auto where = "Could not listen on " + bind + ":" + std::to_string(port);
    switch (code) {
    case WSAEADDRINUSE:
        return where + ". Another program uses this port. Close that program or choose another port.";
    case WSAEADDRNOTAVAIL:
        return where + ": " + bind + " is not an address of this PC now. If it is the gateway's own "
               "address on the ScadaBench Lab card, press Start again; otherwise choose another network.";
    case WSAEACCES:
        return where + ": Windows does not let programs use this port (another program holds it "
               "for itself, or Windows keeps it reserved). Choose another port.";
    default:
        return where + " (error " + std::to_string(code) + "). Another program may use this port.";
    }
}

int listen_probe(const std::string& bind, int port) {
#ifdef _WIN32
    WSADATA data;
    if (WSAStartup(MAKEWORD(2, 2), &data) != 0) return 0;
#endif
    sockaddr_in address{};
    address.sin_family = AF_INET;
    address.sin_port = htons(uint16_t(port));
    int code = 0;
    auto probe = socket(AF_INET, SOCK_STREAM, IPPROTO_TCP);
    if (probe != INVALID_SOCKET && inet_pton(AF_INET, bind.c_str(), &address.sin_addr) == 1) {
        int reuse = 1;
        setsockopt(probe, SOL_SOCKET, SO_REUSEADDR, reinterpret_cast<const char*>(&reuse), sizeof(reuse));
        if (::bind(probe, reinterpret_cast<sockaddr*>(&address), sizeof(address)) != 0) {
#ifdef _WIN32
            code = WSAGetLastError();
#else
            code = errno;
#endif
        }
    }
    if (probe != INVALID_SOCKET) closesocket(probe);
#ifdef _WIN32
    WSACleanup();
#endif
    return code;
}
