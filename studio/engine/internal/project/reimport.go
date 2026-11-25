package project

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/project/netset"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// Re-import (ADR-0150): a newer CID of an IED already in the project replaces its model. The
// IED keeps its id, its settings (role, addresses, Allow commands, card, GOOSE, test mode, SCD
// deadbands) and every link whose reference is still in the new file with the same type; the
// others are listed in words before anything changes. Pure: no I/O.

// ReimportLost is one link the new file cannot keep: where it is, the reference and why.
type ReimportLost struct{ Where, Ref, Why string }

// ReimportPlan is the project after the swap and what it changes.
type ReimportPlan struct {
	Project *pyjson.Object
	Lost    []ReimportLost
	Kept    int // links to this IED that stay
	Gateway []string // gateway points of this IED whose reference is gone (kept, they will not update)
}

// ReimportedIED is the new model with the old IED's identity and settings.
func ReimportedIED(old, model *pyjson.Object) (*pyjson.Object, error) {
	settings, err := netset.ValidateIEDSettings(old)
	if err != nil {
		return nil, err
	}
	merged := pyval.CopyObject(model)
	merged.Set("id", pyval.Lookup(old, "id", nil)).Set("ip", pyval.Lookup(old, "ip", "")).
		Set("port", pyval.Lookup(old, "port", int64(102))).Set("state", pyval.Lookup(old, "state", "stopped"))
	appendAll(merged, settings)
	if pyval.Lookup(old, "scd_deadbands", nil) == true {
		merged.Set("scd_deadbands", true)
	}
	return merged, nil
}

// Reimport swaps the IED's model for model (imported from the new file under the same name) and
// removes only the links the new model cannot keep. The result passes the same checks as opening
// the project, or an error says why and nothing changes.
func Reimport(project *pyjson.Object, iedID string, model *pyjson.Object) (*ReimportPlan, error) {
	old, err := FindIED(project, iedID)
	if err != nil {
		return nil, err
	}
	if text(model, "name") != text(old, "name") {
		return nil, apperr.New("The new file's IED is " + text(model, "name") + ", not " + text(old, "name"))
	}
	fresh, err := ReimportedIED(old, model)
	if err != nil {
		return nil, err
	}
	result := pyjson.NewObject() // shallow: only the parts that may change are copied
	for _, key := range project.Keys() {
		value, _ := project.Get(key)
		result.Set(key, value)
	}
	ieds := []*pyjson.Object{}
	list := []any{}
	for _, item := range items(pyval.Lookup(project, "ieds", nil)) {
		ied := object(item)
		if text(ied, "id") == iedID {
			ied = fresh
		}
		ieds = append(ieds, ied)
		list = append(list, ied)
	}
	result.Set("ieds", list)
	for _, key := range []string{"diagram", "sources", "test_panel", "scenarios"} {
		if value, present := project.Get(key); present {
			result.Set(key, pyval.DeepCopy(value))
		}
	}
	p := &pruner{target: iedID, name: text(old, "name"), old: old, fresh: fresh, ieds: NewIndex().For(ieds)}
	p.diagram(object(pyval.Lookup(result, "diagram", nil)))
	result.Set("sources", p.list(pyval.Lookup(result, "sources", []any{}), "Signal source", func(entry any) error {
		_, err := ValidateSources([]any{entry}, ieds)
		return err
	}))
	result.Set("test_panel", p.list(pyval.Lookup(result, "test_panel", []any{}), "Test panel", func(entry any) error {
		_, err := ValidatePanel([]any{entry}, ieds)
		return err
	}))
	result.Set("scenarios", p.scenarios(pyval.Lookup(result, "scenarios", []any{}), ieds))
	if err := finalCheck(result, ieds); err != nil {
		return nil, apperr.New("The new file cannot keep this project consistent (" + err.Error() +
			"). Nothing was changed.")
	}
	return &ReimportPlan{Project: result, Lost: p.lost, Kept: p.kept, Gateway: gatewayGone(result, p)}, nil
}

// finalCheck runs the checks a project gets when it is opened.
func finalCheck(project *pyjson.Object, ieds []*pyjson.Object) error {
	if _, err := ValidateDiagram(pyval.Lookup(project, "diagram", pyjson.NewObject()), ieds); err != nil {
		return err
	}
	if _, err := ValidateSources(pyval.Lookup(project, "sources", []any{}), ieds); err != nil {
		return err
	}
	if _, err := ValidateScenarios(pyval.Lookup(project, "scenarios", []any{}), ieds); err != nil {
		return err
	}
	_, err := ValidatePanel(pyval.Lookup(project, "test_panel", []any{}), ieds)
	return err
}

// gatewayGone lists this IED's gateway points whose reference the new file no longer has. They
// are kept as they are (the gateway does not require them), but they will not update.
func gatewayGone(project *pyjson.Object, p *pruner) []string {
	var gone []string
	points := object(pyval.Lookup(object(pyval.Lookup(project, "gateway", nil)), "points", nil))
	if points == nil {
		return gone
	}
	for _, protocol := range points.Keys() {
		value, _ := points.Get(protocol)
		for _, item := range items(value) {
			point := object(item)
			if text(point, "ied") != p.name {
				continue
			}
			ref := text(point, "ref")
			if !p.hasRef(ref) {
				gone = append(gone, protocol+": "+ref)
			}
		}
	}
	return gone
}
