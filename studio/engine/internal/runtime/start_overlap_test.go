package runtime

import (
	"errors"
	"sync"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/testutil"
)

// countingLive is NoLive that counts BusFor calls and can refuse them.
type countingLive struct {
	NoLive
	mu      sync.Mutex
	buses   int
	stops   int
	refuse  error
	started int
}

func (l *countingLive) BusFor([]*pyjson.Object) (map[string]string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buses++
	return map[string]string{}, l.refuse
}

func (l *countingLive) Start(*pyjson.Object, []string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.started++
	return nil
}

func (l *countingLive) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stops++
}

func (l *countingLive) counts() (int, int, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buses, l.stops, l.started
}

func labProject(t *testing.T, net int, portBase int64) (*pyjson.Object, string) {
	project, a, _ := fakeProject(t, net, portBase, false)
	project.Set("network", pyjson.NewObject().Set("mode", "lab").Set("adapter_guid", "{LAB}"))
	return project, a
}

// A fresh Run opens the GOOSE bus and starts the listeners while the lab addresses are
// prepared; a refused lab step still wins, the run stops (and with it the GOOSE engine and the
// listeners) and no IED is left running.
func TestRefusedLabStepWinsOverTheEarlyBus(t *testing.T) {
	project, a := labProject(t, 5, 22500)
	live := &countingLive{}
	refused := errors.New("the lab card is missing")
	c := New(Options{NativeDir: fakeNativeDir(t), DataDir: testutil.TempDir(t), Bus: &MemoryBus{},
		Project: NewMemoryProject(project), Live: live,
		LabAddresses: func(string, []string) ([]string, error) { return nil, refused }})
	defer c.Close()
	if err := c.Start(nil); !errors.Is(err, refused) {
		t.Fatalf("Start error = %v, want the lab error", err)
	}
	buses, stops, started := live.counts()
	if buses != 1 || stops != 1 || started != 1 {
		t.Fatalf("BusFor %d, Stop %d, Start %d; want 1, 1, 1", buses, stops, started)
	}
	if c.State() != "stopped" || iedStateOf(c, a) != "stopped" {
		t.Fatalf("state %q, IED %q after a refused Run", c.State(), iedStateOf(c, a))
	}
}

func TestRefusedBusFailsTheRun(t *testing.T) {
	project, _ := labProject(t, 6, 22600)
	live := &countingLive{refuse: errors.New("the GOOSE engine is outdated")}
	c := New(Options{NativeDir: fakeNativeDir(t), DataDir: testutil.TempDir(t), Bus: &MemoryBus{},
		Project: NewMemoryProject(project), Live: live})
	defer c.Close()
	if err := c.Start(nil); err == nil || err.Error() != "the GOOSE engine is outdated" {
		t.Fatalf("Start error = %v, want the GOOSE engine error", err)
	}
	if c.State() != "stopped" {
		t.Fatalf("state %q after a refused bus", c.State())
	}
}

// An IED added to a running simulation registers with the GOOSE engine only after its checks.
func TestAddedIedOpensNoBusWhenRefused(t *testing.T) {
	project, a := labProject(t, 7, 22700)
	ieds := projectIeds(project)
	ieds[1].Set("role", "simulated")
	b := pyval.Str(ieds[1], "id")
	live := &countingLive{}
	calls := 0
	refused := errors.New("the lab card is missing")
	c := New(Options{NativeDir: fakeNativeDir(t), DataDir: testutil.TempDir(t), Bus: &MemoryBus{},
		Project: NewMemoryProject(project), Live: live,
		LabAddresses: func(string, []string) ([]string, error) {
			calls++
			if calls > 1 {
				return nil, refused
			}
			return nil, nil
		}})
	defer c.Close()
	if err := c.Start([]string{a}); err != nil {
		t.Fatal(err)
	}
	defer c.Stop(nil)
	before, _, _ := live.counts()
	if err := c.Start([]string{b}); !errors.Is(err, refused) {
		t.Fatalf("adding an IED: %v, want the lab error", err)
	}
	if after, _, _ := live.counts(); after != before {
		t.Fatalf("BusFor ran %d times for a refused addition", after-before)
	}
	if iedStateOf(c, a) != "running" || iedStateOf(c, b) == "running" {
		t.Fatalf("after the refused addition: A %q, B %q", iedStateOf(c, a), iedStateOf(c, b))
	}
}

func iedStateOf(c *Coordinator, id string) string {
	state, _ := c.IedState(id)
	return state
}
