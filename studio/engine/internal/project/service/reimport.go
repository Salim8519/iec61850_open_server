package service

import (
	"sort"
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/memtrim"
	"studio-engine/internal/project"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
	"studio-engine/internal/store/recovery"
)

// Reimport reads a newer CID of one IED (ADR-0150). With apply false it only answers what would
// change; with apply true it first writes a verified recovery backup of the project, then swaps
// the IED's model and keeps its id, settings and every link the new file still has (Undo
// restores it). The answer: {id, name, filename, same_file, signals, controls, reports, kept,
// lost[{where, ref, why}], gateway_gone[], applied, recovery_file?}.
func (s *Service) Reimport(key string, source []byte, filename string, apply bool) (*pyjson.Object, error) {
	defer memtrim.Soon()
	ied, err := project.FindIED(s.current.Load().project, key)
	if err != nil {
		return nil, err
	}
	name, _ := ied.Get("name")
	label, _ := name.(string)
	models, err := s.config.Importer.Import(source, filename, scl.Selection{Names: []any{label}})
	if err != nil {
		return nil, explainMissing(err, source, label)
	}
	s.mutate.Lock()
	defer s.mutate.Unlock()
	st := s.current.Load()
	old, err := project.FindIED(st.project, key)
	if err != nil {
		return nil, err
	}
	plan, err := project.Reimport(st.project, key, models[0])
	if err != nil {
		return nil, err
	}
	answer := reimportAnswer(old, models[0], plan).Set("applied", false)
	if !apply {
		return answer, nil
	}
	payload, err := s.encoder.Encode(st.compact)
	if err != nil {
		return nil, err
	}
	backup, err := recovery.WriteReimportBackup(s.config.DataDir, payload)
	if err != nil {
		return nil, err
	}
	if err := s.commit(plan.Project, st.notice, true, nil); err != nil {
		return nil, err
	}
	return answer.Set("applied", true).Set("recovery_file", backup), nil
}

// explainMissing names the IEDs a file has when the project's IED is not one of them.
func explainMissing(err error, source []byte, name string) error {
	doc, parseErr := scl.ParseDocument(source, false)
	if parseErr != nil {
		return err
	}
	var names []string
	for _, element := range doc.Root.FindAll("IED") {
		if text, ok := element.Get("name"); ok {
			if text == name {
				return err // the IED is there; the import failed for another reason
			}
			names = append(names, text)
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return apperr.New("This file has no IED. Choose the CID of " + name)
	}
	return apperr.New("This file has no IED named " + name + "; it has " + strings.Join(names, ", ") +
		". Choose the CID of " + name + " (the IED name must stay the same)")
}

func reimportAnswer(old, model *pyjson.Object, plan *project.ReimportPlan) *pyjson.Object {
	lost := make([]any, 0, len(plan.Lost))
	for _, item := range plan.Lost {
		lost = append(lost, pyjson.NewObject().Set("where", item.Where).Set("ref", item.Ref).Set("why", item.Why))
	}
	gone := make([]any, 0, len(plan.Gateway))
	for _, item := range plan.Gateway {
		gone = append(gone, item)
	}
	oldSource, _ := old.Get("source")
	newSource, _ := model.Get("source")
	return pyjson.NewObject().Set("id", value(old, "id")).Set("name", value(old, "name")).
		Set("filename", value(model, "filename")).Set("same_file", oldSource == newSource).
		Set("signals", counts(old, model, "signals")).Set("controls", counts(old, model, "controls")).
		Set("reports", counts(old, model, "reports")).Set("kept", int64(plan.Kept)).
		Set("lost", lost).Set("gateway_gone", gone)
}

// counts is {before, after, added, removed} of one list of references.
func counts(old, model *pyjson.Object, key string) *pyjson.Object {
	before, after := refs(old, key), refs(model, key)
	added, removed := 0, 0
	for ref := range after {
		if !before[ref] {
			added++
		}
	}
	for ref := range before {
		if !after[ref] {
			removed++
		}
	}
	return pyjson.NewObject().Set("before", int64(len(before))).Set("after", int64(len(after))).
		Set("added", int64(added)).Set("removed", int64(removed))
}

func refs(ied *pyjson.Object, key string) map[string]bool {
	result := map[string]bool{}
	list, _ := ied.Get(key)
	entries, _ := list.([]any)
	for _, item := range entries {
		if entry, ok := item.(*pyjson.Object); ok {
			if ref, isText := value(entry, "ref").(string); isText {
				result[ref] = true
			}
		}
	}
	return result
}

func value(o *pyjson.Object, key string) any {
	found, _ := o.Get(key)
	return found
}
