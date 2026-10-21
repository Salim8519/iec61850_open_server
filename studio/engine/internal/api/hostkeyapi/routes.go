// Package hostkeyapi serves the Go-only routes of the VM host key (ADR-0099): the Settings switch
// "Let my virtual machines use this license", its one Windows Firewall rule and the offline
// activation code. They live under /api/v1/license so the license gate (402) never blocks them.
// The frozen Python engine answers them 404.
package hostkeyapi

import (
	"errors"
	"net/http"
	"strings"

	"studio-engine/internal/api"
	"studio-engine/internal/apperr"
	"studio-engine/internal/httpapi"
	"studio-engine/internal/licensing"
	"studio-engine/internal/licensing/hostkey"
	"studio-engine/internal/pyjson"
)

// Paths of the area.
const (
	Prefix   = "/api/v1/license/host-key"
	CodePath = "/api/v1/license/activation-code"
)

// License is what the routes need from the license service (*licensing.Service).
type License interface {
	HostKeyInfo() licensing.HostKeyInfo
	ActivationCode() (string, error)
}

// Sharing is the Run value and helper process (hostkey.Sharing).
type Sharing interface {
	State() hostkey.State
	Enable() error
	Disable() error
}

// Firewall is the previewed firewall rule (*netcfg.HostKeyFirewall).
type Firewall interface {
	Preview() (*pyjson.Object, error)
	Apply(fingerprint, confirm any) (*pyjson.Object, error)
}

// Deps are the services behind the routes.
type Deps struct {
	License  License
	Sharing  Sharing
	Firewall Firewall
}

// Area is the route set.
type Area struct{ deps Deps }

// New returns the area for deps.
func New(deps Deps) *Area { return &Area{deps: deps} }

// GoOnly reports whether a route key ("GET /api/v1/...") belongs to this area.
func GoOnly(key string) bool {
	_, pattern, _ := strings.Cut(key, " ")
	return pattern == CodePath || pattern == Prefix || strings.HasPrefix(pattern, Prefix+"/")
}

// Routes lists the five routes.
func (a *Area) Routes() []api.Route {
	return []api.Route{
		{Method: "GET", Pattern: Prefix, Handler: handle(a.status)},
		{Method: "PUT", Pattern: Prefix, Handler: handle(a.set)},
		{Method: "GET", Pattern: Prefix + "/firewall", Handler: handle(func(*http.Request) (any, error) {
			return a.deps.Firewall.Preview()
		})},
		{Method: "POST", Pattern: Prefix + "/firewall", Handler: handle(a.firewallApply)},
		{Method: "GET", Pattern: CodePath, Handler: handle(a.code)},
	}
}

func handle(fn func(*http.Request) (any, error)) http.Handler {
	return httpapi.Handle(func(w http.ResponseWriter, r *http.Request) error {
		value, err := fn(r)
		if err != nil {
			return err
		}
		httpapi.WriteJSON(w, http.StatusOK, value)
		return nil
	})
}

// bodyOf is the request's JSON object, or an empty one for anything else.
func bodyOf(value any) *pyjson.Object {
	if object, ok := value.(*pyjson.Object); ok {
		return object
	}
	return pyjson.NewObject()
}

func orNil(text string) any {
	if text == "" {
		return nil
	}
	return text
}

// status is GET /license/host-key: whether sharing is possible (and if not, why), whether it is
// on, and whether the helper runs.
func (a *Area) status(*http.Request) (any, error) {
	info, state := a.deps.License.HostKeyInfo(), a.deps.Sharing.State()
	return pyjson.NewObject().Set("available", info.Available).Set("reason", orNil(info.Reason)).
		Set("key", orNil(info.Key)).Set("enabled", state.Enabled).Set("running", state.Running).
		Set("helper_found", state.HelperFound).Set("port", int64(hostkey.Port)), nil
}

// set is PUT /license/host-key {"enabled": bool}: turning it off is always allowed, turning it on
// only for a license this PC can share.
func (a *Area) set(r *http.Request) (any, error) {
	value, err := httpapi.ReadJSON(r, true)
	if err != nil {
		return nil, err
	}
	enabled, _ := bodyOf(value).Get("enabled")
	flag, isFlag := enabled.(bool)
	if !isFlag {
		return nil, apperr.New("Say whether virtual machines may use this license (enabled: true or false).")
	}
	if !flag {
		if err := a.deps.Sharing.Disable(); err != nil {
			return nil, apperr.New("Sharing could not be switched off: " + err.Error())
		}
		return a.status(r)
	}
	if info := a.deps.License.HostKeyInfo(); !info.Available {
		return nil, apperr.New(info.Reason)
	}
	if err := a.deps.Sharing.Enable(); err != nil {
		return nil, apperr.New("Sharing could not be switched on: " + err.Error())
	}
	return a.status(r)
}

// firewallApply is POST /license/host-key/firewall {"fingerprint", "confirm"}.
func (a *Area) firewallApply(r *http.Request) (any, error) {
	value, err := httpapi.ReadJSON(r, true)
	if err != nil {
		return nil, err
	}
	body := bodyOf(value)
	fingerprint, _ := body.Get("fingerprint")
	confirm, _ := body.Get("confirm")
	return a.deps.Firewall.Apply(fingerprint, confirm)
}

// code is GET /license/activation-code: "<machine ID>.<host key>" for offline activation.
func (a *Area) code(*http.Request) (any, error) {
	code, err := a.deps.License.ActivationCode()
	var action *licensing.ActionError
	if errors.As(err, &action) {
		return nil, apperr.Coded(action.Message, action.Code)
	}
	if err != nil {
		return nil, err
	}
	return pyjson.NewObject().Set("code", code), nil
}
