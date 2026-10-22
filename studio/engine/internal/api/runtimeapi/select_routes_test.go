package runtimeapi

import (
	"strings"
	"testing"
)

// Select and cancel routes (ADR-0108): plain-words answers while stopped, unknown nodes, a
// cancel without a selection, and refused bodies. The running flow is tested in the runtime
// package on its fake workers.
func TestSelectRoutesWhileStopped(t *testing.T) {
	server, _, _ := newServer(t, 21340)
	cases := []struct {
		path, body string
		status     int
		want       string
	}{
		{"/equipment/cb1/select", `{"action":"close"}`, 400, "Run the IEDs before operating equipment"},
		{"/equipment/cb1/select", `[1]`, 400, "Equipment selection must be a JSON object"},
		{"/equipment/nope/cancel", `{}`, 400, "Unknown equipment"},
		{"/equipment/cb1/cancel", `{}`, 200, `"cancelled":false`},
		{"/equipment/cb1/operate", `{"action":"close","options":{"bogus":true}}`, 400, "Run the IEDs"},
	}
	for _, test := range cases {
		status, body := call(t, server, "POST", test.path, "application/json", test.body)
		if status != test.status || !strings.Contains(body, test.want) {
			t.Errorf("%s %s: %d %s, want %d with %q", test.path, test.body, status, body, test.status, test.want)
		}
	}
}

func TestGoOnlyRoutesAreServed(t *testing.T) {
	served := map[string]bool{}
	for _, route := range (Area{}).Routes() {
		served[route.Method+" "+route.Pattern] = true
	}
	for key := range GoOnlyRoutes {
		if !served[key] {
			t.Errorf("%s is listed as Go-only but not served", key)
		}
	}
}
