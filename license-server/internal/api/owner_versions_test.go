package api

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestOwnerVersionsCompareWebsiteAndUpdateCheck(t *testing.T) {
	h := newOwnerHarness(t)
	calls := 0
	h.s.owner.site.fetch = func(_ context.Context, url string) (string, error) {
		calls++
		if url != "https://scadabench.com/download/" {
			t.Fatalf("wrong page %s", url)
		}
		return `<a href="/download/ScadaBench-Setup-0.1.13.exe">Download</a>`, nil
	}
	h.login()
	_, result, _ := h.do("GET", "/owner-test/api/versions", nil, nil)
	if result["website"] != "0.1.13" || result["in_step"] != false || result["release"] != nil {
		t.Fatalf("without a published release apps are not in step: %v", result)
	}
	record := `{"version":"0.1.13","date":"2026-10-19","notes":"x","page":"https://scadabench.com/download",` +
		`"installer":{"url":"https://scadabench.com/download/ScadaBench-Setup-0.1.13.exe","sha256":"` +
		"0000000000000000000000000000000000000000000000000000000000000000" + `"}}`
	os.WriteFile(filepath.Join(h.s.Config.DataDir, "release.json"), []byte(record), 0o600)
	_, result, _ = h.do("GET", "/owner-test/api/versions", nil, nil)
	if result["in_step"] != true || calls != 1 {
		t.Fatalf("published 0.1.13 must be in step and the website read once per 5 minutes: %v (calls %d)", result, calls)
	}
}
