package addpoints

import (
	"strings"

	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
)

// IED is the part of a project IED the planner reads: its signals and controllable objects
// from the SCL, its name and role (simulated, real or off).
type IED struct {
	ID, Name, Role string
	signals        []*pyjson.Object
	byRef          map[string]*pyjson.Object
	controls       []*pyjson.Object
}

// NewIED reads one project IED; malformed parts are skipped (they offer nothing to add).
func NewIED(raw *pyjson.Object) *IED {
	ied := &IED{ID: text(raw, "id"), Name: text(raw, "name"), Role: text(raw, "role"), byRef: map[string]*pyjson.Object{}}
	if ied.Role == "" {
		ied.Role = "simulated"
	}
	for _, item := range list(raw, "signals") {
		if signal, ok := item.(*pyjson.Object); ok && text(signal, "ref") != "" {
			ied.signals = append(ied.signals, signal)
			ied.byRef[text(signal, "ref")] = signal
		}
	}
	for _, item := range append(list(raw, "controls"), gateway.TypedControlsOf(raw)...) {
		if control, ok := item.(*pyjson.Object); ok && text(control, "ref") != "" {
			ied.controls = append(ied.controls, control) // typed controls follow SPC/DPC (ADR-0104)
		}
	}
	return ied
}

// Index is every project IED by id, in project order.
type Index struct {
	List []*IED
	byID map[string]*IED
}

// NewIndex reads the project IEDs.
func NewIndex(ieds []*pyjson.Object) *Index {
	index := &Index{byID: map[string]*IED{}}
	for _, raw := range ieds {
		ied := NewIED(raw)
		index.List = append(index.List, ied)
		index.byID[ied.ID] = ied
	}
	return index
}

// ByID is the IED with this id, or nil.
func (index *Index) ByID(id string) *IED { return index.byID[id] }

// Find is the IED a person wrote in a sheet: its name (any case) or its id.
func (index *Index) Find(written string) *IED {
	want := strings.TrimSpace(written)
	if ied := index.byID[want]; ied != nil {
		return ied
	}
	for _, ied := range index.List {
		if strings.EqualFold(ied.Name, want) {
			return ied
		}
	}
	return nil
}

// Control is the controllable object with this reference, or nil.
func (ied *IED) Control(ref string) *pyjson.Object {
	for _, control := range ied.controls {
		if text(control, "ref") == ref {
			return control
		}
	}
	return nil
}

// Signal is the signal with this reference, or nil.
func (ied *IED) Signal(ref string) *pyjson.Object { return ied.byRef[ref] }

// FullRef is ref as stored when a person left out the IED name ("LD0/XCBR1.Pos.stVal" for
// "FEED1LD0/XCBR1.Pos.stVal"); the reference itself when it is already exact.
func (ied *IED) FullRef(ref string, command bool) string {
	ref = strings.TrimSpace(ref)
	exact := ied.byRef[ref] != nil
	if command {
		exact = ied.Control(ref) != nil
	}
	if exact || strings.HasPrefix(ref, ied.Name) {
		return ref
	}
	return ied.Name + ref
}

// Description is the SCL description of a reference: the signal's own description, else
// the description of a signal of the same data object (a command uses its status signal's).
func (ied *IED) Description(ref string) string {
	if signal := ied.byRef[ref]; signal != nil && strings.TrimSpace(text(signal, "description")) != "" {
		return strings.TrimSpace(text(signal, "description"))
	}
	if control := ied.Control(ref); control != nil {
		if status := text(control, "status"); status != "" && status != ref {
			return ied.Description(status)
		}
	}
	object := dataObjectOf(ref)
	for _, signal := range ied.signals {
		if dataObjectOf(text(signal, "ref")) == object && strings.TrimSpace(text(signal, "description")) != "" {
			return strings.TrimSpace(text(signal, "description"))
		}
	}
	return ""
}

// ShortName is the readable name used when the SCL has no description: "XCBR1 Pos".
func ShortName(ref string) string {
	after := ref
	if at := strings.IndexByte(ref, '/'); at >= 0 {
		after = ref[at+1:]
	}
	parts := strings.Split(after, ".")
	if len(parts) >= 2 {
		return parts[0] + " " + parts[1]
	}
	return after
}

// dataObjectOf is "LD/LN.DO" of a reference.
func dataObjectOf(ref string) string {
	parts := strings.SplitN(ref, ".", 3)
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return ref
}

// StatusOf is the status point of a command: its reference and gateway kind ("" when none).
func (ied *IED) StatusOf(control *pyjson.Object) (string, string) {
	status := text(control, "status")
	signal := ied.byRef[status]
	if status == "" || signal == nil {
		return "", ""
	}
	kind := map[string]string{"DPC": "double", "BSC": gateway.KindStepPosition, "ISC": gateway.KindStepPosition,
		"APC": "float", "INC": "integer"}[text(control, "cdc")]
	if kind == "" {
		kind = "single"
	}
	valueKind := text(signal, "kind")
	if kind == gateway.KindStepPosition { // a tap position is a whole number (ADR-0107)
		if !strings.HasPrefix(valueKind, "INT") {
			return "", ""
		}
	} else if !contains(gateway.AllowedKinds(valueKind), kind) {
		return "", ""
	}
	return status, kind
}

// commandTypeFor names the command type that operates a control class, for a refusal.
func commandTypeFor(cdc string) string {
	switch cdc {
	case "DPC":
		return "a double command"
	case "BSC", "ISC":
		return "a regulating step command"
	case "APC", "INC":
		return "a set point command"
	case "ENC", "BAC":
		return "a command in the SCADA master or IED Explorer (no gateway type carries it yet)"
	}
	return "a single command"
}

func text(object *pyjson.Object, key string) string {
	if object == nil {
		return ""
	}
	value, _ := object.Get(key)
	result, _ := value.(string)
	return result
}

func list(object *pyjson.Object, key string) []any {
	value, _ := object.Get(key)
	items, _ := value.([]any)
	return items
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
