/* Studio extensions of the libiec61850 HAL for event-driven simulated IEDs
 * (ADR-0079; precise timers: speed lane N1, docs/architecture/go-engine/streams/speed-n1.md).
 *
 * Every executable links the same HAL. A thread that never selects a loop mode keeps the
 * vendor socket behaviour exactly; only the studio_ied server loop thread opts in.
 * - Wait mode: the next Handleset_waitReady on this thread sleeps until a socket of the set
 *   has a network event, the wake object is signalled (from any thread), or the precise
 *   clock reaches the deadline. A high-resolution waitable timer keeps the deadline to well
 *   under a millisecond without raising the machine-wide timer resolution.
 * - Poll mode: Handleset_waitReady on this thread never blocks. Used while the loop thread
 *   handles MMS traffic it already waited for, so an idle client costs no 10 ms wait.
 * Hal_getTimeInMs/Hal_getTimeInNs read GetSystemTimePreciseAsFileTime (hal/time_studio.c) in
 * every executable, so timestamps and deadlines no longer step with the 15.6 ms system tick.
 */
#ifndef STUDIO_HAL_H
#define STUDIO_HAL_H

#include <stdbool.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct sStudioWake* StudioWake;

StudioWake StudioWake_create(void);
/* False when Windows has no high-resolution waitable timers (before Windows 10 1803): waits
 * then use millisecond timeouts, which need the 1 ms system timer to stay on time. */
bool StudioWake_isHighResolution(StudioWake self);
void StudioWake_signal(StudioWake self);
void StudioWake_destroy(StudioWake self);

/* deadlineNs uses the Hal_getTimeInNs clock; a deadline in the past only polls. */
void StudioSocket_beginWait(StudioWake wake, uint64_t deadlineNs);
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
