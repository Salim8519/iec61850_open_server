/* The vendor's Windows socket HAL, compiled unmodified under Vendor_* names for the two
 * functions Studio wraps (ADR-0006 pattern, as ethernet_pcap.c). Threads that never call
 * StudioSocket_begin* get the vendor functions unchanged; see studio_hal.h. */
#define Handleset_waitReady Vendor_Handleset_waitReady
#define Socket_write Vendor_Socket_write
#include "socket_win32.c"
#undef Handleset_waitReady
#undef Socket_write

#include <stdlib.h>
#include <string.h>
#include "studio_hal.h"

enum { STUDIO_VENDOR = 0, STUDIO_WAIT = 1, STUDIO_POLL = 2 };

struct sStudioWake {
    SOCKET socket; /* UDP on 127.0.0.1, connected to itself */
};

static __declspec(thread) int loopMode = STUDIO_VENDOR;
static __declspec(thread) StudioWake loopWake = NULL;
static volatile LONG backpressure = 0;

StudioWake
StudioWake_create(void)
{
    WSADATA data;
    if (WSAStartup(MAKEWORD(2, 2), &data) != 0)
        return NULL;
    StudioWake self = (StudioWake) calloc(1, sizeof(struct sStudioWake));
    SOCKET s = socket(AF_INET, SOCK_DGRAM, IPPROTO_UDP);
    struct sockaddr_in local;
    int length = sizeof(local);
    memset(&local, 0, sizeof(local));
    local.sin_family = AF_INET;
    local.sin_addr.s_addr = htonl(INADDR_LOOPBACK);
    u_long nonBlocking = 1;
    if (self == NULL || s == INVALID_SOCKET || bind(s, (struct sockaddr*) &local, sizeof(local)) != 0
            || getsockname(s, (struct sockaddr*) &local, &length) != 0
            || connect(s, (struct sockaddr*) &local, sizeof(local)) != 0
            || ioctlsocket(s, FIONBIO, &nonBlocking) != 0) {
        if (s != INVALID_SOCKET) closesocket(s);
        free(self);
        WSACleanup();
        return NULL;
    }
    self->socket = s;
    return self;
}

void
StudioWake_signal(StudioWake self)
{
    char token = 1;
    if (self) send(self->socket, &token, 1, 0);
}

void
StudioWake_destroy(StudioWake self)
{
    if (self == NULL) return;
    closesocket(self->socket);
    free(self);
    WSACleanup();
}

static void
drainWake(StudioWake self)
{
    char buffer[64];
    while (recv(self->socket, buffer, sizeof(buffer), 0) > 0) {
    }
}

void StudioSocket_beginWait(StudioWake wake) { loopMode = STUDIO_WAIT; loopWake = wake; }
void StudioSocket_beginPoll(void) { loopMode = STUDIO_POLL; }
void StudioSocket_endLoopMode(void) { loopMode = STUDIO_VENDOR; loopWake = NULL; }

bool
StudioSocket_takeBackpressure(void)
{
    return InterlockedExchange(&backpressure, 0) != 0;
}

/* Same contract as the vendor function: -1 for an empty set, else select()'s result. */
static int
waitWithWake(HandleSet self, unsigned int timeoutMs, StudioWake wake)
{
    if (self == NULL || self->maxHandle == INVALID_SOCKET)
        return -1;
    fd_set handles;
    memcpy((void*) &handles, &(self->handles), sizeof(fd_set));
    FD_SET(wake->socket, &handles);
    struct timeval timeout;
    timeout.tv_sec = timeoutMs / 1000;
    timeout.tv_usec = (timeoutMs % 1000) * 1000;
    int result = select(0, &handles, NULL, NULL, &timeout);
    if (result > 0 && FD_ISSET(wake->socket, &handles))
        drainWake(wake);
    return result;
}

int
Handleset_waitReady(HandleSet self, unsigned int timeoutMs)
{
    if (loopMode == STUDIO_POLL)
        return Vendor_Handleset_waitReady(self, 0);
    if (loopMode == STUDIO_WAIT && loopWake != NULL)
        return waitWithWake(self, timeoutMs, loopWake);
    return Vendor_Handleset_waitReady(self, timeoutMs);
}

int
Socket_write(Socket self, uint8_t* buf, int size)
{
    int sent = Vendor_Socket_write(self, buf, size);
    if (sent >= 0 && sent < size)
        InterlockedExchange(&backpressure, 1);
    return sent;
}
