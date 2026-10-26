// Package labapi serves the Go-only routes of the "This PC lab network" (ADR-0120) under
// /api/v1/network/lab. The frozen Python engine answers them 404.
package labapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"studio-engine/internal/api"
	"studio-engine/internal/apperr"
	"studio-engine/internal/httpapi"
	"studio-engine/internal/labnet"
	"studio-engine/internal/labsetup"
	"studio-engine/internal/pyjson"
)

// Prefix is the area's path.
const Prefix = "/api/v1/network/lab"

// Lab is the page's logic (*labsetup.Service).
type Lab interface {
	Status(chosen map[string]string) (labsetup.Status, error)
	Install() (labsetup.Status, error)
	Use(chosen map[string]string) (labsetup.Status, error)
	SetAddress(id, address string) (labsetup.Status, error)
	Ping(address string) (labnet.Pinged, error)
	Remove(leave func() error) (labsetup.Status, error)
}

// Deps are the services behind the routes; Leave switches the project back to "This PC only".
type Deps struct {
	Lab   Lab
	Leave func() error
}

// Area is the route set.
type Area struct{ deps Deps }

// New returns the area for deps.
func New(deps Deps) *Area { return &Area{deps: deps} }

// GoOnly reports whether a route key ("GET /api/v1/...") belongs to this area.
func GoOnly(key string) bool {
	_, pattern, _ := strings.Cut(key, " ")
	return pattern == Prefix || strings.HasPrefix(pattern, Prefix+"/")
}

// Routes lists the seven routes.
func (a *Area) Routes() []api.Route {
	return []api.Route{
		{Method: "GET", Pattern: Prefix, Handler: handle(func(*http.Request) (any, error) {
			return a.deps.Lab.Status(nil)
		}), Polled: true},
		{Method: "POST", Pattern: Prefix + "/check", Handler: handle(func(r *http.Request) (any, error) {
			body, err := read(r)
			if err != nil {
				return nil, err
			}
			return a.deps.Lab.Status(body.Addresses)
		}), Polled: true},
		{Method: "POST", Pattern: Prefix + "/install", Handler: handle(func(*http.Request) (any, error) {
			return a.deps.Lab.Install()
		})},
		{Method: "POST", Pattern: Prefix + "/use", Handler: handle(func(r *http.Request) (any, error) {
			body, err := read(r)
			if err != nil {
				return nil, err
			}
			return a.deps.Lab.Use(body.Addresses)
		})},
		{Method: "PUT", Pattern: Prefix + "/address", Handler: handle(func(r *http.Request) (any, error) {
			body, err := read(r)
			if err != nil {
				return nil, err
			}
			if body.IEDID == "" {
				return nil, apperr.New("Say which IED moves (ied_id).")
			}
			return a.deps.Lab.SetAddress(body.IEDID, body.IP)
		})},
		{Method: "POST", Pattern: Prefix + "/ping", Handler: handle(func(r *http.Request) (any, error) {
			body, err := read(r)
			if err != nil {
				return nil, err
			}
			return a.deps.Lab.Ping(body.IP)
		})},
		{Method: "POST", Pattern: Prefix + "/remove", Handler: handle(func(*http.Request) (any, error) {
			return a.deps.Lab.Remove(a.deps.Leave)
		})},
	}
}

// body is every field a lab request may carry.
type body struct {
	Addresses map[string]string `json:"addresses"`
	IEDID     string            `json:"ied_id"`
	IP        string            `json:"ip"`
}

func read(r *http.Request) (body, error) {
	var parsed body
	if r.Body == nil {
		return parsed, nil
	}
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	if err := decoder.Decode(&parsed); err != nil && !errors.Is(err, io.EOF) {
		return parsed, apperr.New("The request is not valid JSON for the lab network.")
	}
	return parsed, nil
}

func handle(fn func(*http.Request) (any, error)) http.Handler {
	return httpapi.Handle(func(w http.ResponseWriter, r *http.Request) error {
		value, err := fn(r)
		if err != nil {
			return err
		}
		converted, err := toPy(value)
		if err != nil {
			return err
		}
		httpapi.WriteJSON(w, http.StatusOK, converted)
		return nil
	})
}

// toPy turns a Go value into the pyjson values WriteJSON encodes.
func toPy(value any) (any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return pyjson.Decode(data)
}
