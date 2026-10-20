// Package service is the server side of the SNMP devices tool (ADR-0092): it keeps the
// project's SNMP section, starts studio_snmp.exe while SNMP is in use, and holds the state the
// page reads (device states, request and trap logs, the reader's table).
package service

import (
	"encoding/json"
	"fmt"

	"studio-engine/internal/pyjson"
)

// ToPy turns typed data into the pyjson values the API and worker frames use.
func ToPy(value any) (any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return pyjson.Decode(data)
}

// ToObject is ToPy for a value that encodes as a JSON object.
func ToObject(value any) (*pyjson.Object, error) {
	converted, err := ToPy(value)
	if err != nil {
		return nil, err
	}
	object, isObject := converted.(*pyjson.Object)
	if !isObject {
		return nil, fmt.Errorf("expected an object")
	}
	return object, nil
}

// FromPy fills target (a pointer) from a pyjson value.
func FromPy(value any, target any) error {
	data, err := pyjson.Compact(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
