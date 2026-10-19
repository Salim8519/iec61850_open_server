package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"substation-license-server/internal/certs"
)

func stampOf(t *testing.T, s *Server, result map[string]any) certs.TimeStamp {
	t.Helper()
	raw, _ := json.Marshal(result["server_time"])
	var bundle certs.Bundle
	json.Unmarshal(raw, &bundle)
	stamp, err := certs.VerifyTime(s.Signer.Public(), bundle)
	if err != nil {
		t.Fatalf("no valid signed server time: %v (%v)", err, result)
	}
	return stamp
}

func TestEveryLicensingAnswerCarriesSignedServerTime(t *testing.T) {
	s := newTestServer(t)
	hwid := strings.Repeat("a", 64)
	_, result := call(t, s, "/api/v1/trial", map[string]any{"hwid": hwid}, true)
	if stamp := stampOf(t, s, result); stamp.Time < s.Now().Unix()-2 || stamp.KeyID != s.Signer.KeyID {
		t.Fatalf("wrong stamp %+v", stamp)
	}
	_, result = call(t, s, "/api/v1/verify", map[string]any{"hwid": hwid, "license_key": "TRIAL-" + hwid[:12]}, false)
	stampOf(t, s, result)
	license, _, _ := s.IssueKey("buyer@example.com", "lifetime", 0, "")
	_, result = call(t, s, "/api/v1/activate", map[string]any{"hwid": hwid, "license_key": license.Key}, true)
	stampOf(t, s, result)
	// A stamp is never a certificate: it has no type, machine or license.
	raw, _ := json.Marshal(result["server_time"])
	if strings.Contains(string(raw), "hwid") || strings.Contains(string(raw), "lifetime") {
		t.Fatal("time stamp leaks certificate fields")
	}
}

func hardware(firmware string) map[string]any {
	return map[string]any{"guid": certs.PartHash("guid", "g1"), "volume": certs.PartHash("volume", "AAAA0001"),
		"firmware": certs.PartHash("firmware", firmware)}
}

func expireTrials(s *Server) {
	start := s.Now()
	s.Now = func() time.Time { return start.Add(30 * 24 * time.Hour) }
}

func TestSuspiciousTrialConfirmsAnEmailCodeOncePerAddress(t *testing.T) {
	s := newTestServer(t)
	mailer := &fakeMail{}
	s.Mailer = mailer
	call(t, s, "/api/v1/trial", map[string]any{"hwid": strings.Repeat("a", 64), "parts": hardware("11112222333344445555666677778888")}, true)
	expireTrials(s)
	clone := map[string]any{"hwid": strings.Repeat("b", 64), "parts": hardware("99998888777766665555444433332222")}
	code, result := call(t, s, "/api/v1/trial", clone, true)
	if code != http.StatusConflict || result["code"] != "email_required" {
		t.Fatalf("2 of 3 match: %d %v", code, result)
	}
	ask := map[string]any{"hwid": strings.Repeat("b", 64), "email": "Lab.User+trial@Example.com"}
	if code, result = call(t, s, "/api/v1/trial/email-code", ask, true); code != http.StatusOK || result["status"] != "sent" {
		t.Fatalf("code request: %d %v", code, result)
	}
	sent := regexp.MustCompile(`code is (\d{6})`).FindStringSubmatch(mailer.sent[0])
	if len(sent) != 2 || !strings.HasPrefix(mailer.sent[0], "Lab.User+trial@Example.com|") {
		t.Fatalf("mail not sent to the typed address: %v", mailer.sent)
	}
	wrong := map[string]any{"hwid": strings.Repeat("b", 64), "parts": clone["parts"], "email": "lab.user@example.com", "email_code": "000000"}
	if sent[1] != "000000" {
		if code, result = call(t, s, "/api/v1/trial", wrong, true); result["code"] != "bad_code" {
			t.Fatalf("wrong code: %d %v", code, result)
		}
	}
	wrong["email_code"] = sent[1]
	if code, result = call(t, s, "/api/v1/trial", wrong, true); code != http.StatusOK {
		t.Fatalf("right code: %d %v", code, result)
	}
	again := map[string]any{"hwid": strings.Repeat("c", 64), "email": "lab.user@example.com"}
	if _, result = call(t, s, "/api/v1/trial/email-code", again, true); result["code"] != "email_used" {
		t.Fatalf("one address, two trials: %v", result)
	}
	throwaway := map[string]any{"hwid": strings.Repeat("c", 64), "email": "x@mailinator.com"}
	if _, result = call(t, s, "/api/v1/trial/email-code", throwaway, true); result["code"] != "email_disposable" {
		t.Fatalf("disposable address: %v", result)
	}
	if _, err := s.RunAdmin("allow-email", AdminRequest{TrialEmail: "LAB.user@example.com"}); err != nil {
		t.Fatal(err)
	}
	if _, result = call(t, s, "/api/v1/trial/email-code", again, true); result["status"] != "sent" {
		t.Fatalf("owner allow-email did not help: %v", result)
	}
}

func TestMailOutageNeverCountsAgainstTheUserAndLogsHoldNoSecrets(t *testing.T) {
	s := newTestServer(t) // no Mailer: email delivery is down or not set up
	for n, hwid := range []string{"1", "2", "3"} {
		if code, result := call(t, s, "/api/v1/trial", map[string]any{"hwid": strings.Repeat(hwid, 64)}, true); code != http.StatusOK {
			t.Fatalf("office colleague %d: %d %v", n+1, code, result)
		}
	}
	fourth := map[string]any{"hwid": strings.Repeat("4", 64)}
	if _, result := call(t, s, "/api/v1/trial", fourth, true); result["code"] != "email_required" {
		t.Fatalf("busy network did not ask for an email: %v", result)
	}
	fourth["email"] = "fourth@office.example"
	if _, result := call(t, s, "/api/v1/trial/email-code", fourth, true); result["status"] != "no_code_needed" {
		t.Fatalf("mail outage: %v", result)
	}
	if code, result := call(t, s, "/api/v1/trial", fourth, true); code != http.StatusOK {
		t.Fatalf("trial refused during a mail outage: %d %v", code, result)
	}
	logged, _ := os.ReadFile(filepath.Join(s.Config.LogDir, "server.log"))
	for _, secret := range []string{strings.Repeat("4", 64), "fourth@office.example", "office.example"} {
		if strings.Contains(string(logged), secret) {
			t.Fatalf("log contains %q", secret)
		}
	}
	for _, kind := range []string{"trial_refused", "busy_ip", "trial_email_unsent"} {
		if !strings.Contains(string(logged), kind) {
			t.Fatalf("log misses %s: %s", kind, logged)
		}
	}
	if listed, err := s.RunAdmin("list", AdminRequest{}); err != nil || listed.(map[string]any)["busy_networks"] != 1 {
		t.Fatalf("owner list: %v %v", listed, err)
	}
	if result, err := s.RunAdmin("reset-trial-ip", AdminRequest{IP: "127.0.0.1"}); err != nil || result.(map[string]any)["had_count"] != true {
		t.Fatalf("reset-trial-ip: %v %v", result, err)
	}
}
