// How a simulated IED keeps GOOSE MinTime, SBO and report timers on time (ADR-0070 TAL rules)
// without raising the machine-wide Windows timer resolution (speed lane N1).
// - The process opts out of power throttling (execution speed, timer resolution), so Windows
//   does not slow its timers when no window of the app is visible.
// - The server loop waits with a high-resolution waitable timer (hal/studio_wake.c). Only when
//   Windows has none (before Windows 10 1803) does the worker raise the timer resolution to
//   1 ms for its lifetime, as it always did before, and say so once on stderr.
#pragma once

class TimerPolicy {
public:
    TimerPolicy();
    ~TimerPolicy();
    TimerPolicy(const TimerPolicy&) = delete;
    TimerPolicy& operator=(const TimerPolicy&) = delete;

private:
    bool raised_resolution = false;
};
