package toolsapi

import (
	"testing"

	"studio-engine/internal/api/runtimeapi"
	"studio-engine/internal/httpapi"
)

// TestComposesWithRuntimeRoutes: the server refuses a (method, path) registered twice, so the
// tools area must leave GET /project/test-panel to runtimeapi and still mount beside it.
func TestComposesWithRuntimeRoutes(t *testing.T) {
	routes := append(New(Deps{}).Routes(), runtimeapi.Area{}.Routes()...)
	if _, err := httpapi.New(httpapi.Config{Routes: routes}); err != nil {
		t.Fatal(err)
	}
	put := false
	for _, route := range New(Deps{}).Routes() {
		if route.Pattern == "/api/v1/project/test-panel" {
			if route.Method != "PUT" {
				t.Fatalf("tools area serves %s /project/test-panel", route.Method)
			}
			put = true
		}
	}
	if !put {
		t.Fatal("PUT /project/test-panel missing from the tools area")
	}
}
