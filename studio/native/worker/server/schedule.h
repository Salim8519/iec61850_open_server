/* When libiec61850's periodic server tasks are next due (workstream G event loop). */
#pragma once
#include <stdint.h>
#include "iec61850_server.h"

#ifdef __cplusplus
extern "C" {
#endif

#define STUDIO_NO_DEADLINE UINT64_MAX

/* Earliest time (Hal_getTimeInMs) at which IedServer_performPeriodicTasks has control or
 * report work, or a connection tick has queued reports to send. `now` when work is pending. */
uint64_t studio_server_deadline(IedServer server, uint64_t now);

/* Distinct MinTime/MaxTime values of the enabled GOOSE control blocks; returns the count. */
int studio_goose_intervals(IedServer server, uint32_t* intervals, int capacity);

/* What IedServer_stop does besides stopping its thread: disable every report block
 * (raising the same RCB events) and stop listening. Call after the loop thread ended. */
void studio_server_stop(IedServer server);

#ifdef __cplusplus
}
#endif
