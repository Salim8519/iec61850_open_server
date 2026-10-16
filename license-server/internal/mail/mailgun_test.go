package mail

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendPostsFormWithBasicAuth(t *testing.T) {
	var got map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, key, _ := r.BasicAuth()
		r.ParseForm()
		got = map[string]string{"path": r.URL.Path, "user": user, "key": key, "to": r.FormValue("to"),
			"text": r.FormValue("text")}
	}))
	defer server.Close()
	m := New("mg.example.com", "KEY", "Studio <l@example.com>", "us")
	m.BaseURL = server.URL
	subject, text := LicenseEmail("Studio", "SUBS-AAAA-BBBB-CCCC-DDDD", "https://lic.example.com/", "help@example.com")
	if err := m.Send(context.Background(), "buyer@example.com", subject, text); err != nil {
		t.Fatal(err)
	}
	if got["path"] != "/v3/mg.example.com/messages" || got["user"] != "api" || got["key"] != "KEY" {
		t.Fatalf("wrong request %v", got)
	}
	if !strings.Contains(got["text"], "SUBS-AAAA-BBBB-CCCC-DDDD") || !strings.Contains(got["text"], "https://lic.example.com/activate") {
		t.Fatal("email misses key or offline link")
	}
	if New("d", "k", "f", "eu").BaseURL != "https://api.eu.mailgun.net" {
		t.Fatal("EU region ignored")
	}
}
