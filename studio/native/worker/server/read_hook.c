/* Observe MMS reads without changing their result (ADR-0066).
 *
 * libiec61850 installs one read-access check on its MMS server (setting groups and the
 * optional user handler). Installing IedServer_setReadAccessHandler would change the error
 * returned for nonexistent objects, so instead this wraps the installed check: the original
 * runs first and its answer is returned unchanged; a successful read is only counted.
 * Uses the private MmsServer layout of the pinned libiec61850 commit (native/README.md).
 */
#include "read_hook.h"
#include "mms_server_internal.h"
#include <stdlib.h>

typedef struct {
    MmsReadAccessHandler original;
    void* original_parameter;
    StudioReadObserver observer;
    void* context;
} ReadHook;

static MmsDataAccessError observe(void* parameter, MmsDomain* domain, char* variableId,
        MmsServerConnection connection, bool isDirectAccess)
{
    ReadHook* hook = (ReadHook*) parameter;
    MmsDataAccessError result = hook->original
        ? hook->original(hook->original_parameter, domain, variableId, connection, isDirectAccess)
        : DATA_ACCESS_ERROR_SUCCESS;
    if (result == DATA_ACCESS_ERROR_SUCCESS && connection != NULL && variableId != NULL) {
        const char* peer = MmsServerConnection_getClientAddress(connection);
        hook->observer(hook->context, domain ? MmsDomain_getName(domain) : "", variableId, peer ? peer : "");
    }
    return result;
}

bool studio_observe_reads(IedServer server, StudioReadObserver observer, void* context)
{
    MmsServer mms = IedServer_getMmsServer(server);
    if (mms == NULL || observer == NULL)
        return false;
    /* Lives as long as the process: one IED server per worker, and MMS threads may still
       hold the pointer while the server stops. */
    ReadHook* hook = (ReadHook*) calloc(1, sizeof(ReadHook));
    if (hook == NULL)
        return false;
    hook->original = mms->readAccessHandler;
    hook->original_parameter = mms->readAccessHandlerParameter;
    hook->observer = observer;
    hook->context = context;
    MmsServer_installReadAccessHandler(mms, observe, hook);
    return true;
}
