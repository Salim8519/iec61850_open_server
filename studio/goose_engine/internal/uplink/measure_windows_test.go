package uplink

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"syscall"
	"testing"
	"time"
)

var (
	winmm               = syscall.NewLazyDLL("winmm.dll")
	procTimeBeginPeriod = winmm.NewProc("timeBeginPeriod")
	procTimeEndPeriod   = winmm.NewProc("timeEndPeriod")
)

// millisecondTimer is what studio_goose did before this change (timer_windows.go).
func millisecondTimer() func() {
	procTimeBeginPeriod.Call(1)
	return func() { procTimeEndPeriod.Call(1) }
}

type latency struct {
	n             int
	p50, p99, max time.Duration
}

func (l latency) String() string {
	return fmt.Sprintf("n=%d p50=%v p99=%v max=%v", l.n, l.p50, l.p99, l.max)
}

// measureSend publishes n frames gap apart and reports publish -> hand-off to the
// device (the moment pcap_sendpacket would be called).
func measureSend(u *Uplink, dev *fakeDevice, n int, gap time.Duration) latency {
	u.AllowSend(true)
	start := dev.sentCount()
	published := make([]int64, n)
	for i := 0; i < n; i++ {
		published[i] = qpc()
		for !u.Send(numbered(i)) {
			published[i] = qpc()
		}
		time.Sleep(gap)
	}
	for deadline := time.Now().Add(5 * time.Second); dev.sentCount() < start+n && time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
	}
	dev.mu.Lock()
	defer dev.mu.Unlock()
	hz := float64(qpcFrequency())
	samples := make([]time.Duration, 0, n)
	for i := 0; i < n && start+i < len(dev.sentAt); i++ {
		samples = append(samples, time.Duration(float64(dev.sentAt[start+i]-published[i])/hz*1e9))
	}
	sort.Slice(samples, func(a, b int) bool { return samples[a] < samples[b] })
	if len(samples) == 0 {
		return latency{}
	}
	return latency{n: len(samples), p50: samples[len(samples)/2], p99: samples[len(samples)*99/100],
		max: samples[len(samples)-1]}
}

// legacyPump is the loop before this change, kept only to measure against: each pass
// waits up to 1 ms for a frame, as pcap_next_ex did (Packet.dll's
// WaitForSingleObject(ReadEvent, ReadTimeOut)), then drains the send queue.
func (u *Uplink) legacyPump(dev device) {
	for {
		select {
		case <-u.stop:
			return
		default:
		}
		syscall.WaitForSingleObject(syscall.Handle(dev.Event()), 1)
		u.wakes.Add(1)
		if frame, _ := dev.Next(); frame != nil {
			u.onFrame(frame)
		}
		u.flush(dev)
	}
}

func startLegacy(t *testing.T, dev *fakeDevice) *Uplink {
	t.Helper()
	sig, err := newSignals()
	if err != nil {
		t.Fatal(err)
	}
	u := &Uplink{guid: "{GUID}", sig: sig, out: make(chan []byte, 1024), onFrame: func([]byte) {},
		onState: func(string, string) {}, stop: make(chan struct{}), done: make(chan struct{})}
	go func() {
		runtime.LockOSThread()
		defer close(u.done)
		u.legacyPump(dev)
	}()
	return u
}

func measureIdle(u *Uplink, seconds int) float64 {
	time.Sleep(50 * time.Millisecond)
	before := u.wakes.Load()
	time.Sleep(time.Duration(seconds) * time.Second)
	return float64(u.wakes.Load()-before) / float64(seconds)
}

func idleSeconds() int {
	var seconds int
	if _, err := fmt.Sscan(os.Getenv("STUDIO_GOOSE_IDLE_SECONDS"), &seconds); err != nil || seconds <= 0 {
		return 10
	}
	return seconds
}

// TestMeasureBeforeAfter prints the numbers recorded in the speed-n2 stream log. It
// runs only with STUDIO_GOOSE_MEASURE=1 (about a minute); no adapter is used.
func TestMeasureBeforeAfter(t *testing.T) {
	if os.Getenv("STUDIO_GOOSE_MEASURE") == "" {
		t.Skip("set STUDIO_GOOSE_MEASURE=1 to measure")
	}
	seconds := idleSeconds()
	modes := []struct {
		name   string
		legacy bool
		timer  bool
	}{
		{"before: 1 ms poll + timeBeginPeriod(1)", true, true},
		{"before: 1 ms poll, no timer request", true, false},
		{"after: event wait, no timer request", false, false},
	}
	for _, mode := range modes {
		restore := func() {}
		if mode.timer {
			restore = millisecondTimer()
		}
		dev := newFakeDevice(t)
		var u *Uplink
		if mode.legacy {
			u = startLegacy(t, dev)
		} else {
			rec := newRecorder()
			u = start("{GUID}", dev.opener(), rec.frame, rec.state)
			rec.await(t, "open")
		}
		wakes := measureIdle(u, seconds)
		send := measureSend(u, dev, 2000, 2*time.Millisecond)
		u.Close()
		restore()
		t.Logf("%-40s idle wakes/s=%.1f (over %d s)  publish->hand-off %s", mode.name, wakes, seconds, send)
	}
}

// TestMeasureGoTimers checks the engine's own timers do not need the 1 ms system
// timer: Go 1.23+ sleeps on high-resolution waitable timers.
func TestMeasureGoTimers(t *testing.T) {
	if os.Getenv("STUDIO_GOOSE_MEASURE") == "" {
		t.Skip("set STUDIO_GOOSE_MEASURE=1 to measure")
	}
	for _, timer := range []bool{false, true} {
		restore := func() {}
		if timer {
			restore = millisecondTimer()
		}
		hz := float64(qpcFrequency())
		samples := make([]time.Duration, 0, 500)
		for i := 0; i < 500; i++ {
			began := qpc()
			time.Sleep(time.Millisecond)
			samples = append(samples, time.Duration(float64(qpc()-began)/hz*1e9))
		}
		restore()
		sort.Slice(samples, func(a, b int) bool { return samples[a] < samples[b] })
		t.Logf("timeBeginPeriod(1)=%v: time.Sleep(1ms) p50=%v p99=%v max=%v", timer,
			samples[250], samples[495], samples[499])
	}
}
