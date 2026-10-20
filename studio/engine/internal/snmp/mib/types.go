package mib

import "studio-engine/internal/snmp/model"

// baseTypes maps SMI base types (SMIv1 and SMIv2 names) to point types. They are checked
// before any type assignment, because SNMPv2-SMI itself defines them with ASN.1 tags.
var baseTypes = map[string]string{
	"INTEGER": model.TypeInteger, "Integer32": model.TypeInteger,
	"OCTET STRING": model.TypeOctets, "BITS": model.TypeOctets,
	"OBJECT IDENTIFIER": model.TypeOID,
	"IpAddress":         model.TypeIPAddress, "NetworkAddress": model.TypeIPAddress,
	"Counter": model.TypeCounter32, "Counter32": model.TypeCounter32,
	"Gauge": model.TypeGauge32, "Gauge32": model.TypeGauge32, "Unsigned32": model.TypeGauge32,
	"TimeTicks": model.TypeTimeTicks, "Counter64": model.TypeCounter64, "Opaque": model.TypeOpaque,
}

// sequenceType marks a SYNTAX that resolves to a SEQUENCE (a table row).
const sequenceType = "SEQUENCE"

// typeInfo is a resolved SYNTAX.
type typeInfo struct {
	base  string
	hex   bool
	enums map[int64]string
}

// typeOf resolves a SYNTAX through textual conventions to its base type. ok is false when
// a type name is not defined in any loaded module.
func (r *resolver) typeOf(m *module, s syntaxRef, depth int) (typeInfo, bool) {
	if b, ok := baseTypes[s.name]; ok {
		hex := s.name == "OCTET STRING" || s.name == "BITS" || b == model.TypeOpaque
		return typeInfo{base: b, hex: hex, enums: s.enums}, true
	}
	if s.name == "SEQUENCE" {
		return typeInfo{base: sequenceType}, true
	}
	td, owner := r.findType(m, s.name)
	if td == nil || depth > 16 {
		return typeInfo{enums: s.enums}, false
	}
	info, ok := r.typeOf(owner, td.syntax, depth+1)
	if info.base == model.TypeOctets {
		if td.hint != "" {
			info.hex = !textHint(td.hint)
		}
		if s.name == "DisplayString" {
			info.hex = false
		}
	}
	if len(s.enums) > 0 {
		info.enums = s.enums
	}
	return info, ok
}

// findType finds the module that defines a type name as seen from module m.
func (r *resolver) findType(m *module, name string) (*typeDef, *module) {
	if td := m.types[name]; td != nil {
		return td, m
	}
	if from, ok := m.imports[name]; ok {
		if fm := r.mods[from]; fm != nil && fm.types[name] != nil {
			return fm.types[name], fm
		}
	}
	for _, om := range r.typeOwners[name] {
		return om.types[name], om
	}
	return nil, nil
}

// textHint reports whether a DISPLAY-HINT shows the whole value as text ("255a", "255t").
func textHint(hint string) bool {
	i := 0
	for i < len(hint) && isDigit(hint[i]) {
		i++
	}
	return hint[i:] == "a" || hint[i:] == "t"
}

// fromMissing reports whether a name is imported from a module that is not loaded.
func (r *resolver) fromMissing(m *module, name string) bool {
	from, ok := m.imports[name]
	return ok && r.mods[from] == nil
}
