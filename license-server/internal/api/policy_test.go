package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// pageText returns a served page with all whitespace folded, so sentences that wrap in the
// template still match.
func pageText(t *testing.T, s *Server, path string) string {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.RemoteAddr = "127.0.0.1:1"
	s.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("%s answered %d", path, recorder.Code)
	}
	return strings.Join(strings.Fields(recorder.Body.String()), " ")
}

func TestCustomerPagesStateTheOneComputerPolicyAndNeverOfferMoves(t *testing.T) {
	s := newTestServer(t)
	for _, path := range []string{"/buy", "/activate"} {
		text := pageText(t, s, path)
		for _, want := range []string{
			"Each license works on one computer only.",
			"cannot be moved to another computer",
			"Reinstalling Windows on the same computer is fine.",
			"a new motherboard, or a reset of the computer's security chip (TPM) needs a new license.",
		} {
			if !strings.Contains(text, want) {
				t.Fatalf("%s misses %q", path, want)
			}
		}
		for _, banned := range []string{"times a year", "Move the key", "Move the license", "move it to this computer",
			"/api/v1/transfer", "id=\"move\""} {
			if strings.Contains(text, banned) {
				t.Fatalf("%s still contains %q", path, banned)
			}
		}
	}
	if text := pageText(t, s, "/buy"); !strings.Contains(text, "No refunds after a license is activated.") ||
		!strings.Contains(text, "Use the free 7-day trial to test ScadaBench first.") {
		t.Fatal("the buy page must state the refund rule and point to the trial")
	}
}
