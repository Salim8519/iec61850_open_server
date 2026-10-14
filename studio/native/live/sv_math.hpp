#pragma once
// Per-channel RMS and fundamental phasor over whole cycles of Sampled Values.
// Samples are placed on the cycle by smpCnt, so the phasor angle is stable between
// updates and comparable between channels of the same stream.
#include <cmath>
#include <vector>

struct ChannelCycle {
    double sum_squares = 0, re = 0, im = 0;
    int count = 0;
    double rms = 0, magnitude = 0, angle = 0;
    bool ready = false;

    // A cycle starts at index 0 and counts only when all its samples arrived; a cycle
    // with a lost sample is dropped instead of being mixed into the next one.
    void add(double value, int index_in_cycle, int per_cycle) {
        const double pi = 3.14159265358979323846;
        if (index_in_cycle == 0) reset();
        double phase = 2.0 * pi * index_in_cycle / per_cycle;
        sum_squares += value * value;
        re += value * std::cos(phase);
        im -= value * std::sin(phase);
        ++count;
        if (index_in_cycle != per_cycle - 1) return;
        if (count == per_cycle) {
            rms = std::sqrt(sum_squares / count);
            magnitude = std::sqrt(re * re + im * im) * std::sqrt(2.0) / count;
            angle = std::atan2(im, re) * 180.0 / pi;
            ready = true;
        }
        reset();
    }

    void reset() { sum_squares = re = im = 0; count = 0; }
};

// Counts lost samples from smpCnt, which wraps at `wrap` (samples per second for
// smpMod 0 at the nominal frequency, or smpRate for smpMod 1). A jump of more than
// half the wrap is a publisher restart, not loss.
struct SampleCounter {
    int wrap = 0;
    long long last = -1;
    long long lost = 0;
    long long restarts = 0;

    void next(int smp_cnt) {
        if (last >= 0 && wrap > 0) {
            long long gap = ((smp_cnt - last - 1) % wrap + wrap) % wrap;
            if (gap > wrap / 2) restarts++;
            else lost += gap;
        }
        last = smp_cnt;
    }
};
