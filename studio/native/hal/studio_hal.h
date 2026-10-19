/* Studio extensions of the libiec61850 HAL for event-driven simulated IEDs
 * (workstream G, proposed ADR in docs/architecture/go-engine/streams/g-native.md).
 *
 * Every executable links the same HAL. A thread that never selects a loop mode keeps the
 * vendor behaviour exactly; only the studio_ied server loop thread opts in.
 * - Wait mode: the next Handleset_waitReady on this thread also returns when the wake
 *   object is signalled (from any thread), so the loop can sleep until its next deadline.
 * - Poll mode: Handleset_waitReady on this thread never blocks. Used while the loop thread
 *   handles MMS traffic it already waited for, so an idle client costs no 10 ms wait.
 */
#ifndef STUDIO_HAL_H
#define STUDIO_HAL_H

#include <stdbool.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct sStudioWake* StudioWake;

StudioWake StudioWake_create(void);
void StudioWake_signal(StudioWake self);
void StudioWake_destroy(StudioWake self);

void StudioSocket_beginWait(StudioWake wake);
void StudioSocket_beginPoll(void);
void StudioSocket_endLoopMode(void);

/* True once after any TCP send could not take all bytes (the vendor keeps the rest and
 * retries on its connection tick). */
bool StudioSocket_takeBackpressure(void);

/* Called after every Ethernet frame the HAL sends (GOOSE publishing). */
typedef void (*StudioFrameObserver)(void* context);
void StudioEthernet_setSendObserver(StudioFrameObserver observer, void* context);

#ifdef __cplusplus
}
#endif

#endif
