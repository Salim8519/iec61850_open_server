package ports

import (
	"testing"

	"studio-engine/internal/acquisition"
	"studio-engine/internal/api/projectapi"
	"studio-engine/internal/api/toolsapi"
	"studio-engine/internal/gateway/service"
	"studio-engine/internal/native"
	"studio-engine/internal/runtime"
	"studio-engine/internal/testutil"
	"studio-engine/internal/tools/worker"
)

// The consumers' interfaces, checked at compile time.
var (
	_ projectapi.Runtime      = (*runtime.Coordinator)(nil)
	_ toolsapi.Runtime        = (*runtime.Coordinator)(nil)
	_ acquisition.Runtime     = (*runtime.Coordinator)(nil)
	_ acquisition.Emitter     = (*runtime.Coordinator)(nil)
	_ acquisition.Publisher   = (*runtime.Coordinator)(nil)
	_ acquisition.Worker      = (*native.Worker)(nil)
	_ worker.Process          = (*native.Worker)(nil)
	_ acquisition.StartWorker = StartWorker
)

func TestAdaptersAnswerLikeTheCoordinator(t *testing.T) {
	bus := &runtime.MemoryBus{}
	project := runtime.NewMemoryProject(nil)
	c := runtime.New(runtime.Options{NativeDir: testutil.TempDir(t), DataDir: testutil.TempDir(t), Bus: bus, Project: project})
	defer c.Close()
	g := Gateway(c)
	if g.RunState() != "stopped" {
		t.Fatalf("run state %q", g.RunState())
	}
	if _, ok := g.Signal("x", "y"); ok {
		t.Fatal("signal of an IED outside the run")
	}
	if _, _, err := g.Endpoint("x"); err == nil || err.Error() != "IED is not running" {
		t.Fatalf("endpoint error %v", err)
	}
	got := make(chan service.Change, 1)
	cancel := g.Subscribe(func(change service.Change) { got <- change })
	defer cancel()
	a := ForAcquisition(c)
	a.States.SetState("x", "real_live")
	if change := <-got; change != (service.Change{IEDID: "x"}) {
		t.Fatalf("change %+v", change)
	}
	if state, _ := c.IedState("x"); state != "real_live" {
		t.Fatalf("state %q", state)
	}
	if a.Values.Has("x") || a.Samples.Running("x") {
		t.Fatal("IED outside the run reported as indexed or running")
	}
	err := c.Stopped(func() error { return nil })
	if err != nil {
		t.Fatalf("stopped refused: %v", err)
	}
}
