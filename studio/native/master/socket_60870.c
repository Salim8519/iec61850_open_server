/* lib60870's Windows socket HAL, compiled unmodified with Socket_connect renamed (ADR-0006
 * pattern: the vendor tree is never edited; cmake/master_socket_support.cmake pins the file).
 *
 * The vendor Socket_connect waits for the socket to become writable only. Windows reports a
 * failed non-blocking connect in the except set instead, so a refused connection (RST) was
 * noticed only when the whole connect timeout (IEC 104 t0) had passed. This Socket_connect
 * sends the same SYN and the same IEC 104 bytes afterwards, and differs only in when a failure
 * is reported:
 *  - refused (WSAECONNREFUSED): at once. Windows itself answers an RST by repeating the SYN for
 *    about 2 s, so the first attempt asks for no SYN repetition (SIO_TCP_INITIAL_RTO); when that
 *    attempt only times out (no answer, ~1 s), a second socket with the default Windows
 *    retransmissions continues until the connect timeout, like the vendor's single socket.
 *  - no answer (unreachable or silent host), or any other error: at the connect timeout, as before.
 * A socket bound to a local address (CS104_Connection_setLocalAddress) keeps one attempt. */
#define Socket_connect Vendor_Socket_connect
#include "socket_win32.c"
#undef Socket_connect

#define STUDIO_SIO_TCP_INITIAL_RTO _WSAIOW(IOC_VENDOR, 17)
#define STUDIO_RTO_UNSPECIFIED_RTT ((USHORT) -1)
#define STUDIO_RTO_NO_SYN_RETRANSMISSIONS ((UCHAR) -2)

typedef struct {
    USHORT Rtt;
    UCHAR MaxSynRetransmissions;
} StudioInitialRto; /* TCP_INITIAL_RTO_PARAMETERS of mstcpip.h (Windows 8 and later) */

enum { ATTEMPT_CONNECTED = 0, ATTEMPT_TIMEOUT = -1, ATTEMPT_NOT_STARTED = -2 };

/* 1 when the last Socket_connect failed because the device refused it (change 0178), so the
 * master can say "nothing is listening" instead of "no answer within t0". One connect at a time. */
int studio_connect_refused = 0;

static bool
studioIsBound(Socket self)
{
    struct sockaddr_storage local;
    int length = sizeof(local);
    return getsockname(self->fd, (struct sockaddr*)&local, &length) == 0;
}

static bool
studioSkipSynRetransmissions(Socket self)
{
    StudioInitialRto parameters = {STUDIO_RTO_UNSPECIFIED_RTT, STUDIO_RTO_NO_SYN_RETRANSMISSIONS};
    DWORD returned = 0;
    return WSAIoctl(self->fd, STUDIO_SIO_TCP_INITIAL_RTO, &parameters, sizeof(parameters), NULL, 0, &returned,
                    NULL, NULL) == 0;
}

static DWORD
studioRemaining(ULONGLONG deadline)
{
    ULONGLONG now = GetTickCount64();
    return now >= deadline ? 0 : (DWORD)(deadline - now);
}

/* One connect on self->fd until the deadline: ATTEMPT_CONNECTED, ATTEMPT_TIMEOUT, ATTEMPT_NOT_STARTED
 * (vendor connectAsync failed and already closed the socket) or the positive Winsock error. */
static int
studioAttempt(Socket self, const char* address, int port, ULONGLONG deadline)
{
    if (Socket_connectAsync(self, address, port) == false)
        return ATTEMPT_NOT_STARTED;

    DWORD remaining = studioRemaining(deadline);
    struct timeval timeout;
    timeout.tv_sec = remaining / 1000;
    timeout.tv_usec = (remaining % 1000) * 1000;

    fd_set writable, failed;
    FD_ZERO(&writable);
    FD_ZERO(&failed);
    FD_SET(self->fd, &writable);
    FD_SET(self->fd, &failed);

    if (select(0, NULL, &writable, &failed, &timeout) < 1)
        return ATTEMPT_TIMEOUT;

    int so_error = 0;
    int length = sizeof(so_error);

    if (getsockopt(self->fd, SOL_SOCKET, SO_ERROR, (char*)&so_error, &length) == SOCKET_ERROR)
        return WSAGetLastError() > 0 ? WSAGetLastError() : WSAEINVAL;

    if (FD_ISSET(self->fd, &writable) && so_error == 0)
        return ATTEMPT_CONNECTED;

    return so_error > 0 ? so_error : WSAEINVAL;
}

static void
studioCloseFd(Socket self)
{
    if (self->fd != INVALID_SOCKET)
        closesocket(self->fd);

    self->fd = INVALID_SOCKET;
}

bool
Socket_connect(Socket self, const char* address, int port)
{
    ULONGLONG deadline = GetTickCount64() + self->connectTimeout;
    bool quickAttempt = !studioIsBound(self) && studioSkipSynRetransmissions(self);
    int result;

    studio_connect_refused = 0;
    result = studioAttempt(self, address, port, deadline);

    if (result == ATTEMPT_CONNECTED)
        return true;

    if (result == ATTEMPT_NOT_STARTED)
        return false;

    studioCloseFd(self);

    if (quickAttempt && result != WSAECONNREFUSED && result != ATTEMPT_TIMEOUT && studioRemaining(deadline) > 0)
    {
        /* The SYN got no answer: continue with Windows' own SYN retransmissions until t0. */
        self->fd = socket(AF_INET, SOCK_STREAM, 0);

        if (self->fd == INVALID_SOCKET)
            return false;

        result = studioAttempt(self, address, port, deadline);

        if (result == ATTEMPT_CONNECTED)
            return true;

        if (result == ATTEMPT_NOT_STARTED)
            return false;

        studioCloseFd(self);
    }

    /* Only a refusal is reported early; every other failure keeps the vendor's full connect timeout. */
    studio_connect_refused = result == WSAECONNREFUSED;
    if (result != WSAECONNREFUSED)
        Sleep(studioRemaining(deadline));

    return false;
}
