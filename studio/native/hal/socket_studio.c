/* The vendor's Windows socket HAL, compiled unmodified under Vendor_* names for the two
 * functions Studio wraps (ADR-0006 pattern, as ethernet_pcap.c). Threads that never call
 * StudioSocket_begin* get the vendor functions unchanged; see studio_hal.h. */
#define Handleset_waitReady Vendor_Handleset_waitReady
#define Socket_write Vendor_Socket_write
#include "socket_win32.c"
#undef Handleset_waitReady
#undef Socket_write

#include "loop_wait.h"

enum { STUDIO_VENDOR = 0, STUDIO_WAIT = 1, STUDIO_POLL = 2 };

static __declspec(thread) int loopMode = STUDIO_VENDOR;
static __declspec(thread) StudioWake loopWake = NULL;
static __declspec(thread) uint64_t loopDeadline = 0;
static volatile LONG backpressure = 0;

void
StudioSocket_beginWait(StudioWake wake, uint64_t deadlineNs)
{
    loopMode = STUDIO_WAIT;
    loopWake = wake;
    loopDeadline = deadlineNs;
}

void StudioSocket_beginPoll(void) { loopMode = STUDIO_POLL; }
void StudioSocket_endLoopMode(void) { loopMode = STUDIO_VENDOR; loopWake = NULL; }

bool
StudioSocket_takeBackpressure(void)
{
    return InterlockedExchange(&backpressure, 0) != 0;
}

int
StudioHandleset_poll(HandleSet self)
{
    return Vendor_Handleset_waitReady(self, 0);
}

int
StudioHandleset_watch(HandleSet self, void* event)
{
    if (self == NULL || self->maxHandle == INVALID_SOCKET)
        return -1;
    /* Re-associating also records events that are already pending (WSAEventSelect), and
     * the poll below reports anything that was ready before the association. */
    for (u_int index = 0; index < self->handles.fd_count; ++index)
        WSAEventSelect(self->handles.fd_array[index], (WSAEVENT) event, FD_ACCEPT | FD_READ | FD_CLOSE);
    return Vendor_Handleset_waitReady(self, 0);
}

/* The server's sockets are non-blocking already (setSocketNonBlocking), so the event
 * association does not change how the vendor reads, accepts or writes. */
int
Handleset_waitReady(HandleSet self, unsigned int timeoutMs)
{
    if (loopMode == STUDIO_POLL)
        return Vendor_Handleset_waitReady(self, 0);
    if (loopMode == STUDIO_WAIT && loopWake != NULL)
        return StudioWake_wait(loopWake, self, loopDeadline);
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
