package native

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

func TestFrameRoundTripKeepsNumberKinds(t *testing.T) {
	message := pyjson.NewObject().Set("version", int64(1)).Set("f", 1.0).Set("b", true)
	frame, err := EncodeFrame(message)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(frame[4:]); got != `{"version":1,"f":1.0,"b":true}` {
		t.Fatalf("payload %s", got)
	}
	decoded, err := ReadFrame(bytes.NewReader(frame))
	if err != nil {
		t.Fatal(err)
	}
	f, _ := decoded.(*pyjson.Object).Get("f")
	if _, ok := f.(float64); !ok {
		t.Fatalf("float became %T", f)
	}
}

func TestFrameErrorsUsePythonTexts(t *testing.T) {
	if _, err := ReadFrame(bytes.NewReader([]byte{0, 0, 0, 0})); err == nil ||
		err.Error() != "Invalid native frame length 0" {
		t.Fatalf("zero length: %v", err)
	}
	if _, err := ReadFrame(bytes.NewReader([]byte{5, 0, 0, 0, '{'})); err != ErrPipeClosed {
		t.Fatalf("short frame: %v", err)
	}
	if _, err := EncodeFrame(pyjson.NewObject().Set("v", nanValue())); err == nil ||
		err.Error() != "Out of range float values are not JSON compliant: nan" {
		t.Fatalf("nan: %v", err)
	}
}

func nanValue() float64 { zero := 0.0; return zero / zero }

func TestIedWorkerStartUpdateReadAndClose(t *testing.T) {
	events := make(chan *pyjson.Object, 256)
	client, started := startTemplate(t, events, "127.63.0.11", 21011)
	if running, _ := started.Get("running"); running != true {
		t.Fatalf("start reply %v", started)
	}
	ref := "BayACTRL/GGIO1.AnIn1.mag.f"
	value, err := client.Update(ref, 12.5)
	if err != nil || value != 12.5 {
		t.Fatalf("update: %v %v", value, err)
	}
	batch, err := client.UpdateBatch([]any{pyjson.NewObject().Set("ref", "BayACTRL/XCBR1.Pos.stVal").Set("value", int64(2))})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := batch.Get("BayACTRL/XCBR1.Pos.stVal"); got != int64(2) {
		t.Fatalf("batch %v (%T)", got, got)
	}
	versioned, err := client.ReadVersioned("BayACTRL/XCBR1.Pos.stVal")
	if err != nil {
		t.Fatal(err)
	}
	revision, _ := versioned.Get("revision")
	guard := pyjson.NewObject().Set("ref", "BayACTRL/XCBR1.Pos.stVal").Set("revision", revision)
	stale := pyjson.NewObject().Set("ref", "BayACTRL/XCBR1.Pos.stVal").Set("revision", int64(-5))
	updates := []any{pyjson.NewObject().Set("ref", "BayACTRL/XCBR1.Pos.stVal").Set("value", int64(1))}
	if _, err := client.UpdateGuarded(updates, guard); err != nil {
		t.Fatal(err)
	}
	_, err = client.UpdateGuarded(updates, stale)
	if failure, ok := err.(*apperr.Error); !ok || failure.Kind != apperr.Runtime ||
		failure.Message != "Signal changed; stale pulse action cancelled" {
		t.Fatalf("stale guard: %v", err)
	}
	if _, err := client.Read("BayACTRL/NoSuch.stVal"); err == nil {
		t.Fatal("unknown ref accepted")
	}
	client.Close()
	if exited, _ := client.Exited(); !exited {
		t.Fatal("worker still alive after Close")
	}
	_, err = client.Read(ref)
	if err == nil || !strings.HasPrefix(err.Error(), "Native worker exited: ") {
		t.Fatalf("after close: %v", err)
	}
	for len(events) > 0 {
		if kind, _ := (<-events).Get("event"); kind == "worker_failed" {
			t.Fatal("worker_failed reported while closing")
		}
	}
}

func TestKilledWorkerReportsFailureAndFailsPending(t *testing.T) {
	events := make(chan *pyjson.Object, 256)
	client, _ := startTemplate(t, events, "127.63.0.12", 21012)
	_ = client.command.Process.Kill()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case event := <-events:
			if kind, _ := event.Get("event"); kind == "worker_failed" {
				if text, _ := event.Get("error"); text != "Native worker pipe closed" {
					t.Fatalf("failure text %v", text)
				}
				return
			}
		case <-deadline:
			t.Fatal("no worker_failed")
		}
	}
}

func TestRequestTimeoutText(t *testing.T) {
	client, _ := startTemplate(t, nil, "127.63.0.13", 21013)
	_, err := client.Request("read", pyjson.NewObject().Set("ref", "BayACTRL/XCBR1.Pos.stVal"), time.Nanosecond)
	if err != nil && err.Error() != "Native read request timed out" {
		t.Fatalf("timeout text %v", err)
	}
}
