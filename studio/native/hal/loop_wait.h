/* Internal to Studio's HAL: how the studio_ied loop thread sleeps in wait mode
 * (socket_studio.c owns the vendor handle set, studio_wake.c the wake objects). */
#ifndef STUDIO_LOOP_WAIT_H
#define STUDIO_LOOP_WAIT_H

#include <stdint.h>
#include "hal_socket.h"
#include "studio_hal.h"

/* Associates every socket of the set with `event` (FD_ACCEPT, FD_READ, FD_CLOSE) and then
 * polls it without blocking: -1 for an empty set, else the number of ready sockets.
 * Readiness that arrives after this call signals `event`, so no network event is lost. */
int StudioHandleset_watch(HandleSet self, void* event);

/* Non-blocking poll with the vendor's select contract (-1 for an empty set). */
int StudioHandleset_poll(HandleSet self);

/* Sleeps until the set has a network event, the wake is signalled or deadlineNs
 * (Hal_getTimeInNs clock) passes. Same return contract as Handleset_waitReady. */
int StudioWake_wait(StudioWake self, HandleSet set, uint64_t deadlineNs);

#endif
