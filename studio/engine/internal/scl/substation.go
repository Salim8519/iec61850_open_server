package scl

import (
	"strings"

	"studio-engine/internal/scl/pyrules"
	"studio-engine/internal/scl/xmldoc"
)

// ESLD is the Transpower SSD single-line-diagram extension namespace.
const ESLD = "{https://transpower.co.nz/SCL/SSD/SLD/v0}"

// Grounded is the terminal path of an earthed terminal.
const Grounded = "grounded"

// Topology is substation.read_substation: equipment, connectivity nodes, LNodes and
// logical-device names, flat and drawing-neutral. Unsupported elements are listed.
type Topology struct {
	Substations []string
	Equipment   []*Equipment
	Nodes       map[string]*ConnectivityNode
	NodeOrder   []string // dict insertion order of Nodes
	Other       []OtherItem
	Domains     map[[2]string]string // (ied, ldInst) -> ldName or ied+inst
	DomainOrder [][2]string
	Diagnostics []string
}

// Equipment is one ConductingEquipment. Terminals hold node paths, Grounded or "" (None).
type Equipment struct {
	Path, Name, Type string
	Level, Bay       int
	BayName          string
	ESLD             *[2]float64
	LNodes           []LNode
	Terminals        []string
}

// ConnectivityNode is one declared node; BusY is nil when no busbar is drawn.
type ConnectivityNode struct {
	Path, Name string
	Level, Bay int
	BayName    string
	BusY       *float64
}

// OtherItem is an element listed but not drawn (transformers, general equipment).
type OtherItem struct{ Path, Element, Type string }

// LNode is the identity of one LNode written on equipment.
type LNode struct{ IEDName, LDInst, Prefix, LNClass, LNInst string }

// ReadSubstation is read_substation(source).
func ReadSubstation(source string) (*Topology, error) {
	parsed, err := ParseDocument([]byte(source), false)
	if err != nil {
		return nil, err
	}
	root := parsed.Root
	result := &Topology{Nodes: map[string]*ConnectivityNode{}, Domains: map[[2]string]string{}}
	result.readDomains(root)
	for _, substation := range root.FindAll("Substation") {
		name := substation.GetOr("name", "")
		result.Substations = append(result.Substations, name)
		for _, item := range substation.FindAll("PowerTransformer") {
			result.other(item, []string{name})
		}
		for levelIndex, level := range substation.FindAll("VoltageLevel") {
			result.voltageLevel(level, name, levelIndex)
		}
	}
	for _, equipment := range result.Equipment {
		for index, path := range equipment.Terminals {
			equipment.Terminals[index] = result.resolve(equipment, path)
		}
	}
	return result, nil
}

func (t *Topology) voltageLevel(level *xmldoc.Element, substation string, levelIndex int) {
	names := []string{substation, level.GetOr("name", "")}
	for _, item := range level.FindAll("PowerTransformer") {
		t.other(item, names)
	}
	for bayIndex, bay := range level.FindAll("Bay") {
		bayNames := append(append([]string(nil), names...), bay.GetOr("name", ""))
		bayName := bayNames[len(bayNames)-1]
		for _, node := range bay.FindAll("ConnectivityNode") {
			path := node.GetOr("pathName", "")
			if path == "" {
				path = strings.Join(append(append([]string(nil), bayNames...), node.GetOr("name", "")), "/")
			}
			if _, seen := t.Nodes[path]; !seen {
				t.NodeOrder = append(t.NodeOrder, path)
			}
			t.Nodes[path] = &ConnectivityNode{Path: path, Name: node.GetOr("name", ""), Level: levelIndex,
				Bay: bayIndex, BayName: bayName, BusY: busY(node)}
		}
		for _, item := range bay.FindAll("ConductingEquipment") {
			equipment := &Equipment{Path: strings.Join(append(append([]string(nil), bayNames...), item.GetOr("name", "")), "/"),
				Name: item.GetOr("name", ""), Type: item.GetOr("type", ""), Level: levelIndex, Bay: bayIndex,
				BayName: bayName, ESLD: position(item)}
			for _, terminal := range item.FindAll("Terminal") {
				equipment.Terminals = append(equipment.Terminals, terminalPath(terminal))
			}
			for _, node := range item.FindAll("LNode") {
				equipment.LNodes = append(equipment.LNodes, LNode{node.GetOr("iedName", ""), node.GetOr("ldInst", ""),
					node.GetOr("prefix", ""), node.GetOr("lnClass", ""), node.GetOr("lnInst", "")})
			}
			t.Equipment = append(t.Equipment, equipment)
		}
		for _, item := range append(bay.FindAll("PowerTransformer"), bay.FindAll("GeneralEquipment")...) {
			t.other(item, bayNames)
		}
	}
}

