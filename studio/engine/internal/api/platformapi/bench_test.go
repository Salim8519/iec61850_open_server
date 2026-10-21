package platformapi

import (
	"net/http/httptest"
	"testing"
)

// BenchmarkPlatformReads measures the read routes the UI calls when a settings page opens,
// handler only (no security pipeline), with the fakes of the golden replay.
func BenchmarkPlatformReads(b *testing.B) {
	h := newHarness(b, loadGolden(b))
	h.newLicense(false, false) // development mode, as the Python timing (an enforced app answers 402 there)
	for _, path := range []string{"/api/v1/network", "/api/v1/network/adapters", "/api/v1/app/update",
		"/api/v1/license/status", "/api/v1/network/cleanup/preview"} {
		b.Run(path[len("/api/v1/"):], func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				recorder := httptest.NewRecorder()
				h.mux.ServeHTTP(recorder, httptest.NewRequest("GET", "http://localhost"+path, nil))
				if recorder.Code != 200 {
					b.Fatalf("%s: %d %s", path, recorder.Code, recorder.Body.String())
				}
			}
		})
	}
}
