package platformapi

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/licensing"
	"studio-engine/internal/netcfg"
	"studio-engine/internal/project/service"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/testutil"
	"studio-engine/internal/updates"
)

// harness is the Go side of http_gen.Session: the area wired to real services whose
// outside world (license server, update server, Windows) is faked.
type harness struct {
	t        testing.TB
	golden   *golden
	root     string
	clock    float64
	ticks    time.Duration
	licenses int
	license  *licenseProxy
	client   *fakeLicenseClient
	server   *fakeServer
	windows  *fakeWindows
	runtime  *fakeRuntime
	projects *service.Service
	mux      *http.ServeMux
}

func newHarness(t testing.TB, g *golden) *harness {
	t.Setenv("STUDIO_LICENSE_CERT_PATH", "") // licensing.New exports it when enforcing
	root := testutil.TempDir(t)
	h := &harness{t: t, golden: g, root: root, license: &licenseProxy{}, client: &fakeLicenseClient{},
		server: &fakeServer{}, runtime: &fakeRuntime{state: "stopped"},
		windows: &fakeWindows{t: t, env: decode(t, g.NetworkEnv).(*pyjson.Object)}}
	for _, folder := range []string{"project", "net", "updates", "sysroot"} {
		if err := os.MkdirAll(filepath.Join(root, folder), 0o777); err != nil {
			t.Fatal(err)
		}
	}
	projects, err := service.New(service.Config{DataDir: filepath.Join(root, "project"), NewID: counterIDs()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = projects.Close() })
	if _, err := projects.RecoverStartup(); err != nil {
		t.Fatal(err)
	}
	h.projects = projects
	ids := counterIDs()
	projectNetwork := ProjectNetwork{Service: projects}
	network := &netcfg.NetworkService{Projects: projectNetwork, Directory: filepath.Join(root, "net"),
		Program: g.Program, Executable: `C:\Program Files\ScadaBench\ScadaBench.exe`,
		Journal: netcfg.NewJournal(filepath.Join(root, "net")), Source: netcfg.AdapterFunc(h.windows.adapters),
		Rules: h.windows.names("rules"), Elevator: netcfg.ElevatorFunc(h.windows.elevate),
		Probe: netcfg.ProberFunc(h.windows.probe), Firewall: h.windows.firewall, Clock: h.monotonic, NewID: ids}
	firewall := &netcfg.GatewayFirewall{Network: network, Program: g.Gateway, Rules: h.windows.names("gateway_rules"),
		Blocking: h.windows.blocking}
	server := g.ServerURL
	checks := updates.New(updates.Options{Directory: filepath.Join(root, "updates"),
		ServerURL: func() (string, error) { return server, nil }, Current: g.Current, Keys: h.keys(),
		Opener: h.server, Clock: func() float64 { return h.clock }})
	sysroot := filepath.Join(root, "sysroot")
	area := New(Deps{License: h.license, Updates: checks, Network: network, Firewall: firewall,
		Projects: projectNetwork, Runtime: h.runtime, AppVersion: g.AppVersion,
		Npcap: func() *pyjson.Object { return netcfg.NpcapStatus(sysroot) }})
	h.mux = http.NewServeMux()
	for _, route := range area.Routes() {
		h.mux.Handle(route.Method+" "+route.Pattern, route.Handler)
	}
	return h
}

// monotonic is time.monotonic for the adapter cache: 1000 s plus every "tick".
func (h *harness) monotonic() time.Time {
	return time.Unix(1000, 0).Add(h.ticks)
}

func (h *harness) keys() map[string][]byte {
	public, err := hex.DecodeString(h.golden.TestKey.Public)
	if err != nil {
		h.t.Fatal(err)
	}
	return map[string][]byte{h.golden.TestKey.Kid: public}
}

func (h *harness) licenseRoot() string {
	return filepath.Join(h.root, fmt.Sprintf("license%d", h.licenses))
}

// newLicense builds the license service; testServer is the test-only seam (a server on this PC).
func (h *harness) newLicense(enforce, testServer bool) {
	h.licenses++
	native := filepath.Join(h.licenseRoot(), "native")
	if err := os.MkdirAll(native, 0o777); err != nil {
		h.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(native, "studio_ied.exe"), []byte("worker"), 0o666); err != nil {
		h.t.Fatal(err)
	}
	hwid := h.golden.HWID
	h.license.Service = licensing.New(licensing.Options{Home: filepath.Join(h.licenseRoot(), "home"),
		DataDir: filepath.Join(h.licenseRoot(), "data"), Enforce: enforce,
		HWID: func() (string, error) { return hwid, nil }, Clock: func() float64 { return h.clock },
		Keys: h.keys(), NativeFolder: native, Debugger: func() bool { return false }, Logf: h.t.Logf,
		ClientFactory: func(string) licensing.Client { return h.client }, AllowTestServer: testServer,
		Nonce: func() string { return strings.Repeat("0123456789abcdef", 2) }, // gen_license.NONCE
		Parts: func() (*pyjson.Object, error) { return pyjson.NewObject(), nil }})
}

func (h *harness) write(path string, data []byte) {
	if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
		h.t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o666); err != nil {
		h.t.Fatal(err)
	}
}

// setup applies one non-request step exactly as http_gen.Session.setup does.
func (h *harness) setup(s step) {
	switch s.Op {
	case "now":
		h.clock = number(h.t, s.Value)
	case "license":
		h.newLicense(s.Enforce, s.TestServer)
	case "license_file":
		data := []byte(s.Data)
		if s.Hex != "" {
			data, _ = hex.DecodeString(s.Hex)
		}
		h.write(filepath.Join(h.licenseRoot(), filepath.FromSlash(s.Path)), data)
	case "client":
		h.client.answers = map[string][]any{}
		answers := decode(h.t, s.Answers).(*pyjson.Object)
		for _, name := range answers.Keys() {
			queue, _ := answers.Get(name)
			h.client.answers[name] = queue.([]any)
		}
	case "answers":
		h.server.queue = decode(h.t, s.Queue).([]any)
	case "update_file":
		h.write(filepath.Join(h.root, "updates", "update_check.json"), []byte(s.Data))
	case "runtime":
		h.runtime.state = s.State
	case "ieds":
		ieds := decode(h.t, s.Value)
		if _, err := h.projects.Update(func(draft *pyjson.Object) (*pyjson.Object, error) {
			draft.Set("ieds", ieds)
			return nil, nil
		}); err != nil {
			h.t.Fatal(err)
		}
	case "net":
		h.windows.env.Set(s.Key, decode(h.t, s.Value))
	case "tick":
		h.ticks += time.Duration(s.Seconds * float64(time.Second))
	default:
		h.t.Fatalf("unknown step %q", s.Op)
	}
}

// counterIDs is a deterministic uuid4().hex: 1, 2, 3... as 32 hex digits.
func counterIDs() func() string {
	count := 0
	return func() string {
		count++
		return fmt.Sprintf("%032x", count)
	}
}
