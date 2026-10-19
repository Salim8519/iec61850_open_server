package updates

import (
	"context"
	"math/rand/v2"
	"os"
	"time"
)

// Schedule holds the timing seams of the background loop; tests replace Sleep and
// Uniform so nothing really waits.
type Schedule struct {
	// Sleep waits seconds or until ctx ends; false means stop.
	Sleep func(ctx context.Context, seconds float64) bool
	// Uniform is random.uniform(low, high).
	Uniform func(low, high float64) float64
}

// RealSchedule sleeps on the wall clock with random jitter.
func RealSchedule() Schedule {
	return Schedule{
		Sleep: func(ctx context.Context, seconds float64) bool {
			timer := time.NewTimer(time.Duration(seconds * float64(time.Second)))
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return false
			case <-timer.C:
				return true
			}
		},
		Uniform: func(low, high float64) float64 { return low + rand.Float64()*(high-low) },
	}
}

// Disabled reports SCADABENCH_UPDATE_CHECK=off (the test suites set it so they never
// reach the internet).
func Disabled() bool { return os.Getenv("SCADABENCH_UPDATE_CHECK") == "off" }

// Run is start_background's loop: wait 20-120 s after start, then every hour plus up to
// 10 minutes of jitter check when Due. Failures are logged, never shown. It returns when
// ctx ends.
func (s *Service) Run(ctx context.Context, schedule Schedule) {
	if !schedule.Sleep(ctx, schedule.Uniform(StartDelayMin, StartDelayMax)) {
		return
	}
	for {
		due, err := s.Due()
		if err == nil && due {
			_, err = s.Check(false)
		}
		if err != nil {
			s.options.Logf("Update check failed: %v", err)
		}
		if !schedule.Sleep(ctx, 3600+schedule.Uniform(0, 600)) {
			return
		}
	}
}

// StartBackground runs the loop on its own goroutine unless the check is disabled.
func (s *Service) StartBackground(ctx context.Context) {
	if Disabled() {
		return
	}
	go s.Run(ctx, RealSchedule())
}
