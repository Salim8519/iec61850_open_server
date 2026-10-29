package goosedom

import (
	"strings"

	"studio-engine/internal/acquisition/livedom"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

const (
	notAPoint    = "its input address is not a data point of this IED"
	manyPoints   = "its input address matches more than one data point"
	structureMsg = ": the member is a structure; its layout is known only when the publisher's file is loaded"
)

// Binding is one bound input, the Python dict {target, kind, member, path, desc,
// source_ref}; an alias so bindings stay pyjson values.
type Binding = *pyjson.Object

// signalTable is {signal["ref"]: signal} with Python dict order (first position, last value).
type signalTable struct {
	refs []string
	byID map[string]*pyjson.Object
}

func newSignalTable(signals []*pyjson.Object) signalTable {
	table := signalTable{byID: map[string]*pyjson.Object{}}
	for _, signal := range signals {
		ref := pyv.PyStr(pyv.Get(signal, "ref"))
		if _, found := table.byID[ref]; !found {
			table.refs = append(table.refs, ref)
		}
		table.byID[ref] = signal
	}
	return table
}

// target is _target: the one data point an intAddr names, or a reason.
func (t signalTable) target(intAddr any, ld any) (string, string) {
	address, _ := intAddr.(string)
	if !pyv.Truthy(intAddr) || strings.ContainsAny(address, " |") {
		return "", notAPoint
	}
	candidates := []string{address}
	if pyv.Truthy(ld) {
		candidates = append(candidates, pyv.PyStr(ld)+"/"+address)
	}
	for _, candidate := range candidates {
		if _, found := t.byID[candidate]; found {
			return candidate, ""
		}
	}
	var matches []string
	for _, ref := range t.refs {
		if strings.HasSuffix(ref, "/"+address) {
			matches = append(matches, ref)
		}
	}
	switch {
	case len(matches) == 1:
		return matches[0], ""
	case len(matches) > 1:
		return "", manyPoints
	}
	return "", notAPoint
}

// BindInputs returns (bindings, problems) for one subscription of one simulated
// subscriber IED. Each binding is a new object {target, kind, member, path, desc,
// source_ref}; an input is applied only when its intAddr names exactly one
// runtime-editable data attribute (ADR-0066).
func BindInputs(ied, subscription *pyjson.Object) ([]Binding, []string) {
	return NewBinder().Bind(ied, subscription)
}

// Binder binds several subscriptions and reads each subscriber IED's signals once: an IED with
// many subscriptions no longer rebuilds its whole signal map for each (ADR-0129).
type Binder struct{ tables map[*pyjson.Object]signalTable }

// NewBinder is a Binder for one preparation of the GOOSE inputs.
func NewBinder() *Binder { return &Binder{tables: map[*pyjson.Object]signalTable{}} }

// Bind is BindInputs with the subscriber's signal map kept for the next subscription.
func (b *Binder) Bind(ied, subscription *pyjson.Object) ([]Binding, []string) {
	signals, found := b.tables[ied]
	if !found {
		signals = newSignalTable(objects(ied, "signals"))
		b.tables[ied] = signals
	}
	domain := ""
	if ref := pyv.Get(subscription, "dataset_ref"); pyv.Truthy(ref) {
		domain = strings.SplitN(pyv.PyStr(ref), "/", 2)[0]
	}
	bindings, problems := []Binding{}, []string{}
	for _, item := range objects(subscription, "inputs") {
		label := pyv.Get(item, "desc")
		if !pyv.Truthy(label) {
			label = pyv.Get(item, "int_addr")
		}
		if !pyv.Truthy(label) {
			label = pyv.GetOr(item, "source", "")
		}
		desc := pyv.PyStr(label)
		if pyv.Get(item, "member") == nil {
			problems = append(problems, "Input "+desc+" is not in the dataset of "+
				pyv.PyStr(pyv.Get(subscription, "gocb_ref")))
			continue
		}
		target, reason := signals.target(pyv.GetOr(item, "int_addr", ""), pyv.GetOr(item, "ld", ""))
		if target == "" {
			problems = append(problems, "Input "+desc+" is shown but not applied: "+reason)
			continue
		}
		signal := signals.byID[target]
		if !pyv.Truthy(pyv.Get(signal, "writable")) || pyv.Equal(pyv.Get(signal, "fc"), "CF") {
			problems = append(problems, "Input "+desc+" is shown but not applied: "+target+
				" cannot be changed while running")
			continue
		}
		source := strings.SplitN(strOr(item, "source", ""), "/", 3)
		sourceRef := ""
		if domain != "" && len(source) == 3 {
			sourceRef = domain + "/" + source[2]
		}
		bindings = append(bindings, pyv.O("target", target, "kind", pyv.Get(signal, "kind"),
			"member", pyv.Get(item, "member"), "path", pyv.GetOr(item, "member_path", ""), "desc", label,
			"source_ref", sourceRef))
	}
	return bindings, problems
}

// AcceptFrame returns the reason a received frame must not drive inputs, or "" when it
// may. memberCount 0 means the SCD member count is unknown.
func AcceptFrame(event *pyjson.Object, memberCount int64) string {
	if !pyv.Equal(pyv.Get(event, "state"), "receiving") || pyv.Truthy(pyv.Get(event, "problem")) {
		why := pyv.Get(event, "problem")
		if !pyv.Truthy(why) {
			why = pyv.Get(event, "state")
		}
		return "GOOSE is " + pyv.PyStr(why)
	}
	values, isList := pyv.Get(event, "values").([]any)
	if !isList {
		return "the frame carried no dataset values"
	}
	if memberCount != 0 && int64(len(values)) != memberCount {
		return "the dataset size differs from the SCD"
	}
	return ""
}

// InputUpdates returns the (target, value) writes for every binding whose value can be
// read without guessing, plus worded problems. publisherValues may be nil.
func InputUpdates(bindings []Binding, values []any, publisherValues *pyjson.Object) ([]Update, []string) {
	updates, problems := []Update{}, []string{}
	for _, binding := range bindings {
		desc := pyv.PyStr(pyv.Get(binding, "desc"))
		raw := memberValue(values, pyv.Get(binding, "member"))
		if _, structured := raw.([]any); structured {
			raw = nil
			if pyv.Truthy(pyv.Get(binding, "path")) {
				raw = pyv.Get(publisherValues, pyv.PyStr(pyv.Get(binding, "source_ref")))
			}
			if raw == nil {
				problems = append(problems, "Input "+desc+structureMsg)
				continue
			}
		}
		value, err := livedom.Convert(pyv.Str(pyv.Get(binding, "kind")), raw)
		if err != nil {
			problems = append(problems, "Input "+desc+": "+err.Error())
			continue
		}
		updates = append(updates, Update{Ref: pyv.PyStr(pyv.Get(binding, "target")), Value: value})
	}
	return updates, problems
}

// memberValue is `values[member] if member < len(values) else None`, with Python's
// negative indexing; an index below -len(values) (IndexError in Python) gives nil.
func memberValue(values []any, member any) any {
	index, ok := pyv.Int64(member)
	if flag, isBool := member.(bool); isBool {
		index, ok = 0, true
		if flag {
			index = 1
		}
	}
	if !ok || index >= int64(len(values)) {
		return nil
	}
	if index < 0 {
		index += int64(len(values))
		if index < 0 {
			return nil
		}
	}
	return values[index]
}