func (t *Topology) other(item *xmldoc.Element, names []string) {
	path := strings.Join(append(append([]string(nil), names...), item.GetOr("name", "")), "/")
	t.Other = append(t.Other, OtherItem{Path: path, Element: item.Tag, Type: item.GetOr("type", "")})
}

// terminalPath is _terminal_path ("" for None).
func terminalPath(terminal *xmldoc.Element) string {
	if terminal.GetOr("cNodeName", "\x00") == Grounded || strings.HasSuffix(terminal.GetOr("connectivityNode", ""), "/"+Grounded) {
		return Grounded
	}
	if path := terminal.GetOr("connectivityNode", ""); path != "" {
		return path
	}
	parts := make([]string, 0, 4)
	for _, key := range []string{"substationName", "voltageLevelName", "bayName", "cNodeName"} {
		value := terminal.GetOr(key, "")
		if value == "" {
			return ""
		}
		parts = append(parts, value)
	}
	return strings.Join(parts, "/")
}

func (t *Topology) resolve(equipment *Equipment, path string) string {
	if _, declared := t.Nodes[path]; path == Grounded || path != "" && declared {
		return path
	}
	shown := path
	if shown == "" {
		shown = "(missing)"
	}
	t.Diagnostics = append(t.Diagnostics, equipment.Path+": terminal points to connectivity node "+shown+
		" which is not declared; no wire is drawn for it")
	return ""
}

// attrNumber is _number: float(value), nil when absent or unreadable.
func attrNumber(element *xmldoc.Element, key string) *float64 {
	text, present := element.Get(key)
	if !present {
		return nil
	}
	value, err := pyrules.Float(text)
	if err != nil {
		return nil
	}
	return &value
}

func position(item *xmldoc.Element) *[2]float64 {
	x, y := attrNumber(item, ESLD+"x"), attrNumber(item, ESLD+"y")
	if x == nil || y == nil {
		return nil
	}
	return &[2]float64{*x, *y}
}

// busY is _bus_y: the lowest esld y of the vertices of bus="true" sections, or nil.
func busY(node *xmldoc.Element) *float64 {
	var heights []float64
	iterTag(node, "Section", func(section *xmldoc.Element) {
		if section.GetOr("bus", "") != "true" {
			return
		}
		iterTag(section, "Vertex", func(vertex *xmldoc.Element) {
			if value := attrNumber(vertex, ESLD+"y"); value != nil {
				heights = append(heights, *value)
			}
		})
	})
	if len(heights) == 0 {
		return nil
	}
	lowest := heights[0] // min(): an item replaces the current one only when smaller
	for _, value := range heights[1:] {
		if value < lowest {
			lowest = value
		}
	}
	return &lowest
}

func (t *Topology) readDomains(root *xmldoc.Element) {
	for _, ied := range root.FindAll("IED") {
		name := ied.GetOr("name", "")
		for _, device := range stationDevices(root, ied) {
			inst := device.GetOr("inst", "")
			domain := device.GetOr("ldName", "")
			if domain == "" {
				domain = name + inst
			}
			key := [2]string{name, inst}
			if _, seen := t.Domains[key]; !seen {
				t.DomainOrder = append(t.DomainOrder, key)
			}
			t.Domains[key] = domain
		}
	}
}
