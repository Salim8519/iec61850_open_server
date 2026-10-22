package toolsapi

import (
	"strconv"
	"strings"
	"testing"

	"studio-engine/internal/testutil"
	"studio-engine/internal/tools/nativetest"
)

// TestMasterExtraCommands exercises the change-0169 additions against ScadaBench's own 104
// outstation (studio_gateway): deactivate (COT 8), the regulating step type, the QU check and
// the cyclic sender. Master and outstation share lib60870, so this is not an independent check;
// it confirms the Go validation and sequencing and that the frames reach a controlled station.
func TestMasterExtraCommands(t *testing.T) {
	var commands []string
	port := int64(testutil.FreePort(t, "127.0.0.1"))
	device(t, port, &commands)
	execs := map[string]string{"gateway": nativetest.Program(t, "studio_gateway.exe"),
		"master": nativetest.Program(t, "studio_master.exe")}
	deps := newDeps(&fakeProjects{}, &fakeRuntime{state: "stopped"}, execs)
	defer deps.Master.Disconnect(false)
	srv := server(t, deps)
	call(t, srv, "POST", "/tools/gateway/role", `{"role":"master"}`)
	call(t, srv, "PUT", "/tools/master/config",
		`{"protocol":"iec104","host":"127.0.0.1","settings":{"port":`+strconv.FormatInt(port, 10)+`}}`)
	if _, text := call(t, srv, "POST", "/tools/master/connect", ""); !strings.Contains(text, `"state":"connected"`) {
		t.Fatal(text)
	}

	// A qualifier outside 0-31 is refused before anything is sent.
	if status, text := call(t, srv, "POST", "/tools/master/command",
		`{"kind":"double","address":6001,"value":2,"qualifier":99,"confirm":true}`); status != 400 ||
		!strings.Contains(text, "qualifier") {
		t.Fatalf("%d %s", status, text)
	}

	// Deactivate (COT 8) of a double command: the outstation confirms the cancel, no termination.
	if _, text := call(t, srv, "POST", "/tools/master/command",
		`{"kind":"double","address":6001,"value":2,"deactivate":true,"confirm":true}`); !strings.Contains(text, `"ok":true`) ||
		!strings.Contains(text, "deactivation confirmed") || strings.Contains(text, `"terminated":true`) {
		t.Fatalf("deactivate: %s", text)
	}

	// The regulating step (C_RC) is accepted (Go-only kind), not rejected as "not available".
	if _, text := call(t, srv, "POST", "/tools/master/command",
		`{"kind":"regulating","address":6001,"value":2,"confirm":true}`); strings.Contains(text, "not available") {
		t.Fatalf("regulating refused by validation: %s", text)
	}

	// Cyclic: a too-fast interval is refused; a valid one runs and Stop ends it.
	if status, text := call(t, srv, "POST", "/tools/master/cyclic",
		`{"kind":"double","address":6001,"value":2,"interval_ms":10,"confirm":true}`); status != 400 ||
		!strings.Contains(text, "cycle time") {
		t.Fatalf("%d %s", status, text)
	}
	if _, text := call(t, srv, "POST", "/tools/master/cyclic",
		`{"kind":"double","address":6001,"value":2,"interval_ms":200,"address_step":1,"toggle":true,"confirm":true}`); !strings.Contains(text, `"running":true`) {
		t.Fatalf("cyclic start: %s", text)
	}
	waitFor(func() bool {
		_, text := call(t, srv, "GET", "/tools/master/cyclic", "")
		return strings.Contains(text, `"running":true`) && !strings.Contains(text, `"sends":0`)
	})
	if _, text := call(t, srv, "POST", "/tools/master/cyclic/stop", ""); !strings.Contains(text, `"running":false`) {
		t.Fatalf("cyclic stop: %s", text)
	}
}
