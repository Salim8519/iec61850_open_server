package runtimeapi

import (
	"strings"
	"testing"
)

// GOOSE send route (ADR-0112): plain-words answers while stopped, unknown nodes and refused
// bodies. Sending on a run is tested in the runtime package on its fake workers.
func TestGooseSendRouteWhileStopped(t *testing.T) {
	server, _, _ := newServer(t, 21360)
	cases := []struct {
		path, body string
		status     int
		want       string
	}{
		{"/equipment/cb1/goose", `{"action":"open"}`, 400, "Run the IEDs to send GOOSE. Press Run, then try again."},
		{"/equipment/cb1/goose", `[1]`, 400, "A GOOSE send must be a JSON object"},
		{"/equipment/nope/goose", `{}`, 400, "Unknown equipment"},
	}
	for _, test := range cases {
		status, body := call(t, server, "POST", test.path, "application/json", test.body)
		if status != test.status || !strings.Contains(body, test.want) {
			t.Errorf("%s %s: %d %s, want %d with %q", test.path, test.body, status, body, test.status, test.want)
		}
	}
}
