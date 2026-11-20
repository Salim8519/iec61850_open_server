package runtime

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"studio-engine/internal/testutil"
)

func sparePids(c *Coordinator) []int {
	c.spares.mu.Lock()
	defer c.spares.mu.Unlock()
	pids := make([]int, 0, len(c.spares.ready))
	for _, spare := range c.spares.ready {
		pids = append(pids, spare.worker.Pid())
	}
	sort.Ints(pids)
	return pids
}

func waitSpares(t *testing.T, c *Coordinator, want int) []int {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for {
		if pids := sparePids(c); len(pids) == want {
			return pids
		}
		if time.Now().After(deadline) {
			t.Fatalf("%d spare processes, want %d", len(sparePids(c)), want)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func workerPids(c *Coordinator) []int {
	c.mu.Lock()
	defer c.mu.Unlock()
	var pids []int
	for _, item := range c.runs {
		if item.worker != nil {
			pids = append(pids, item.worker.Pid())
		}
	}
	sort.Ints(pids)
	return pids
}

func samePids(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestRunTakesSpareProcesses (trial R3): while stopped the coordinator keeps one greeted
// studio_ied per simulated IED; Run takes them instead of starting processes, Stop readies new
// ones (never the closed ones again) and Close ends them.
func TestRunTakesSpareProcesses(t *testing.T) {
	project, _, _ := fakeProject(t, 31, 22600, false)
	c := New(Options{NativeDir: fakeNativeDir(t), DataDir: testutil.TempDir(t), Bus: &MemoryBus{},
		Project: NewMemoryProject(project)})
	defer c.Close()
	c.PrepareSpares()
	spares := waitSpares(t, c, len(projectIeds(project)))
	if err := c.Start(nil); err != nil {
		t.Fatal(err)
	}
	if running := workerPids(c); !samePids(running, spares) {
		t.Fatalf("Run started %v, the spares were %v", running, spares)
	}
	if left := sparePids(c); len(left) != 0 {
		t.Fatalf("spares left while running: %v", left)
	}
	if err := c.Stop(nil); err != nil {
		t.Fatal(err)
	}
	next := waitSpares(t, c, len(spares))
	for _, pid := range next {
		for _, old := range spares {
			if pid == old {
				t.Fatalf("a closed worker came back as a spare: %d", pid)
			}
		}
	}
	c.Close()
	if left := sparePids(c); len(left) != 0 {
		t.Fatalf("spares left after Close: %v", left)
	}
}

// TestSparesNotUsedAfterTheProgramChanged: a rebuilt or updated studio_ied is started anew.
func TestSparesNotUsedAfterTheProgramChanged(t *testing.T) {
	folder := testutil.TempDir(t)
	source := filepath.Join(fakeNativeDir(t), "studio_ied.exe")
	for _, name := range []string{"studio_ied.exe", "studio_control.exe"} {
		if err := copyFile(source, filepath.Join(folder, name)); err != nil {
			t.Fatal(err)
		}
	}
	project, _, _ := fakeProject(t, 32, 22700, false)
	c := New(Options{NativeDir: folder, DataDir: testutil.TempDir(t), Bus: &MemoryBus{}, Project: NewMemoryProject(project)})
	defer c.Close()
	c.PrepareSpares()
	spares := waitSpares(t, c, len(projectIeds(project)))
	later := time.Now().Add(time.Minute)
	if err := os.Chtimes(filepath.Join(folder, "studio_ied.exe"), later, later); err != nil {
		t.Fatal(err)
	}
	if err := c.Start(nil); err != nil {
		t.Fatal(err)
	}
	for _, pid := range workerPids(c) {
		for _, old := range spares {
			if pid == old {
				t.Fatalf("a spare of the old program was used: %d", pid)
			}
		}
	}
	if err := c.Stop(nil); err != nil {
		t.Fatal(err)
	}
}
