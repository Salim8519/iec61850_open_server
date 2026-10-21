package events

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
)

func python(t *testing.T) string {
	path := os.Getenv("STUDIO_PYTHON")
	if path == "" {
		path = filepath.Join("..", "..", "..", ".venv", "Scripts", "python.exe")
	}
	if _, err := os.Stat(path); err != nil {
		t.Skipf("no CPython at %s; set STUDIO_PYTHON", path)
	}
	return path
}

// TestBusAgainstPython replays random publish/after operations on the Python EventBus and
// on Bus (capacity 7, so eviction and resync happen often) and compares every byte.
func TestBusAgainstPython(t *testing.T) {
	backend, _ := filepath.Abs(filepath.Join("..", "..", "..", "backend"))
	command := exec.Command(python(t), "-I", filepath.Join("testdata", "bus_vectors.py"),
		"--backend", backend, "--seed", "61850", "--count", "4000", "--capacity", "7")
	command.Stderr = os.Stderr
	output, err := command.StdoutPipe()
	if err != nil || command.Start() != nil {
		t.Fatalf("start python: %v", err)
	}
	defer func() { _ = command.Wait() }()
	bus := NewBus(7)
	clock := 1760900000.0
	bus.SetClock(func() float64 { clock += 0.1234567; return clock })
	lines := bufio.NewScanner(output)
	lines.Buffer(make([]byte, 1<<20), 1<<24)
	operations, failures := 0, 0
	for lines.Scan() && failures < 10 {
		decoded, err := pyjson.Decode(lines.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		line := decoded.(*pyjson.Object)
		op, _ := line.Get("op")
		operations++
		if op == "publish" {
			kind, _ := line.Get("kind")
			payload, _ := line.Get("payload")
			want, _ := line.Get("data")
			event, err := bus.Publish(kind.(string), payload.(*pyjson.Object))
			if err != nil || string(event.Data) != want {
				t.Errorf("publish %d:\n got %s (%v)\nwant %s", operations, event.Data, err, want)
				failures++
			}
			continue
		}
		cursor, _ := line.Get("cursor")
		result, _ := line.Get("result")
		got := bus.After(cursor.(int64), 0, nil)
		want := result.([]any)
		same := len(got) == len(want)
		for index := 0; same && index < len(got); index++ {
			same = string(got[index].Data) == want[index].(string)
		}
		if !same {
			t.Errorf("after(%d) at op %d: got %d events, want %v", cursor, operations, len(got), want)
			failures++
		}
	}
	if operations != 4000 && failures == 0 {
		t.Fatalf("Python wrote %d of 4000 operations", operations)
	}
	t.Logf("%d bus operations compared with Python, %d differences", operations, failures)
}

func TestAfterWaitsOnceForPublish(t *testing.T) {
	bus := NewBus(Capacity)
	started := time.Now()
	if items := bus.After(0, 30*time.Millisecond, nil); items != nil || time.Since(started) < 25*time.Millisecond {
		t.Fatalf("empty bus: %v after %v", items, time.Since(started))
	}
	go func() {
		time.Sleep(20 * time.Millisecond)
		_, _ = bus.Publish("runtime", pyjson.NewObject().Set("state", "running"))
	}()
	items := bus.After(0, 5*time.Second, nil)
	if len(items) != 1 || items[0].ID != 1 || items[0].Kind != "runtime" {
		t.Fatalf("woken read: %+v", items)
	}
	done := make(chan struct{})
	close(done)
	if items := bus.After(5, time.Hour, done); items != nil {
		t.Fatalf("future cursor: %+v", items)
	}
}

// BenchmarkPublish measures one signal event: encode once, store in the ring, wake readers.
func BenchmarkPublish(b *testing.B) {
	bus := NewBus(Capacity)
	payload := pyjson.NewObject().Set("ied_id", "ied-1").Set("ref", "LD0/MMXU1.TotW.mag.f").
		Set("value", 1234.5).Set("source", "automation").Set("quality", "good").
		Set("source_timestamp", 1760900000.25).Set("updates", pyjson.NewObject())
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = bus.Publish("signal", payload)
	}
}
