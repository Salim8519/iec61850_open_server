package licensing

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

type sentRequest struct{ method, path, body string }

// recordServer answers the proof-of-work challenge and "{}" to everything else, and records
// what the client sent.
func recordServer(sent *[]sentRequest) doerFunc {
	return func(request *http.Request) (*http.Response, error) {
		body := ""
		if request.Body != nil {
			data, _ := io.ReadAll(request.Body)
			body = string(data)
		}
		*sent = append(*sent, sentRequest{request.Method, request.URL.Path, body})
		answer := "{}"
		if request.URL.Path == "/api/v1/challenge" {
			answer = `{"nonce": "n1", "difficulty": 1, "timestamp": 5, "mac": "m1"}`
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(answer))}, nil
	}
}

func TestTheChipOfferTravelsWithTheActivationAndTheAnswerGoesToItsOwnRoute(t *testing.T) {
	var sent []sentRequest
	client := NewHTTPClient(DefaultServerURL, recordServer(&sent))
	offer := pyjson.NewObject().Set("ek_cert", "Q0VSVA==").Set("ek_public", "UFVC").
		Set("ak", pyjson.NewObject().Set("public", "QUs="))
	if _, err := client.ActivateWithChip("SUBS-X", "ab", offer); err != nil {
		t.Fatal(err)
	}
	last := sent[len(sent)-1]
	if last.method != "POST" || last.path != "/api/v1/activate" ||
		!strings.HasPrefix(last.body, `{"license_key": "SUBS-X", "hwid": "ab", "tpm": {"ek_cert": "Q0VSVA==", "ek_public": "UFVC", `+
			`"ak": {"public": "QUs="}}, "pow_nonce": "n1"`) {
		t.Fatalf("activation request %s %s", last.path, last.body)
	}
	sent = nil
	if _, err := client.ActivateWithChip("SUBS-X", "ab", nil); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(sent[len(sent)-1].body, "tpm") {
		t.Fatalf("a request without a chip carried one: %s", sent[len(sent)-1].body)
	}
	sent = nil
	if _, err := client.ActivateChip("s1", "c2VjcmV0"); err != nil {
		t.Fatal(err)
	}
	if len(sent) != 1 || sent[0].path != "/api/v1/activate/tpm" || sent[0].body != `{"session": "s1", "secret": "c2VjcmV0"}` {
		t.Fatalf("answer request %+v: the challenge answer needs no proof of work", sent)
	}
}

func TestTheLeaseTokenIsSentOnlyWhenThereIsOne(t *testing.T) {
	var sent []sentRequest
	client := NewHTTPClient(DefaultServerURL, recordServer(&sent))
	client.Verify("SUBS-X", "ab", "00")
	client.VerifyLease("SUBS-X", "ab", "00", strings.Repeat("a", 32))
	if sent[0].body != `{"license_key": "SUBS-X", "hwid": "ab", "nonce": "00"}` {
		t.Fatalf("the plain verify changed: %s", sent[0].body)
	}
	if sent[1].path != "/api/v1/verify" || sent[1].body != `{"license_key": "SUBS-X", "hwid": "ab", "nonce": "00", "lease_token": "`+strings.Repeat("a", 32)+`"}` {
		t.Fatalf("lease verify %s %s", sent[1].path, sent[1].body)
	}
}

func TestAChallengeIsReadOnlyWhenComplete(t *testing.T) {
	good := mustDecode(t, `{"challenge": {"session": "s", "credential": "Y3JlZA==", "secret": "c2VjcmV0"}}`)
	challenge, found := challengeOf(good)
	if !found || challenge.Session != "s" || string(challenge.Credential) != "cred" || string(challenge.Secret) != "secret" {
		t.Fatalf("%+v %v", challenge, found)
	}
	for _, bad := range []string{`{}`, `{"certificate": {}}`, `{"challenge": "x"}`, `{"challenge": {"session": "", "credential": "Y3JlZA==", "secret": "c2VjcmV0"}}`,
		`{"challenge": {"session": "s", "credential": "!!", "secret": "c2VjcmV0"}}`, `{"challenge": {"session": "s", "credential": "Y3JlZA==", "secret": ""}}`,
		`{"challenge": {"session": "s", "credential": "Y3JlZA=="}}`} {
		if _, found := challengeOf(mustDecode(t, bad)); found {
			t.Errorf("%s was taken for a challenge", bad)
		}
	}
	if _, found := challengeOf("text"); found {
		t.Error("text was taken for a challenge")
	}
}

func TestTheSignedAnswerCarriesTheLeaseTokenAndOldAnswersStillWork(t *testing.T) {
	env := newBindEnv(t, nil)
	token := strings.Repeat("a", 32)
	answer := func(extra map[string]any) any {
		payload := map[string]any{"hwid": "ab", "license_id": bindKey, "nonce": "n", "purpose": VerifyAnswerPurpose,
			"status": "valid", "time": 77, "version": 1}
		for key, value := range extra {
			payload[key] = value
		}
		return env.server.decode(map[string]any{"status": "valid", "answer": env.server.sign(payload)})
	}
	keys := env.server.keys()
	verified, ok := VerifyAnswer(answer(map[string]any{"lease_token": token}), "n", bindKey, "ab", keys)
	if !ok || verified.Status != "valid" || verified.LeaseToken != token || verified.Time != 77 || verified.Bundle.Payload == "" {
		t.Fatalf("%+v %v", verified, ok)
	}
	for name, extra := range map[string]map[string]any{"no lease_token key (a Python-shaped answer)": nil,
		"empty token": {"lease_token": ""}, "malformed token": {"lease_token": "XYZ"}, "not text": {"lease_token": 5}} {
		verified, ok := VerifyAnswer(answer(extra), "n", bindKey, "ab", keys)
		if !ok || verified.LeaseToken != "" {
			t.Errorf("%s: %+v %v", name, verified, ok)
		}
	}
	if _, ok := VerifyAnswer(answer(nil), "other", bindKey, "ab", keys); ok {
		t.Error("an answer for another nonce counted")
	}
	if _, ok := VerifyAnswer(answer(nil), "n", bindKey, "cd", keys); ok {
		t.Error("an answer for another computer counted")
	}
}
