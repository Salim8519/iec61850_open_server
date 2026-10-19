package httpapi

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"studio-engine/internal/api"
	"studio-engine/internal/apperr"
)

func testServer(t *testing.T, routes ...api.Route) *Server {
	s, err := New(Config{Token: "tok", SecretKey: "secret", Routes: routes})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func call(s http.Handler, method, target string) (int, string, http.Header) {
	r := httptest.NewRequest(method, "http://127.0.0.1:8765"+target, nil)
	r.RemoteAddr = "127.0.0.1:50000"
	r.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	body, _ := io.ReadAll(w.Result().Body)
	return w.Code, string(body), w.Result().Header
}

func echo(name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteBody(w, 200, "text/plain", []byte(name+":"+r.PathValue("key")+r.PathValue("edge_id")+r.PathValue("rest")))
	})
}

func TestRoutingLikeWerkzeug(t *testing.T) {
	s := testServer(t,
		api.Route{Method: "GET", Pattern: "/api/v1/ieds/{key}/settings", Handler: echo("dyn")},
		api.Route{Method: "GET", Pattern: "/api/v1/ieds/import/settings", Handler: echo("static")},
		api.Route{Method: "GET", Pattern: "/api/v1/edges/{edge_id}.csv", Handler: echo("csv")},
		api.Route{Method: "PUT", Pattern: "/api/v1/panel", Handler: echo("put")},
		api.Route{Method: "GET", Pattern: "/api/v1/panel", Handler: echo("get")},
		api.Route{Method: "GET", Pattern: "/api/v1/files/{rest...}", Handler: echo("rest")})
	cases := []struct {
		method, target string
		status         int
		body           string
	}{
		{"GET", "/api/v1/ieds/F1/settings", 200, "dyn:F1"},
		{"GET", "/api/v1/ieds/import/settings", 200, "static:"},
		{"GET", "/api/v1/edges/a.b.csv", 200, "csv:a.b"},
		{"GET", "/api/v1/edges/.csv", 404, TextNotFound},
		{"GET", "/api/v1/ieds//settings", 404, TextNotFound},
		{"GET", "/api/v1/panel/", 404, TextNotFound},
		{"DELETE", "/api/v1/panel", 405, TextMethodNotAllow},
		{"HEAD", "/api/v1/panel", 200, ""},
		{"GET", "/api/v1/files/x/y.txt", 200, "rest:x/y.txt"},
		{"GET", "/api/v1/files/", 404, TextNotFound},
	}
	for _, c := range cases {
		status, body, _ := call(s, c.method, c.target)
		if status != c.status || !strings.Contains(body, c.body) {
			t.Errorf("%s %s = %d %q; want %d %q", c.method, c.target, status, body, c.status, c.body)
		}
	}
	status, _, header := call(s, "OPTIONS", "/api/v1/panel")
	if status != 200 || header.Get("Allow") != "GET, HEAD, OPTIONS, PUT" {
		t.Errorf("OPTIONS = %d Allow %q", status, header.Get("Allow"))
	}
}

func TestErrorsLikeFlask(t *testing.T) {
	failing := func(err any) http.Handler {
		return http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(err) })
	}
	s := testServer(t,
		api.Route{Method: "GET", Pattern: "/api/v1/studio", Handler: failing(apperr.New("Bad name"))},
		api.Route{Method: "GET", Pattern: "/api/v1/conflict", Handler: failing(&apperr.Error{Kind: apperr.Studio,
			Message: "Expired", Code: "conflict", Details: []string{"a", "b"}})},
		api.Route{Method: "GET", Pattern: "/api/v1/key", Handler: failing(apperr.KeyError("ied_id"))},
		api.Route{Method: "GET", Pattern: "/api/v1/runtime", Handler: failing(apperr.RuntimeError("Native read request timed out"))},
		api.Route{Method: "GET", Pattern: "/api/v1/other", Handler: failing(errors.New("boom"))},
		api.Route{Method: "GET", Pattern: "/api/v1/returned", Handler: Handle(func(http.ResponseWriter, *http.Request) error {
			return ErrUnsupportedJSON
		})})
	want := map[string]string{
		"/api/v1/studio":   "400 {\"code\":\"validation\",\"details\":[],\"error\":\"Bad name\"}\n",
		"/api/v1/conflict": "409 {\"code\":\"conflict\",\"details\":[\"a\",\"b\"],\"error\":\"Expired\"}\n",
		"/api/v1/key":      "400 {\"error\":\"'ied_id'\"}\n",
		"/api/v1/runtime":  "503 {\"error\":\"Native read request timed out\"}\n",
		"/api/v1/other":    "500 {\"error\":\"" + TextInternal + "\"}\n",
		"/api/v1/returned": "415 {\"error\":\"" + TextUnsupportedJSON + "\"}\n",
	}
	for target, expected := range want {
		status, body, header := call(s, "GET", target)
		if line := strconv.Itoa(status) + " " + body; line != expected || header.Get("Content-Type") != "application/json" {
			t.Errorf("%s = %q; want %q", target, line, expected)
		}
	}
}

func TestLoopbackCaller(t *testing.T) {
	for address, want := range map[string]bool{"127.0.0.1:5": true, "127.9.9.9:5": true, "[::1]:5": true,
		"192.168.1.5:5": false, "[::ffff:127.0.0.1]:5": true, "garbage": false, "": true} {
		if got := loopbackCaller(address); got != want {
			t.Errorf("loopbackCaller(%q) = %v", address, got)
		}
	}
}
