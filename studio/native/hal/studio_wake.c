/* How the studio_ied server loop sleeps (speed lane N1). Socket network events, a wake event
 * from other threads and a high-resolution waitable timer are waited on together, so the loop
 * wakes within a fraction of a millisecond of its deadline although Windows keeps its default
 * 15.6 ms timer resolution. Without such timers (before Windows 10 1803) the wait uses
 * millisecond timeouts; the worker then raises the timer resolution as before (ADR-0070). */
#include <winsock2.h>
#include <windows.h>
#include <stdlib.h>
#include "hal_time.h"
#include "loop_wait.h"

#ifndef CREATE_WAITABLE_TIMER_HIGH_RESOLUTION
#define CREATE_WAITABLE_TIMER_HIGH_RESOLUTION 0x00000002
#endif

struct sStudioWake {
    HANDLE sockets; /* manual-reset: network events of the server's handle set */
    HANDLE wake;    /* auto-reset: signalled by other threads */
    HANDLE timer;   /* high-resolution waitable timer, NULL when Windows has none */
};

StudioWake
StudioWake_create(void)
{
    StudioWake self = (StudioWake) calloc(1, sizeof(struct sStudioWake));
    if (self == NULL)
        return NULL;
    self->sockets = CreateEventW(NULL, TRUE, FALSE, NULL);
    self->wake = CreateEventW(NULL, FALSE, FALSE, NULL);
    self->timer = CreateWaitableTimerExW(NULL, NULL, CREATE_WAITABLE_TIMER_HIGH_RESOLUTION,
                                         TIMER_ALL_ACCESS);
    if (self->sockets == NULL || self->wake == NULL) {
        StudioWake_destroy(self);
        return NULL;
    }
    return self;
}

bool StudioWake_isHighResolution(StudioWake self) { return self && self->timer != NULL; }

void
StudioWake_signal(StudioWake self)
{
    if (self) SetEvent(self->wake);
}

void
StudioWake_destroy(StudioWake self)
{
    if (self == NULL) return;
    if (self->sockets) CloseHandle(self->sockets);
    if (self->wake) CloseHandle(self->wake);
    if (self->timer) CloseHandle(self->timer);
    free(self);
}

/* Milliseconds for WaitForMultipleObjects, rounded up so the deadline has passed on return. */
static DWORD
milliseconds_until(uint64_t remainingNs)
{
    uint64_t ms = (remainingNs + 999999) / 1000000;
    return ms >= INFINITE ? INFINITE - 1 : (DWORD) ms;
}

int
StudioWake_wait(StudioWake self, HandleSet set, uint64_t deadlineNs)
{
    ResetEvent(self->sockets);
    int ready = StudioHandleset_watch(set, self->sockets);
    if (ready != 0)
        return ready; /* empty set, or traffic is already waiting */
    uint64_t now = Hal_getTimeInNs();
    if (deadlineNs <= now)
        return 0;
    HANDLE handles[3] = {self->sockets, self->wake, self->timer};
    DWORD count = 2;
    DWORD timeout = milliseconds_until(deadlineNs - now);
    if (self->timer) {
        LARGE_INTEGER due; /* negative: relative, in 100 ns units */
        due.QuadPart = -(LONGLONG) ((deadlineNs - now + 99) / 100);
        if (SetWaitableTimer(self->timer, &due, 0, NULL, NULL, FALSE)) {
            count = 3;
            timeout = INFINITE;
        }
    }
    DWORD result = WaitForMultipleObjects(count, handles, FALSE, timeout);
    if (count == 3 && result != WAIT_OBJECT_0 + 2)
        CancelWaitableTimer(self->timer); /* woken early: no stray timer expiry later */
    if (result == WAIT_FAILED)
        Sleep(1); /* never spin: the loop comes back within a millisecond */
    return StudioHandleset_poll(set);
}
