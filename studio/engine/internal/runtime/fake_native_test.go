package runtime

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"studio-engine/internal/native"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// fakeEnv makes this test binary act as a native worker (studio_ied or studio_control by
// its file name), so hub and stress tests run without the native build.
const fakeEnv = "STUDIO_RUNTIME_FAKE_NATIVE"

// fakeNoCancelEnv makes the fake refuse every Cancel as a model without Cancel does.
const fakeNoCancelEnv = "STUDIO_RUNTIME_FAKE_NO_CANCEL"

func TestMain(m *testing.M) {
	if os.Getenv(fakeEnv) == "1" {
		name := strings.ToLower(strings.TrimSuffix(filepath.Base(os.Args[0]), filepath.Ext(os.Args[0])))
		runFakeNative(name, os.Stdin, os.Stdout)
		os.Exit(0)
	}
	code := m.Run()
	if fake.folder != "" { // every fake worker has exited; Defender may hold a file briefly
		for attempt := 0; attempt < 20 && os.RemoveAll(fake.folder) != nil; attempt++ {
			time.Sleep(100 * time.Millisecond)
		}
	}
	os.Exit(code)
}

// fake is the one folder of fake workers per test process: copies of this binary (a hard
// link could not be deleted while the test binary runs).
var fake struct {
	once   sync.Once
	folder string
	err    error
}

// fakeNativeDir returns a folder whose studio_ied.exe and studio_control.exe are this test
// binary, and makes the children it starts act as fake workers.
func fakeNativeDir(t testing.TB) string {
	t.Helper()
	fake.once.Do(func() {
		executable, err := os.Executable()
		if err == nil {
			fake.folder, err = os.MkdirTemp("", "speedr-fake-")
		}
		for _, name := range []string{"studio_ied.exe", "studio_control.exe"} {
			if err == nil {
				err = copyFile(executable, filepath.Join(fake.folder, name))
			}
		}
		fake.err = err
	})
	if fake.err != nil {
		t.Fatal(fake.err)
	}
	t.Setenv(fakeEnv, "1")
	return fake.folder
}

func copyFile(source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(target)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// fakeWorker answers the requests the coordinator sends. Every 7th update_batch it also
// reports an MMS control of its first control object (control_applied), so native feedback
// runs while canvas commands, writes and Run/Stop compete for the runtime's locks.
type fakeWorker struct {
	out      *bufio.Writer
	values   map[string]any
	controls []any
	batches  int
	revision int64
	flip     bool
}

func runFakeNative(name string, stdin io.Reader, stdout io.Writer) {
	w := &fakeWorker{out: bufio.NewWriter(stdout), values: map[string]any{}}
	in := bufio.NewReader(stdin)
	for {
		message, err := native.ReadFrame(in)
		if err != nil {
			return
		}
		frame, _ := message.(*pyjson.Object)
		op := pyval.Str(frame, "op")
		result := w.answer(name, op, frame)
		w.write(pyval.Obj("id", pyval.Get(frame, "id"), "ok", true, "result", result))
		if op == "stop" {
			return
		}
		if op == "update_batch" {
			w.maybeControl()
		}
	}
}

func (w *fakeWorker) write(frame *pyjson.Object) {
	encoded, err := native.EncodeFrame(frame)
	if err != nil {
		return
	}
	_, _ = w.out.Write(encoded)
	_ = w.out.Flush()
}

func (w *fakeWorker) answer(name, op string, frame *pyjson.Object) any {
	switch op {
	case "hello":
		return pyval.Obj("capabilities", []any{native.MuxCapability, native.TypedCapability})
	case "start":
		w.controls = pyval.List(frame, "controls")
		capabilities := make([]any, len(native.IedRequired))
		for i, capability := range native.IedRequired {
			capabilities[i] = capability
		}
		return pyval.Obj("capabilities", capabilities, "ip", pyval.Get(frame, "ip"), "port", pyval.Get(frame, "port"),
			"goose", pyjson.NewObject())
	case "update":
		w.values[pyval.Str(frame, "ref")] = pyval.Get(frame, "value")
		return pyval.Get(frame, "value")
	case "update_batch", "update_guarded":
		written := pyjson.NewObject()
		for _, entry := range pyval.List(frame, "updates") {
			update := entry.(*pyjson.Object)
			w.values[pyval.Str(update, "ref")] = pyval.Get(update, "value")
			written.Set(pyval.Str(update, "ref"), pyval.Get(update, "value"))
		}
		if op == "update_guarded" {
			w.revision++
			return pyval.Obj("values", written, "revision", w.revision)
		}
		return written
	case "read":
		return w.values[pyval.Str(frame, "ref")]
	case "read_versioned":
		return pyval.Obj("value", w.values[pyval.Str(frame, "ref")], "revision", w.revision)
	case "typed_select": // ADR-0104: the fake selects every SBO control for 10 s
		return pyval.Obj("ok", true, "ref", pyval.Get(frame, "ref"), "selected", true, "expires_ms", int64(10000))
	case "typed_cancel":
		if os.Getenv(fakeNoCancelEnv) == "1" { // a model without the CO Cancel structure (change 0176)
			return pyval.Obj("ok", false, "ref", pyval.Get(frame, "ref"), "client_error", int64(22),
				"error", "Cancel refused by the device: object-does-not-exist")
		}
		return pyval.Obj("ok", true, "ref", pyval.Get(frame, "ref"), "canceled", true)
	case "typed_operate":
		time.Sleep(time.Millisecond)
		return pyval.Obj("ok", true, "ref", pyval.Get(frame, "ref"), "operated", op == "typed_operate",
			"canceled", op == "typed_cancel")
	case "canvas_operate":
		time.Sleep(time.Millisecond) // an association and one control round trip
		return pyval.Obj("ok", true)
	}
	return nil
}

func (w *fakeWorker) maybeControl() {
	w.batches++
	if w.batches%7 != 0 || len(w.controls) == 0 {
		return
	}
	control, _ := w.controls[0].(*pyjson.Object)
	w.flip = !w.flip
	value := int64(1)
	if w.flip {
		value = 2
	}
	w.write(pyval.Obj("event", "control_applied", "ref", pyval.Get(control, "ref"), "status_ref",
		pyval.Get(control, "status"), "value", value, "ctl_value", w.flip, "timestamp", time.Now().UnixMilli(),
		"peer", "127.67.0.9:50000"))
}
