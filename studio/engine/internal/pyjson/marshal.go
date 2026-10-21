package pyjson

// MarshalJSON lets typed structs that hold pyjson values (the Go-only SNMP tool, ADR-0092)
// go through encoding/json; the object is written compactly with its key order and number kinds.
func (o *Object) MarshalJSON() ([]byte, error) { return Compact(o) }
