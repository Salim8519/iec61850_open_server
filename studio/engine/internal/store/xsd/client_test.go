package xsd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"studio-engine/internal/apperr"
)

func TestValidateThroughHelperThenCache(t *testing.T) {
	f := newFixture(t, 20*time.Second)
	source := []byte("<SCL>broken</SCL>")
	verdict, err := f.validator.Validate(source, "SCL2007B4")
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("fake SCL2007B4 %d bytes", len(source))
	if verdict.Status != "invalid" || len(verdict.Errors) != 1 || verdict.Errors[0].Line != 2 ||
		verdict.Errors[0].Message != want {
		t.Fatalf("unexpected verdict %+v", verdict)
	}
	if valid, err := f.validator.Validate([]byte("<SCL>VALID</SCL>"), "SCL2003"); err != nil ||
		valid.Status != "valid" || valid.Errors == nil || len(valid.Errors) != 0 {
		t.Fatalf("valid verdict: %+v %v", valid, err)
	}
	again, err := f.validator.Validate(source, "SCL2007B4")
	if err != nil || again.Errors[0].Message != want {
		t.Fatalf("cached verdict: %+v %v", again, err)
	}
	if calls := f.helperCalls(t); calls != 2 {
		t.Fatalf("the helper must see each document once, saw %d validations", calls)
	}
	if starts := f.validator.HelperStarts(); starts != 1 {
		t.Fatalf("one helper process expected, started %d", starts)
	}
	work, _ := os.ReadDir(filepath.Join(f.validator.config.DataDir, "cache", "xsd", "work"))
	if len(work) != 0 {
		t.Fatalf("work files must be removed after each answer, found %d", len(work))
	}
}

func TestDoctypeIsValueErrorAndNotCached(t *testing.T) {
	f := newFixture(t, 20*time.Second)
	for attempt := 0; attempt < 2; attempt++ {
		_, err := f.validator.Validate([]byte("<!DOCTYPE SCL><SCL/>"), "SCL2007B")
		var appErr *apperr.Error
		if !errors.As(err, &appErr) || appErr.Kind != apperr.Value ||
			appErr.Message != "DTD declarations are not allowed in SCL" {
			t.Fatalf("want Python's ValueError, got %#v", err)
		}
	}
	if calls := f.helperCalls(t); calls != 2 {
		t.Fatalf("DOCTYPE outcomes are not cached, want 2 helper calls, got %d", calls)
	}
}

func TestMissingHelperOnCacheMiss(t *testing.T) {
	f := newFixtureWithHelper(t, filepath.Join(t.TempDir(), "studio_xsd.exe"), time.Second)
	_, err := f.validator.Validate([]byte("<SCL/>"), "SCL2007B")
	requireRuntime(t, err, "SCL schema validation is not available: the studio_xsd helper is missing. Reinstall ScadaBench.")
}

func TestUnknownSchemaName(t *testing.T) {
	f := newFixture(t, time.Second)
	_, err := f.validator.Validate([]byte("<SCL/>"), "SCL2099")
	var appErr *apperr.Error
	if !errors.As(err, &appErr) || appErr.Kind != apperr.Value {
		t.Fatalf("want ValueError, got %#v", err)
	}
}

func TestHelloRefusesOtherLibxml2(t *testing.T) {
	f := newFixture(t, 5*time.Second)
	t.Setenv("STUDIO_XSD_FAKE_LIBXML2", "2.12.0")
	_, err := f.validator.Validate([]byte("<SCL/>"), "SCL2007B")
	requireRuntime(t, err, `uses libxml2 "2.12.0", but ScadaBench needs libxml2 2.11.9`)
	if calls := f.helperCalls(t); calls != 0 {
		t.Fatal("a refused helper must not receive documents")
	}
}

func TestHelloRefusesOtherBuild(t *testing.T) {
	f := newFixture(t, 5*time.Second)
	f.validator.client.expectedBuild = "0.2.0"
	t.Setenv("STUDIO_XSD_FAKE_BUILD", "0.1.14")
	_, err := f.validator.Validate([]byte("<SCL/>"), "SCL2007B")
	requireRuntime(t, err, `another ScadaBench build ("0.1.14", expected "0.2.0")`)
}

func TestCrashThenRestart(t *testing.T) {
	f := newFixture(t, 20*time.Second)
	_, err := f.validator.Validate([]byte("<SCL>CRASH</SCL>"), "SCL2007B")
	requireRuntime(t, err, "The studio_xsd helper stopped during SCL schema validation")
	verdict, err := f.validator.Validate([]byte("<SCL>VALID after crash</SCL>"), "SCL2007B")
	if err != nil || verdict.Status != "valid" {
		t.Fatalf("restart after crash: %+v %v", verdict, err)
	}
	if starts := f.validator.HelperStarts(); starts != 2 {
		t.Fatalf("want a second helper after the crash, started %d", starts)
	}
}

func TestTimeoutKillsAndRestarts(t *testing.T) {
	f := newFixture(t, 1500*time.Millisecond)
	began := time.Now()
	_, err := f.validator.Validate([]byte("<SCL>SLEEP</SCL>"), "SCL2007B")
	requireRuntime(t, err, "took longer than 1 seconds and was stopped")
	if elapsed := time.Since(began); elapsed > 10*time.Second {
		t.Fatalf("timeout took %v", elapsed)
	}
	verdict, err := f.validator.Validate([]byte("<SCL>VALID</SCL>"), "SCL2007B")
	if err != nil || verdict.Status != "valid" || f.validator.HelperStarts() != 2 {
		t.Fatalf("after timeout: %+v %v starts=%d", verdict, err, f.validator.HelperStarts())
	}
}

func TestConcurrentValidations(t *testing.T) {
	f := newFixture(t, 30*time.Second)
	const workers, documents = 6, 4
	var group sync.WaitGroup
	failures := make(chan error, workers*documents)
	for worker := 0; worker < workers; worker++ {
		group.Add(1)
		go func(worker int) {
			defer group.Done()
			for document := 0; document < documents; document++ {
				source := []byte(fmt.Sprintf("<SCL>doc %d %s</SCL>", document, "padding"[:worker+1]))
				verdict, err := f.validator.Validate(source, "SCL2007B4")
				want := fmt.Sprintf("fake SCL2007B4 %d bytes", len(source))
				if err != nil || verdict.Errors[0].Message != want {
					failures <- fmt.Errorf("worker %d document %d: %+v %v", worker, document, verdict, err)
				}
			}
		}(worker)
	}
	group.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	if starts := f.validator.HelperStarts(); starts != 1 {
		t.Fatalf("concurrent callers must share one helper, started %d", starts)
	}
}

func TestCloseStopsHelper(t *testing.T) {
	f := newFixture(t, 10*time.Second)
	if _, err := f.validator.Validate([]byte("<SCL>VALID</SCL>"), "SCL2007B"); err != nil {
		t.Fatal(err)
	}
	proc := f.validator.client.proc
	f.validator.Close()
	select {
	case <-proc.done:
	case <-time.After(5 * time.Second):
		t.Fatal("the helper did not exit after Close")
	}
	_, err := f.validator.Validate([]byte("<SCL>new</SCL>"), "SCL2007B")
	requireRuntime(t, err, "ScadaBench is closing")
	if cached, err := f.validator.Validate([]byte("<SCL>VALID</SCL>"), "SCL2007B"); err != nil || cached.Status != "valid" {
		t.Fatalf("cached verdicts stay available after Close: %+v %v", cached, err)
	}
}
