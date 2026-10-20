// Package snmpapi serves the Go-only routes of the SNMP devices tool (ADR-0092) under
// /api/v1/tools/snmp. The frozen Python engine answers them 404.
package snmpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"studio-engine/internal/api"
	"studio-engine/internal/apperr"
	"studio-engine/internal/httpapi"
	"studio-engine/internal/snmp/service"
)

// Prefix starts every route of this area; routes_test treats them as Go-only.
const Prefix = "/api/v1/tools/snmp"

// maxBody fits a 50 MB walk file sent as JSON text.
const maxBody = 64 << 20

// Area is the SNMP tool's routes.
type Area struct{ snmp *service.Service }

// New returns the area for the service.
func New(snmp *service.Service) *Area { return &Area{snmp: snmp} }

// GoOnly reports whether a route key ("GET /api/v1/...") belongs to this area.
func GoOnly(key string) bool {
	_, pattern, _ := strings.Cut(key, " ")
	return pattern == Prefix || strings.HasPrefix(pattern, Prefix+"/")
}

func route(method, path string, polled bool, fn func(*http.Request) (any, error)) api.Route {
	return api.Route{Method: method, Pattern: Prefix + path, Handler: handle(fn), Polled: polled}
}

// Routes lists every route of the tool.
func (a *Area) Routes() []api.Route {
	routes := a.toolRoutes()
	routes = append(routes, a.deviceRoutes()...)
	routes = append(routes, a.readerRoutes()...)
	return append(routes, a.mibRoutes()...)
}

// download is a text file answer (CSV export).
type download struct {
	body     string
	filename string
}

func handle(fn func(*http.Request) (any, error)) http.Handler {
	return httpapi.Handle(func(w http.ResponseWriter, r *http.Request) error {
		value, err := fn(r)
		if err != nil {
			return userError(err)
		}
		if file, isFile := value.(*download); isFile {
			w.Header().Set("Content-Disposition", httpapi.ContentDisposition("attachment", file.filename))
			httpapi.WriteBody(w, http.StatusOK, "text/csv; charset=utf-8", []byte(file.body))
			return nil
		}
		converted, err := service.ToPy(value)
		if err != nil {
			return err
		}
		httpapi.WriteJSON(w, http.StatusOK, converted)
		return nil
	})
}

// userError keeps app errors and words anything else as a 400 the page shows.
func userError(err error) error {
	var known *apperr.Error
	var exception *httpapi.HTTPError
	if errors.As(err, &known) || errors.As(err, &exception) {
		return err
	}
	return apperr.ValueError(err.Error())
}

// decode reads a JSON body into target.
func decode(r *http.Request, target any) error {
	data, err := io.ReadAll(io.LimitReader(r.Body, maxBody+1))
	if err != nil {
		return httpapi.ErrBadRequest
	}
	if len(data) > maxBody {
		return httpapi.ErrTooLarge
	}
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, target); err != nil {
		return apperr.ValueError("The request was not understood: " + err.Error())
	}
	return nil
}

func (a *Area) toolRoutes() []api.Route {
	s := a.snmp
	return []api.Route{
		route("GET", "", true, func(*http.Request) (any, error) { return s.Status(), nil }),
		route("GET", "/config", false, func(*http.Request) (any, error) { return s.Config(), nil }),
		route("POST", "/role", false, func(r *http.Request) (any, error) {
			var body struct {
				Role string `json:"role"`
			}
			if err := decode(r, &body); err != nil {
				return nil, err
			}
			return s.SetRole(body.Role)
		}),
		route("POST", "/start", false, func(*http.Request) (any, error) { return s.Start() }),
		route("POST", "/stop", false, func(*http.Request) (any, error) {
			if err := s.Stop(); err != nil {
				return nil, err
			}
			return s.Status(), nil
		}),
		route("GET", "/requests", true, func(r *http.Request) (any, error) {
			return s.Requests.After(after(r), 500), nil
		}),
		route("GET", "/traps", true, func(r *http.Request) (any, error) { return s.Traps.After(after(r), 500), nil }),
	}
}

// after is the ?after= cursor of a log page.
func after(r *http.Request) int64 {
	var value int64
	if err := json.Unmarshal([]byte(r.URL.Query().Get("after")), &value); err != nil {
		return 0
	}
	return value
}
