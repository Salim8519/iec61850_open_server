package toolsapi

import (
	"strconv"
	"strings"
	"testing"

	"studio-engine/internal/gateway"
	"studio-engine/internal/testutil"
	"studio-engine/internal/tools/nativetest"
)

// TestMasterKeepsRunningWhileTheGatewayViewOpens: with the SCADA master connected to a real IEC 104
// device (studio_gateway.exe), the role switches to Station gateway and back without a refusal; only
// Protocol converter is locked, and /activity says so (ADR-0143).
func TestMasterKeepsRunningWhileTheGatewayViewOpens(t *testing.T) {
	var commands []string
	port := int64(testutil.FreePort(t, "127.0.0.1"))
	device(t, port, &commands)
	executables := map[string]string{"gateway": nativetest.Program(t, "studio_gateway.exe"),
		"master": nativetest.Program(t, "studio_master.exe")}
	deps := newDeps(&fakeProjects{}, &fakeRuntime{state: "stopped"}, executables)
	defer deps.Master.Disconnect(false)
	srv := server(t, deps)
	if _, text := call(t, srv, "GET", "/activity", ""); strings.Contains(text, "role_locks") {
		t.Fatalf("idle activity must stay as Python answers it: %s", text)
	}
	call(t, srv, "PUT", "/tools/master/config",
		`{"protocol":"iec104","host":"127.0.0.1","settings":{"port":`+strconv.FormatInt(port, 10)+`}}`)
	// Connect from the Station gateway view: any role may connect now.
	if _, text := call(t, srv, "POST", "/tools/master/connect", ""); !strings.Contains(text, `"state":"connected"`) {
		t.Fatal(text)
	}
	for _, role := range []string{"master", "gateway", "master"} {
		if status, text := call(t, srv, "POST", "/tools/gateway/role", `{"role":"`+role+`"}`); status != 200 {
			t.Fatalf("switch to %s while the master runs: %d %s", role, status, text)
		}
	}
	_, text := call(t, srv, "GET", "/activity", "")
	if !strings.Contains(text, `"role_locks":{"converter":"`+gateway.RefuseEnterConverter+`"}`) ||
		!strings.Contains(text, `"state":"connected"`) {
		t.Fatal(text)
	}
	if status, text := call(t, srv, "POST", "/tools/gateway/role", `{"role":"converter"}`); status != 400 ||
		!strings.Contains(text, gateway.RefuseEnterConverter) {
		t.Fatalf("%d %s", status, text)
	}
	if _, text := call(t, srv, "POST", "/tools/master/disconnect", ""); !strings.Contains(text, `"state":"disconnected"`) {
		t.Fatal(text)
	}
	if status, text := call(t, srv, "POST", "/tools/gateway/role", `{"role":"converter"}`); status != 200 {
		t.Fatalf("converter once nothing runs: %d %s", status, text)
	}
}
