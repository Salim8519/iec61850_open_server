package project

import (
	"regexp"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project/gateway"
	"studio-engine/internal/project/netset"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

var projectID = regexp.MustCompile(`^[0-9a-f]{32}$`)

// RestoreInput is what ProjectService.restore works on after decode and SCL re-import.
type RestoreInput struct {
	Raw     *pyjson.Object            // decoded and expanded archive document
	Version int64                     // from StoredIEDs
	Models  map[string]*pyjson.Object // freshly imported model per stored IED id (StoredGroup.Assign)
	// Preserve writes the exact archive bytes to recovery/before-format-8-migration-<uuid>
	// .scadabench and returns its path; it is called only for a legacy format, right after the
	// diagram migration, as Python does.
	Preserve func() (string, error)
	NewID    func() string // uuid4().hex, used when the document has no id
}

// Restored is the project and migration notice (nil for None) that restore() installs;
// the caller swaps them in, saves when persisting and announces the change.
type Restored struct {
	Project *pyjson.Object
	Notice  *pyjson.Object
}

// PrepareRestore is everything ProjectService.restore does between the re-import and the
// swap: per-IED settings, duplicate checks, migration, validation of every section,
// identifiers, the migration notice and loopback addresses.
func PrepareRestore(in RestoreInput) (*Restored, error) {
	raw := in.Raw
	imported := []*pyjson.Object{}
	for _, entry := range items(pyval.Lookup(raw, "ieds", nil)) {
		item := object(entry)
		model := in.Models[text(item, "id")]
		port, err := pyval.Int(pyval.Lookup(item, "port", int64(102)))
		if err != nil {
			return nil, err
		}
		id, _ := item.Get("id")
		model.Set("id", id).Set("ip", pyval.Lookup(item, "ip", "")).Set("port", port)
		settings, err := netset.ValidateIEDSettings(item)
		if err != nil {
			if message, ok := asValueError(err); ok {
				name, _ := item.Get("name")
				return nil, apperr.New("Stored settings for IED " + pyval.Str(name) + ": " + message)
			}
			return nil, err
		}
		appendAll(model, settings)
		imported = append(imported, model)
	}
	names, ids := map[string]bool{}, map[string]bool{}
	for _, model := range imported {
		names[pyval.Str(pyval.Lookup(model, "name", nil))] = true
		ids[text(model, "id")] = true
	}
	if len(names) != len(imported) || len(ids) != len(imported) {
		return nil, apperr.New("Project contains too many or duplicate IEDs")
	}
	migrated, notices, err := MigrateDiagram(pyval.Lookup(raw, "diagram", pyjson.NewObject()), imported, in.Version)
	if err != nil {
		return nil, err
	}
	legacy := in.Version < CurrentFormat
	backup := ""
	if legacy {
		if backup, err = in.Preserve(); err != nil {
			return nil, err
		}
	}
	project, err := restoredProject(raw, migrated, imported, in.NewID)
	if err != nil {
		return nil, err
	}
	network := object(pyval.Lookup(project, "network", nil))
	notice := migrationNotice(in.Version, notices, backup, netset.LabNotice(pyval.Lookup(raw, "network", nil), network))
	if err := AssignEndpoints(imported); err != nil {
		return nil, err
	}
	return &Restored{Project: project, Notice: notice}, nil
}

// restoredProject validates every stored section and builds the project in Python's order.
func restoredProject(raw *pyjson.Object, migrated any, imported []*pyjson.Object, newID func() string) (*pyjson.Object, error) {
	diagram, err := ValidateDiagram(migrated, imported)
	if err != nil {
		return nil, err
	}
	sources, err := ValidateSources(pyval.Lookup(raw, "sources", []any{}), imported)
	if err != nil {
		return nil, err
	}
	scenarios, err := ValidateScenarios(pyval.Lookup(raw, "scenarios", []any{}), imported)
	if err != nil {
		return nil, err
	}
	panel, err := ValidatePanel(pyval.Lookup(raw, "test_panel", []any{}), imported)
	if err != nil {
		return nil, err
	}
	network, err := netset.ValidateNetwork(pyval.Lookup(raw, "network", nil))
	if err != nil {
		return nil, wrapStored("Stored network settings: ", err)
	}
	gatewaySection, err := gateway.NormalizeGateway(pyval.Lookup(raw, "gateway", nil))
	if err != nil {
		return nil, wrapStored("Stored gateway settings: ", err)
	}
	fresh := newID()
	id := pyval.Lookup(raw, "id", fresh)
	if text, isText := id.(string); !isText || !projectID.MatchString(text) {
		return nil, apperr.New("Invalid stored project identifier")
	}
	ieds := make([]any, len(imported))
	for index, model := range imported {
		ieds[index] = model
	}
	name := pyval.Prefix(pyval.Str(pyval.Lookup(raw, "name", "Imported substation")), 100)
	restored := pyjson.NewObject().Set("id", id).Set("name", name).Set("ieds", ieds).Set("diagram", diagram).
		Set("sources", sources).Set("scenarios", scenarios).Set("test_panel", panel).
		Set("network", network).Set("gateway", gatewaySection)
	// The Go-only SNMP devices section (ADR-0092) is kept as stored; the SNMP service checks it.
	if section, isObject := pyval.Lookup(raw, "snmp", nil).(*pyjson.Object); isObject {
		restored.Set("snmp", pyval.DeepCopy(section))
	}
	return restored, nil
}

func wrapStored(prefix string, err error) error {
	if message, ok := asValueError(err); ok {
		return apperr.New(prefix + message)
	}
	return err
}
