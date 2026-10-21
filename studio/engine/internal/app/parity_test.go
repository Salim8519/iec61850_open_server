package app

import (
	"strings"
	"testing"
)

var parityVariants = []variant{
	{name: "dev", origin: "http://127.0.0.1:25173", license: "none", startup: "ok"},
	{name: "packaged", license: "none", startup: "ok"},
	{name: "packaged-unlicensed", license: "unlicensed", startup: "ok"},
	{name: "dev-startup-failed", origin: "http://127.0.0.1:25173", license: "none", startup: "error"},
	{name: "packaged-still-opening", license: "none", startup: "timeout"},
}

// sanity pins a few answers so a harness that compares nothing cannot pass.
var sanity = map[string]string{
	"dev/events stream":                          "200\nContent-Type: text/event-stream; charset=utf-8\nVary: Cookie\nCache-Control: no-cache\n\"retry: 1000\\n\\n\"",
	"dev/session ok":                             "Set-Cookie: studio_session=eyJhdXRoZW50aWNhdGVkIjp0cnVlfQ.<time>.<signature>; HttpOnly; Path=/; SameSite=Strict",
	"dev/no session":                             "401",
	"dev/dev frontend old tab":                   `active_frontend\":\"http://127.0.0.1:25173\"`,
	"packaged/origin foreign":                    `{\"active_frontend\":null,\"error\":\"Cross-origin administration is not permitted\"}`,
	"packaged/events bad cursor":                 `invalid literal for int() with base 10: 'abc'`,
	"packaged-unlicensed/cookie unknown path":    "402",
	"dev-startup-failed/bearer unknown path":     "startup_recovery_failed",
	"packaged-still-opening/bearer unknown path": "project_opening",
	"dev/wrong method":                           "405",
	"dev/session list":                           "500",
}

// TestFlaskParity sends the same requests to the frozen Python app (create_app served by
// Waitress, run by the venv Python) and to the Go app, and requires identical status,
// headers and bodies. One Python process runs at a time (STUDIO_PYTHON must be set).
func TestFlaskParity(t *testing.T) {
	frontend := syntheticFrontend(t)
	for _, v := range parityVariants {
		t.Run(v.name, func(t *testing.T) {
			pythonURL := startPython(t, v, frontend)
			goURL := startGo(t, v, frontend)
			pythonCookie, goCookie := sessionCookie(t, pythonURL), sessionCookie(t, goURL)
			for _, item := range parityRequests {
				want := exchange(t, pythonURL, item, pythonCookie)
				got := exchange(t, goURL, item, goCookie)
				if got != want {
					t.Errorf("%s %s %s (%s)\n--- python\n%s\n--- go\n%s", v.name, item.method, item.path,
						item.name, want, got)
				}
				if expected, listed := sanity[v.name+"/"+item.name]; listed && !strings.Contains(got, expected) {
					t.Errorf("%s (%s): the harness itself looks wrong, %q not in\n%s", v.name, item.name, expected, got)
				}
			}
			crossCookies(t, pythonURL, goURL, pythonCookie, goCookie)
		})
	}
}

// crossCookies: a tab signed in on one engine stays signed in after switching engines
// (same SECRET_KEY), so each engine accepts the other's cookie.
func crossCookies(t *testing.T, pythonURL, goURL, pythonCookie, goCookie string) {
	probe := request{method: "GET", path: "/api/v1/nope", auth: "cookie"}
	onPython := exchange(t, pythonURL, probe, goCookie)
	onGo := exchange(t, goURL, probe, pythonCookie)
	if onPython != onGo || strings.HasPrefix(onGo, "401") {
		t.Errorf("cross-engine cookies: Go cookie on Python %q, Python cookie on Go %q", onPython, onGo)
	}
	forged := exchange(t, goURL, probe, goCookie+"x")
	if !strings.HasPrefix(forged, "401") {
		t.Errorf("a changed signature must not sign in: %q", forged)
	}
}
