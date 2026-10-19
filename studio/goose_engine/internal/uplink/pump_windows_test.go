package uplink

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"time"
)

func startFake(t *testing.T) (*Uplink, *fakeDevice, *recorder) {
	t.Helper()
	dev, rec := newFakeDevice(t), newRecorder()
	u := start("{GUID}", dev.opener(), rec.frame, rec.state)
	rec.await(t, "open")
	t.Cleanup(u.Close)
	return u, dev, rec
}

func numbered(i int) []byte { return []byte(fmt.Sprintf("frame-%06d", i)) }

func TestEveryFrameHeardIsDeliveredInOrder(t *testing.T) {
	_, dev, rec := startFake(t)
	total := 0
	for _, burst := range []int{1, 7, batch, batch + 1, 3 * batch, 2} {
		frames := make([][]byte, burst)
		for i := range frames {
			frames[i] = numbered(total + i)
		}
		dev.arrive(frames...)
		total += burst
		eventually(t, "frames heard", func() bool { return rec.frameCount() == total })
	}
	for i, got := range rec.frames {
		if !bytes.Equal(got, numbered(i)) {
			t.Fatalf("frame %d is %q", i, got)
		}
	}
}

func TestQueuedFramesAreSentUnchangedAndInOrder(t *testing.T) {
	u, dev, _ := startFake(t)
	u.AllowSend(true)
	const count = 3000
	frames := make([][]byte, count)
	for i := range frames {
		frames[i] = numbered(i)
		for !u.Send(frames[i]) { // the queue holds 1024; a full queue refuses
			time.Sleep(50 * time.Microsecond)
		}
	}
	eventually(t, "frames sent", func() bool { return dev.sentCount() == count })
	for i, got := range dev.sent {
		if &got[0] != &frames[i][0] || !bytes.Equal(got, frames[i]) {
			t.Fatalf("frame %d was changed or reordered: %q", i, got)
		}
	}
}

func TestNothingIsSentUntilAllowed(t *testing.T) {
	u, dev, _ := startFake(t)
	if u.Send(numbered(1)) {
		t.Fatal("Send accepted a frame before sending was allowed")
	}
	u.AllowSend(true)
	if !u.Send(numbered(2)) {
		t.Fatal("Send refused an allowed frame")
	}
	eventually(t, "frame sent", func() bool { return dev.sentCount() == 1 })
	u.AllowSend(false)
	if u.Send(numbered(3)) {
		t.Fatal("Send accepted a frame after sending was switched off")
	}
}

func TestIdleAdapterNeverWakesTheThread(t *testing.T) {
	u, dev, rec := startFake(t)
	dev.arrive(numbered(0))
	eventually(t, "frame heard", func() bool { return rec.frameCount() == 1 })
	time.Sleep(20 * time.Millisecond)
	before := u.wakes.Load()
	time.Sleep(time.Second)
	if woke := u.wakes.Load() - before; woke != 0 {
		t.Fatalf("idle adapter thread woke %d times in 1 s", woke)
	}
}

func TestCloseWakesAnIdleThreadAtOnce(t *testing.T) {
	dev, rec := newFakeDevice(t), newRecorder()
	u := start("{GUID}", dev.opener(), rec.frame, rec.state)
	rec.await(t, "open")
	time.Sleep(20 * time.Millisecond)
	began := time.Now()
	u.Close()
	if took := time.Since(began); took > 200*time.Millisecond {
		t.Fatalf("Close took %v", took)
	}
	if _, open := u.OpenSince(); open {
		t.Fatal("adapter still reported open after Close")
	}
	if u.Send(numbered(1)) {
		t.Fatal("Send accepted a frame after Close")
	}
}

func TestLostAdapterIsReported(t *testing.T) {
	_, dev, rec := startFake(t)
	dev.fail(errors.New("cable pulled"))
	rec.await(t, "lost")
}

func TestSendFailureIsReportedOnceThenRecovers(t *testing.T) {
	u, dev, rec := startFake(t)
	u.AllowSend(true)
	dev.mu.Lock()
	dev.sendErr = errors.New("adapter busy")
	dev.mu.Unlock()
	u.Send(numbered(1))
	u.Send(numbered(2))
	rec.await(t, "send_failed")
	dev.mu.Lock()
	dev.sendErr = nil
	dev.mu.Unlock()
	u.Send(numbered(3))
	rec.await(t, "open")
	rec.mu.Lock()
	defer rec.mu.Unlock()
	failed := 0
	for _, state := range rec.states {
		if state == "send_failed" {
			failed++
		}
	}
	if failed != 1 {
		t.Fatalf("send_failed reported %d times: %v", failed, rec.states)
	}
}

// TestSendIsHandedOverWithoutWaitingForATimer guards against a return to polling:
// a polled loop hands frames over 0.5-8 ms late on average.
func TestSendIsHandedOverWithoutWaitingForATimer(t *testing.T) {
	u, dev, _ := startFake(t)
	stats := measureSend(u, dev, 200, 2*time.Millisecond)
	t.Logf("publish -> hand-off: %s", stats)
	if stats.p50 > 300*time.Microsecond {
		t.Fatalf("median hand-off %v; the thread is not woken by Send", stats.p50)
	}
}
