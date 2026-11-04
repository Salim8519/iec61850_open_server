// Package addressapi serves the Go-only routes that set one simulated IED's or SNMP device's
// network address from the Devices page and the SNMP device editor (change 0215). The frozen
// Python engine answers them 404.
package addressapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"studio-engine/internal/api"
	"studio-engine/internal/apperr"
	"studio-engine/internal/httpapi"
	"studio-engine/internal/pyjson"
)

// Prefix is the area's path.
const Prefix = "/api/v1/network/address"

// Addresses is the address change (*netaddr.Service).
type Addresses interface {
	SetIED(id, ip string) (*pyjson.Object, error)
	SetSNMP(id, ip string) (*pyjson.Object, error)
}

// GoOnly reports whether a route key ("PUT /api/v1/...") belongs to this area.
func GoOnly(key string) bool {
	_, pattern, _ := strings.Cut(key, " ")
	return strings.HasPrefix(pattern, Prefix+"/")
}

// Routes lists the two routes: PUT .../ied/{id} and PUT .../snmp/{id} with {"ip": "..."}.
func Routes(addresses Addresses) []api.Route {
	set := func(change func(id, ip string) (*pyjson.Object, error)) http.Handler {
		return httpapi.Handle(func(w http.ResponseWriter, r *http.Request) error {
			ip, err := read(r)
			if err != nil {
				return err
			}
			result, err := change(r.PathValue("id"), ip)
			if err != nil {
				return err
			}
			httpapi.WriteJSON(w, http.StatusOK, result)
			return nil
		})
	}
	return []api.Route{
		{Method: "PUT", Pattern: Prefix + "/ied/{id}", Handler: set(addresses.SetIED)},
		{Method: "PUT", Pattern: Prefix + "/snmp/{id}", Handler: set(addresses.SetSNMP)},
	}
}

func read(r *http.Request) (string, error) {
	var body struct {
		IP string `json:"ip"`
	}
	if r.Body != nil {
		decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<16))
		if err := decoder.Decode(&body); err != nil && !errors.Is(err, io.EOF) {
			return "", apperr.New("The request is not valid JSON for an address.")
		}
	}
	if strings.TrimSpace(body.IP) == "" {
		return "", apperr.New("Type the new address, for example 192.168.1.21.")
	}
	return body.IP, nil
}
