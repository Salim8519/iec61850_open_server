package platformapi

import (
	"sort"
	"strings"
	"testing"
)

// pythonPlatformRoutes are the Flask url_map entries of api/license.py, api/updates.py and
// api/network.py (without PUT /ieds/<key>/settings, which the project area serves).
var pythonPlatformRoutes = []string{
	"GET /api/v1/app/update", "PUT /api/v1/app/update", "POST /api/v1/app/update/check",
	"POST /api/v1/license/activate", "POST /api/v1/license/activate-offline", "PUT /api/v1/license/config",
	"GET /api/v1/license/status", "POST /api/v1/license/trial", "POST /api/v1/license/trial/email-code",
	"POST /api/v1/license/check-online",
	"GET /api/v1/network", "PUT /api/v1/network", "GET /api/v1/network/adapters",
	"POST /api/v1/network/cleanup/apply", "GET /api/v1/network/cleanup/preview",
	"GET /api/v1/network/gateway-firewall", "POST /api/v1/network/gateway-firewall",
	"POST /api/v1/network/plan/apply", "POST /api/v1/network/plan/preview",
}

func TestRoutesMatchFlask(t *testing.T) {
	served := []string{}
	for _, route := range New(Deps{}).Routes() {
		if route.Handler == nil || route.Polled {
			t.Errorf("%s %s: handler %v, polled %v", route.Method, route.Pattern, route.Handler, route.Polled)
		}
		if key := route.Method + " " + route.Pattern; !GoOnlyRoutes[key] {
			served = append(served, key)
		}
	}
	want := append([]string(nil), pythonPlatformRoutes...)
	sort.Strings(served)
	sort.Strings(want)
	if strings.Join(served, "\n") != strings.Join(want, "\n") {
		t.Fatalf("routes\n%s\nwant\n%s", strings.Join(served, "\n"), strings.Join(want, "\n"))
	}
}
