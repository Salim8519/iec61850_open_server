#include "timer_policy.hpp"
#ifdef _WIN32
extern "C" {
#include "studio_hal.h"
}
#include <iostream>
#include <windows.h>
#include <timeapi.h>

#ifndef PROCESS_POWER_THROTTLING_CURRENT_VERSION
#define PROCESS_POWER_THROTTLING_CURRENT_VERSION 1
#endif
#ifndef PROCESS_POWER_THROTTLING_EXECUTION_SPEED
#define PROCESS_POWER_THROTTLING_EXECUTION_SPEED 0x1
#endif
#ifndef PROCESS_POWER_THROTTLING_IGNORE_TIMER_RESOLUTION
#define PROCESS_POWER_THROTTLING_IGNORE_TIMER_RESOLUTION 0x4
#endif

namespace {
// ControlMask = policies this process decides itself; StateMask 0 = never throttle them.
bool set_power_throttling(ULONG control) {
    PROCESS_POWER_THROTTLING_STATE state{};
    state.Version = PROCESS_POWER_THROTTLING_CURRENT_VERSION;
    state.ControlMask = control;
    state.StateMask = 0;
    return SetProcessInformation(GetCurrentProcess(), ProcessPowerThrottling, &state, sizeof(state)) != 0;
}

void opt_out_of_power_throttling() {
    // Windows 10 rejects the timer-resolution flag (Windows 11 only); keep execution speed.
    if (!set_power_throttling(PROCESS_POWER_THROTTLING_EXECUTION_SPEED |
                              PROCESS_POWER_THROTTLING_IGNORE_TIMER_RESOLUTION))
        set_power_throttling(PROCESS_POWER_THROTTLING_EXECUTION_SPEED);
}

bool high_resolution_waits() {
    StudioWake probe = StudioWake_create();
    bool available = StudioWake_isHighResolution(probe);
    StudioWake_destroy(probe);
    return available;
}
}

TimerPolicy::TimerPolicy() {
    opt_out_of_power_throttling();
    if (high_resolution_waits()) return;
    // libiec61850 retransmits GOOSE every MinTime; with millisecond waits Windows' default
    // 15.6 ms timer stretched a 4 ms MinTime past the 12 ms TAL (change 0102, ADR-0070).
    raised_resolution = timeBeginPeriod(1) == TIMERR_NOERROR;
    std::cerr << "High-resolution timers are not available on this Windows version; "
                 "using the 1 ms system timer for GOOSE and control timing" << std::endl;
}

TimerPolicy::~TimerPolicy() {
    if (raised_resolution) timeEndPeriod(1);
}
#else
TimerPolicy::TimerPolicy() = default;
TimerPolicy::~TimerPolicy() = default;
#endif
