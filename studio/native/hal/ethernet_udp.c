/* Send-only loopback Ethernet backend (ADR-0073). A simulated IED's GOOSE frames go
 * to the Studio GOOSE engine (studio_goose.exe) as UDP datagrams on 127.0.0.1; the
 * engine decides whether they also reach a network adapter. The socket binds to the
 * loopback address only, so it can never transmit on a network by itself. */
#include <winsock2.h>
#include <ws2tcpip.h>
#include <mstcpip.h>
#include <stdlib.h>
#include <string.h>
#include "ethernet_backend.h"

#ifndef SIO_UDP_CONNRESET
#define SIO_UDP_CONNRESET _WSAIOW(IOC_VENDOR, 12)
#endif

struct sStudioUdpSocket {
    SOCKET socket;
    struct sockaddr_in engine;
    unsigned long failures; /* counted, never printed per frame */
};

static int hex_value(char c)
{
    if (c >= '0' && c <= '9') return c - '0';
    if (c >= 'a' && c <= 'f') return c - 'a' + 10;
    if (c >= 'A' && c <= 'F') return c - 'A' + 10;
    return -1;
}

static bool parse_interface(const char* id, int* port, uint8_t* mac)
{
    if (!StudioUdp_isInterface(id)) return false;
    char* end = NULL;
    long value = strtol(id + 4, &end, 10);
    if (end == id + 4 || *end != ':' || value <= 0 || value > 65535) return false;
    const char* hex = end + 1;
    if (strlen(hex) != 12) return false;
    for (int i = 0; i < 6; i++) {
        int high = hex_value(hex[i * 2]), low = hex_value(hex[i * 2 + 1]);
        if (high < 0 || low < 0) return false;
        mac[i] = (uint8_t) (high << 4 | low);
    }
    *port = (int) value;
    return true;
}

bool StudioUdp_isInterface(const char* interfaceId)
{
    return interfaceId != NULL && strncmp(interfaceId, "udp:", 4) == 0;
}

bool StudioUdp_getMAC(const char* interfaceId, uint8_t* addr)
{
    int port;
    return parse_interface(interfaceId, &port, addr);
}

StudioUdpSocket StudioUdp_create(const char* interfaceId)
{
    int port;
    uint8_t mac[6];
    if (!parse_interface(interfaceId, &port, mac)) return NULL;
    WSADATA data;
    if (WSAStartup(MAKEWORD(2, 2), &data) != 0) return NULL;
    SOCKET s = socket(AF_INET, SOCK_DGRAM, IPPROTO_UDP);
    if (s == INVALID_SOCKET) { WSACleanup(); return NULL; }
    struct sockaddr_in local;
    memset(&local, 0, sizeof(local));
    local.sin_family = AF_INET;
    local.sin_addr.s_addr = htonl(INADDR_LOOPBACK);
    BOOL off = FALSE;
    DWORD returned = 0;
    WSAIoctl(s, SIO_UDP_CONNRESET, &off, sizeof(off), NULL, 0, &returned, NULL, NULL);
    int buffer = 1 << 20;
    setsockopt(s, SOL_SOCKET, SO_SNDBUF, (const char*) &buffer, sizeof(buffer));
    StudioUdpSocket self = (StudioUdpSocket) calloc(1, sizeof(struct sStudioUdpSocket));
    if (self == NULL || bind(s, (struct sockaddr*) &local, sizeof(local)) != 0) {
        free(self);
        closesocket(s);
        WSACleanup();
        return NULL;
    }
    self->socket = s;
    self->engine.sin_family = AF_INET;
    self->engine.sin_addr.s_addr = htonl(INADDR_LOOPBACK);
    self->engine.sin_port = htons((u_short) port);
    return self;
}

void StudioUdp_send(StudioUdpSocket self, const uint8_t* buffer, int size)
{
    if (sendto(self->socket, (const char*) buffer, size, 0, (struct sockaddr*) &self->engine,
               sizeof(self->engine)) == SOCKET_ERROR)
        self->failures++;
}

void StudioUdp_destroy(StudioUdpSocket self)
{
    closesocket(self->socket);
    free(self);
    WSACleanup();
}
