package mib

import "strings"

// Lookup finds the node with the longest known prefix of a dotted OID. instance is the
// rest without a leading dot ("" for an exact match). node is nil when nothing matches.
func (l *Library) Lookup(oid string) (node *Node, instance string) {
	oid = strings.TrimPrefix(oid, ".")
	if !numericOID(oid) {
		return nil, ""
	}
	idx := l.current()
	for p := oid; p != ""; p = parentOID(p) {
		if n := idx.nodes[p]; n != nil {
			return n, strings.TrimPrefix(oid[len(p):], ".")
		}
	}
	return nil, ""
}

// Name returns "MODULE::name.instance" for an OID, or the dotted OID when it is unknown.
func (l *Library) Name(oid string) string {
	n, inst := l.Lookup(oid)
	if n == nil {
		return strings.TrimPrefix(oid, ".")
	}
	s := n.Name
	if n.Module != "" {
		s = n.Module + "::" + s
	}
	if inst != "" {
		s += "." + inst
	}
	return s
}

// Resolve turns a name into a dotted numeric OID. It accepts "IF-MIB::ifOperStatus.3",
// "ifOperStatus.3", "ifOperStatus", ".1.3.6.1...", "1.3.6.1..." and "iso.3.6.1...".
// Quoted instance parts ("public" or 'public') become their byte arcs like Net-SNMP.
func (l *Library) Resolve(name string) (string, bool) {
	name = strings.TrimPrefix(strings.TrimSpace(name), ".")
	if name == "" {
		return "", false
	}
	if numericOID(name) {
		return name, true
	}
	mod, rest, qualified := strings.Cut(name, "::")
	if !qualified {
		mod, rest = "", name
	}
	label, inst := splitName(rest)
	base, ok := l.resolveLabel(mod, label)
	if !ok {
		return "", false
	}
	if inst == "" {
		return base, true
	}
	arcs, ok := instanceArcs(inst)
	if !ok {
		return l.resolvePath(mod, inst)
	}
	return base + "." + arcs, true
}

// resolvePath handles full symbolic paths (snmpwalk -Of, ".iso.org.dod.internet.mgmt.
// mib-2.system.sysDescr.0") by resolving the last name in the path that is followed only
// by an instance.
func (l *Library) resolvePath(mod, path string) (string, bool) {
	var quote byte
	for i := 0; i < len(path); i++ {
		switch c := path[i]; {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == 0x27: // double or single quote
			quote = c
		case c == '.' && i+1 < len(path) && isLetter(path[i+1]):
			label, inst := splitName(path[i+1:])
			if _, ok := instanceArcs(inst); ok || inst == "" {
				if base, ok := l.resolveLabel(mod, label); ok {
					if inst == "" {
						return base, true
					}
					arcs, _ := instanceArcs(inst)
					return base + "." + arcs, true
				}
			}
		}
	}
	return "", false
}

// resolveLabel finds a single name, in one module when mod is set. A module that is not
// loaded falls back to the name alone, so "SNMPv2-SMI::enterprises" works without it.
func (l *Library) resolveLabel(mod, label string) (string, bool) {
	idx := l.current()
	if mod != "" {
		if names, loaded := idx.byModule[mod]; loaded {
			oid, ok := names[label]
			return oid, ok
		}
		if _, loaded := idx.info[mod]; loaded {
			return "", false
		}
	}
	if oid, ok := idx.byName[label]; ok {
		return oid, true
	}
	return "", false
}

// Children returns the known nodes directly below an OID (the nearest defined
// descendants), sorted by OID, for a tree browser.
func (l *Library) Children(oid string) []*Node {
	oid = strings.TrimPrefix(oid, ".")
	list := l.current().children[oid]
	return append([]*Node(nil), list...)
}

// Roots returns the top-level known nodes (ccitt, iso, joint-iso-ccitt).
func (l *Library) Roots() []*Node {
	idx := l.current()
	var out []*Node
	for _, oid := range []string{"0", "1", "2"} {
		if n := idx.nodes[oid]; n != nil {
			out = append(out, n)
		}
	}
	return out
}

// EnumText returns the label of a named number of the object an OID belongs to, or "".
func (l *Library) EnumText(oid string, value int64) string {
	n, _ := l.Lookup(oid)
	if n == nil {
		return ""
	}
	return n.Enums[value]
}
