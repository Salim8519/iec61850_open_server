package app

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"studio-engine/internal/acquisition"
	"studio-engine/internal/api"
	"studio-engine/internal/events"
	"studio-engine/internal/gateway/service"
	"studio-engine/internal/httpapi"
	"studio-engine/internal/licensing"
	projects "studio-engine/internal/project/service"
	"studio-engine/internal/runtime"
	"studio-engine/internal/store/recovery"
	"studio-engine/internal/store/xsd"
	"studio-engine/internal/tools/control"
	"studio-engine/internal/tools/explorer"
	"studio-engine/internal/tools/scan"
	"studio-engine/internal/updates"
)

// Options are what create_app received (Settings, token, secret, admin origin) plus the
// Go engine's own paths. The fields marked "tests" replace a real service with a fake.
type Options struct {
	DataDir     string
	NativeDir   string // folder of studio_ied.exe, studio_live.exe, studio_control.exe, ...
	FrontendDir string // dist folder or frontend.pak; empty: no frontend build
	SchemasDir  string // bundled XSD folders (SCL2003, SCL2007B, SCL2007B4)
	// XSDHelper is studio_xsd.exe; empty means <NativeDir>/studio_xsd.exe.
	XSDHelper string
	// XSDSeed lists verdict files (Python's lxml verdicts) imported into the verdict cache
	// at start; it stands in for studio_xsd.exe until that helper is built (dev only).
	XSDSeed     []string
	Token       string
	SecretKey   string
	AdminOrigin string
	Version     string // studio_app.version.__version__
	Enforce     bool   // license enforcement: only the packaged app (sys.frozen in Python)
	LicenseHome string // "" means licensing.DefaultHome()
	// CommandKeep keeps a canvas command's MMS association open (0: like studio_probe).
	CommandKeep time.Duration
	// AppExecutable runs the elevated network helper (os.Executable() when empty).
	AppExecutable string
	Logf          func(format string, args ...any) // nil means log.Printf

	// LicenseService and UpdateService replace the default services (the parity build's
	// data seams, create_app's license_service/update_service arguments).
	LicenseService *licensing.Service
	UpdateService  *updates.Service
	License        httpapi.LicenseGate // tests: replaces the license service's gate
	Startup      httpapi.StartupGate // tests: replaces the startup recovery gate
	NoBackground bool                // tests: no update/license loops, no gateway resume
}

// Engine is the composed Go server: every service of create_app and its HTTP handler.
type Engine struct {
	*App
	Projects *projects.Service
	Runtime  *runtime.Coordinator
	Live     *acquisition.Manager
	Gateway  *service.Manager
	Master   *service.Master
	Explorer *explorer.Service
	Scanner  *scan.Service
	Control  *control.Client
	License  *licensing.Service
	Updates  *updates.Service
	Recovery *recovery.Background
	Routes   []api.Route // every route the engine serves (TestEveryPythonRouteServed)

	options   Options
	bus       *events.Bus
	validator *xsd.Validator
	cancel    context.CancelFunc
	resumed   chan struct{} // closed when the gateway resume goroutine ended
	closeOnce sync.Once
}

func (o *Options) defaults() {
	if o.Logf == nil {
		o.Logf = log.Printf
	}
	if o.XSDHelper == "" {
		o.XSDHelper = filepath.Join(o.NativeDir, "studio_xsd.exe")
	}
}

func (o Options) native(name string) string { return filepath.Join(o.NativeDir, name) }

// Compose builds the engine in create_app's order: journal fold, project service (its
// changes become project_changed), background startup recovery, coordinator, live
// acquisition, tools, platform, routes. Nothing is served until the caller mounts Handler;
// the gateway and master resume only after recovery succeeded.
func Compose(options Options) (*Engine, error) {
	options.defaults()
	e := &Engine{options: options, resumed: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	if err := e.wireProject(); err != nil {
		cancel()
		e.closeValidator()
		return nil, err
	}
	live := e.wireRuntime(e.bus)
	routes := e.wireLive(live)
	routes = append(routes, e.wireTools()...)
	platform, err := e.wirePlatform()
	if err != nil {
		e.Close()
		return nil, err
	}
	routes = append(routes, platform...)
	routes = append(routes, e.runtimeRoutes()...)
	routes = append(routes, e.projectRoutes()...)
	license, startup := e.gates()
	composed, err := New(Config{DataDir: options.DataDir, FrontendDir: options.FrontendDir, Token: options.Token,
		SecretKey: options.SecretKey, AdminOrigin: options.AdminOrigin, License: license, Startup: startup,
		Areas: []api.Area{routeList(routes)}, Bus: e.bus})
	if err != nil {
		e.Close()
		return nil, fmt.Errorf("compose routes: %w", err)
	}
	e.App, e.Routes = composed, routes
	e.startBackground(ctx)
	return e, nil
}

// routeList serves a fixed list of routes as one area.
type routeList []api.Route

func (r routeList) Routes() []api.Route { return r }
