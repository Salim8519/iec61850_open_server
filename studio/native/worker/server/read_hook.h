#pragma once
/* Observe MMS reads of one IedServer without changing their result (ADR-0066). */
#include "iec61850_server.h"

#ifdef __cplusplus
extern "C" {
#endif

/* domain: logical device name; item: MMS variable (LN$FC$...); peer: "ip:port" of the client. */
typedef void (*StudioReadObserver)(void* context, const char* domain, const char* item, const char* peer);

/* Wraps the access check libiec61850 installed; returns false if it could not be wrapped. */
bool studio_observe_reads(IedServer server, StudioReadObserver observer, void* context);

#ifdef __cplusplus
}
#endif
