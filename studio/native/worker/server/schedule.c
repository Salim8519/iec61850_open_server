/* Reads the scheduling state libiec61850 itself keeps, so the event loop wakes exactly when
 * IedServer_performPeriodicTasks would act (workstream G). The conditions mirror
 * Control_processControlActions, processEventsForReport (reporting.c), Reporting_sendReports
 * and MmsMapping_checkForSettingGroupReservationTimeouts of the pinned commit; the private
 * headers are hash-checked by cmake/event_loop_support.cmake. Runs on the loop thread, the
 * only thread that runs the periodic tasks and connection ticks. */
#include "schedule.h"
#include "hal_thread.h"
#include "libiec61850_platform_includes.h"
#include "linked_list.h"
#include "mms_mapping.h"
#include "ied_server_private.h"
#include "mms_mapping_internal.h"
#include "mms_server_internal.h"
#include "reporting.h"

static uint64_t earliest(uint64_t a, uint64_t b) { return a < b ? a : b; }

static uint64_t
report_deadline(ReportControl* rc, uint64_t now)
{
    uint64_t next = STUDIO_NO_DEADLINE;
    /* triggered/reportTime/gi change under this mutex when a data model update arrives. */
    Semaphore_wait(rc->createNotificationsMutex);
    if (rc->enabled || rc->isBuffering) {
        if ((rc->triggerOps & TRG_OPT_GI) && rc->gi)
            next = now;
        if ((rc->triggerOps & TRG_OPT_INTEGRITY) && rc->intgPd > 0)
            next = earliest(next, rc->nextIntgReportTime);
        if (rc->triggered)
            next = earliest(next, rc->reportTime);
    }
    Semaphore_post(rc->createNotificationsMutex);
    /* sendNextReportEntry stops after 100 messages or when the socket is busy; the vendor
     * loop retries on every tick, so keep ticking every millisecond until the queue drains. */
    if (rc->enabled && rc->reportBuffer && rc->reportBuffer->nextToTransmit)
        next = earliest(next, now + 1);
    return next;
}

uint64_t
studio_server_deadline(IedServer server, uint64_t now)
{
    MmsMapping* mapping = server->mmsMapping;
    uint64_t next = mapping->nextControlTimeout;
    LinkedList element = LinkedList_getNext(mapping->reportControls);
    while (element) {
        next = earliest(next, report_deadline((ReportControl*) LinkedList_getData(element), now));
        element = LinkedList_getNext(element);
    }
    /* An open setting-group edit session expires on a private timer: poll like the vendor. */
    for (SettingGroupControlBlock* sgcb = server->model->sgcbs; sgcb; sgcb = sgcb->sibling) {
        if (sgcb->editSG != 0)
            next = earliest(next, now + 1);
    }
    return next;
}

int
studio_goose_intervals(IedServer server, uint32_t* intervals, int capacity)
{
    int count = 0;
    LinkedList element = LinkedList_getNext(server->mmsMapping->gseControls);
    while (element) {
        MmsGooseControlBlock gocb = (MmsGooseControlBlock) LinkedList_getData(element);
        if (MmsGooseControlBlock_getGoEna(gocb)) {
            uint32_t values[2] = {MmsGooseControlBlock_getMinTime(gocb), MmsGooseControlBlock_getMaxTime(gocb)};
            for (int v = 0; v < 2; v++) {
                bool known = values[v] == 0;
                for (int i = 0; i < count && !known; i++)
                    known = intervals[i] == values[v];
                if (!known && count < capacity)
                    intervals[count++] = values[v];
            }
        }
        element = LinkedList_getNext(element);
    }
    return count;
}

void
studio_server_stop(IedServer server)
{
    if (!server->running)
        return;
    Reporting_deactivateAllReports(server->mmsMapping);
    IedServer_stopThreadless(server);
}
