package selftest

import (
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"studio-engine/internal/app"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/scl"
)

const (
	selfTestHost = "127.54.0.254"
	alarm        = "SelfTestCTRL/XCBR1.Loc.stVal"
	reference    = "SelfTestCTRL/GGIO1.AnIn1.mag.f"
	arrayRef     = "SelfTestCTRL/GGIO1.AnIn2.samples(2).mag.f"
)

// licensedRun is the first half of check(): the licensed engine on the temporary folder,
// closed afterwards whatever happened (Python's finally: runtime.close()).
func (r *run) licensedRun() error {
	engine, err := r.compose("isolated-self-test")
	if err != nil {
		return err
	}
	status, err := engine.License.Current(true)
	if err != nil {
		_ = engine.Close()
		return err
	}
	if !status.Licensed {
		_ = engine.Close()
		return errors.New(strings.TrimSpace("The self-test runs the licensed engine, so activate ScadaBench on " +
			"this computer first. " + reasonText(status.Reason)))
	}
	err = r.exercise(engine)
	if closed := engine.Close(); err == nil {
		err = closed
	}
	return err
}

// reasonText is the f-string of license_status.reason, empty when it is None or empty.
func reasonText(reason any) string {
	if !pyval.Truthy(reason) {
		return ""
	}
	return pyval.Repr(reason)
}

// exercise is the try block of check(): session, import, run, pulse, values, export.
func (r *run) exercise(engine *app.Engine) error {
	client := newClient(engine.Handler)
	if client.get("/api/v1/project").status != http.StatusUnauthorized {
		return failed("GET /api/v1/project without a session did not answer 401", "")
	}
	session, err := client.sendJSON(http.MethodPost, "/api/v1/session", pyval.Obj("token", "isolated-self-test"))
	if err != nil {
		return err
	}
	if err := expectStatus(session, http.StatusOK, "POST /api/v1/session did not answer 200", false); err != nil {
		return err
	}
	source, err := Fixture()
	if err != nil {
		return err
	}
	names, err := pyjson.Dumps([]any{"SelfTest"})
	if err != nil {
		return err
	}
	imported, err := client.upload("/api/v1/scl/import", "self-test.cid", source, "selected_names", string(names))
	if err != nil {
		return err
	}
	if imported.status != http.StatusOK {
		return errors.New(pyval.Repr(imported.json())) // RuntimeError(response.get_json())
	}
	ied := firstIED(engine.Projects.Project())
	if ied == nil {
		return failed("the SelfTest IED was not imported", "")
	}
	if validation := pyval.Map(ied, "validation"); pyval.Str(validation, "status") != "valid" {
		return failed("the imported SelfTest IED is not schema-valid", pyval.Repr(validation))
	}
	port, err := freePort()
	if err != nil {
		return err
	}
	id := pyval.Str(ied, "id")
	if err := placeIED(engine, port); err != nil {
		return err
	}
	panel, err := client.sendJSON(http.MethodPut, "/api/v1/project/test-panel", []any{
		pyval.Obj("id", "alarm", "label", "Local mode", "ied_id", id, "ref", alarm)})
	if err != nil {
		return err
	}
	if err := expectStatus(panel, http.StatusOK, "PUT /api/v1/project/test-panel did not answer 200", false); err != nil {
		return err
	}
	if err := r.runValues(engine, id, strconv.Itoa(port)); err != nil {
		return err
	}
	if err := r.verifyExport(engine, client, id); err != nil {
		return err
	}
	r.identity = pyval.Get(engine.Projects.Project(), "id")
	return nil
}

// freePort is a socket bound to (127.54.0.254, 0): the port the system picked.
func freePort() (int, error) {
	listener, err := net.Listen("tcp4", net.JoinHostPort(selfTestHost, "0"))
	if err != nil {
		return 0, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	return port, listener.Close()
}

// placeIED is ied.update(ip="127.54.0.254", port=port) on the project's first IED.
func placeIED(engine *app.Engine, port int) error {
	_, err := engine.Projects.Update(func(draft *pyjson.Object) (*pyjson.Object, error) {
		ied := firstIED(draft)
		if ied == nil {
			return nil, errors.New("the SelfTest IED disappeared")
		}
		ied.Set("ip", selfTestHost).Set("port", int64(port))
		return nil, nil
	})
	return err
}

// runValues runs the IED, checks the 100 ms pulse restored the alarm in the native worker,
// then writes the analog value and one array element and reads both back with the probe.
func (r *run) runValues(engine *app.Engine, id, port string) error {
	coordinator := engine.Runtime
	if err := coordinator.Start(nil); err != nil {
		return err
	}
	if _, err := coordinator.Pulse(id, alarm, true, int64(100)); err != nil {
		return err
	}
	deadline := time.Now().Add(3 * time.Second)
	for coordinator.PendingPulses() > 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	restored, err := coordinator.ReadNative(id, alarm)
	if err != nil {
		return err
	}
	if value, ok := restored.(bool); !ok || value {
		return failed("the 100 ms pulse did not restore "+alarm+" to false", "")
	}
	for _, write := range []struct {
		ref   string
		value float64
	}{{reference, 42.5}, {arrayRef, 17.25}} {
		if _, err := coordinator.Update(id, write.ref, write.value, nil, "manual"); err != nil {
			return err
		}
		observed, err := readNative(r.options.NativeDir, selfTestHost, port, write.ref)
		if err != nil {
			return err
		}
		if observed != write.value {
			return failed("studio_probe.exe read "+write.ref+" as "+pyval.Repr(observed)+", expected "+
				pyval.Repr(write.value), "")
		}
	}
	return nil
}

// verifyExport exports the IED with AnIn1 edited to 9.5 and imports the exported bytes again.
func (r *run) verifyExport(engine *app.Engine, client *client, id string) error {
	exported, err := client.sendJSON(http.MethodPost, "/api/v1/ieds/"+id+"/export-scl",
		pyval.Obj("overrides", pyval.Obj(reference, 9.5)))
	if err != nil {
		return err
	}
	if err := expectStatus(exported, http.StatusOK, "the edited SCL export did not answer 200", true); err != nil {
		return err
	}
	models, err := engine.Projects.Importer().Import(exported.data, "import.scd",
		scl.Selection{Names: []any{"SelfTest"}})
	if err != nil {
		return err
	}
	if len(models) == 0 {
		return failed("the exported SCL has no SelfTest IED", "")
	}
	for _, item := range pyval.List(models[0], "signals") {
		signal, _ := item.(*pyjson.Object)
		if pyval.Str(signal, "ref") == reference {
			if !pyval.Equal(pyval.Get(signal, "value"), 9.5) {
				return failed("the exported SCL did not keep the edited value 9.5 for "+reference, "")
			}
			return nil
		}
	}
	return failed("the exported SCL has no signal "+reference, "")
}
