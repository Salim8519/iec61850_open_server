/* The vendor's Windows time HAL, compiled unmodified under Vendor_* names for the two
 * wall-clock functions Studio replaces (ADR-0006 pattern, as socket_studio.c; speed lane N1).
 *
 * The vendor reads GetSystemTimeAsFileTime, which advances only at the system tick: 15.6 ms
 * unless some process raises the timer resolution. libiec61850 schedules GOOSE retransmission
 * (nextPublishTime), SBO/report/integrity timers and every timestamp from Hal_getTimeInMs, so
 * studio_ied used to raise the resolution machine-wide (timeBeginPeriod(1), ADR-0070).
 * GetSystemTimePreciseAsFileTime is the same UTC clock read with sub-microsecond resolution;
 * the conversions below are the vendor's, so values are identical apart from precision. */
#define Hal_getTimeInMs Vendor_Hal_getTimeInMs
#define Hal_getTimeInNs Vendor_Hal_getTimeInNs
#include "time.c"
#undef Hal_getTimeInMs
#undef Hal_getTimeInNs

static uint64_t
preciseFileTime(void)
{
    FILETIME ft;
    GetSystemTimePreciseAsFileTime(&ft);
    return (uint64_t) ft.dwLowDateTime + ((uint64_t) ft.dwHighDateTime << 32);
}

msSinceEpoch
Hal_getTimeInMs(void)
{
    static const uint64_t DIFF_TO_UNIXTIME = 11644473600000ULL;
    return (preciseFileTime() / 10000ULL) - DIFF_TO_UNIXTIME;
}

nsSinceEpoch
Hal_getTimeInNs(void)
{
    static const uint64_t DIFF_TO_UNIXTIME = 11644473600000000000ULL;
    return preciseFileTime() * 100ULL - DIFF_TO_UNIXTIME;
}
