package app

import (
	"context"
	"time"

	"studio-engine/internal/httpapi"
	"studio-engine/internal/licensing"
	"studio-engine/internal/store/recovery"
)

// gates returns the 402 license gate (the license service itself) and the startup wait
// (background recovery), unless a test replaced them.
func (e *Engine) gates() (httpapi.LicenseGate, httpapi.StartupGate) {
	var license httpapi.LicenseGate = licenseGate{e.License}
	var startup httpapi.StartupGate = startupGate{e.Recovery}
	if e.options.License != nil {
		license = e.options.License
	}
	if e.options.Startup != nil {
		startup = e.options.Startup
	}
	return license, startup
}

// startBackground is the end of create_app: license background verify, the daily update
// check and, once startup recovery succeeded, the gateway resume (_resume_gateway).
func (e *Engine) startBackground(ctx context.Context) {
	if e.options.NoBackground {
		close(e.resumed)
		return
	}
	e.License.StartBackgroundVerify()
	e.Updates.StartBackground(ctx)
	go e.resumeGateway(ctx)
}

// resumeGateway is _resume_gateway: a gateway or master that was running when the app
// closed starts again by itself; the master first, since a protocol converter serves what
// the master reads.
func (e *Engine) resumeGateway(ctx context.Context) {
	defer close(e.resumed)
	select {
	case <-e.Recovery.Done():
	case <-ctx.Done():
		return
	}
	if _, err := e.Recovery.Result(); err != nil || ctx.Err() != nil {
		return
	}
	e.Master.Autoconnect()
	if ctx.Err() == nil {
		e.Gateway.Autostart()
	}
}

// licenseGate is licensing.Service as httpapi.LicenseGate. Python's before_request let an
// exception of is_licensed() through (HTTP 500); the panic gives the same answer through
// httpapi's recover.
type licenseGate struct{ service *licensing.Service }

func (g licenseGate) Enforced() bool { return g.service.Enforced() }

func (g licenseGate) Licensed() bool {
	licensed, err := g.service.Licensed()
	if err != nil {
		panic(err)
	}
	return licensed
}

// startupGate is BackgroundRecovery as httpapi.StartupGate.
type startupGate struct{ background *recovery.Background }

func (g startupGate) Wait(timeout time.Duration) bool { return g.background.Wait(timeout) }

func (g startupGate) Err() error {
	_, err := g.background.Result()
	return err
}
