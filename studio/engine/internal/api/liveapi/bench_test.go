package liveapi

import (
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"studio-engine/internal/acquisition/pyv"
)

var benchPaths = []string{"/live", "/goose", "/goose/links", "/mms/access", "/sv", "/health", "/health.csv",
	"/health/report.html", "/goose/share"}

// BenchmarkRoutes replays the Python scenario up to the running state (before Stop) and
// times each GET route, to compare with `STUDIO_BENCH=1 python gen_routes.py`.
func BenchmarkRoutes(b *testing.B) {
	t := &testing.T{}
	doc := loadScenario(t)
	e := newEnv(t, doc)
	defer e.service.Live.Stop()
	feed := ""
	for index, step := range pyv.Objects(pyv.GetList(doc, "steps")) {
		switch pyv.GetStr(step, "step") {
		case "project":
			e.projects.project = pyv.GetObj(step, "project")
			e.projects.runID = pyv.GetStr(step, "run_id")
		case "start":
			e.start(index, step)
			feed = e.service.Live.Sessions()[0].ID()
		case "event":
			e.inject(index, step)
		}
	}
	paths := append(append([]string{}, benchPaths...), "/live/"+feed+"/comparison",
		"/ieds/"+feed+"/signals?limit=100&q=stval")
	doc = nil // keep only what the engine holds (project, sessions), as in the app
	runtime.GC()
	for _, path := range paths {
		b.Run(strings.TrimPrefix(strings.SplitN(path, "?", 2)[0], "/"), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				recorder := httptest.NewRecorder()
				e.mux.ServeHTTP(recorder, httptest.NewRequest("GET", "/api/v1"+path, nil))
				if recorder.Code != 200 {
					b.Fatalf("%s: %d %s", path, recorder.Code, recorder.Body.String())
				}
			}
		})
	}
}
