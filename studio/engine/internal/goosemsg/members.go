package goosemsg

import (
	"sort"
	"strings"
)

// members expands every dataset member of one block. stored are the control's member texts
// ("XCBR1$ST$Pos", "CTRL/XCBR1$ST$Pos$stVal", "GGIO1$ST$Arr 2 stVal"); the dataset's
// fcda_refs hold the same members in the same order with the logical device already resolved
// to its domain (ldName, else IED name + ldInst): "IED1LD0/XCBR1.Pos".
func members(stored []any, dataset any, domain string, signals map[string][]Leaf) []Member {
	refs := list(field(object(dataset), "fcda_refs"))
	if len(refs) == 0 || len(refs) != len(stored) {
		return nil
	}
	result := make([]Member, 0, len(refs))
	for index, item := range refs {
		reference, _ := item.(string)
		entry, _ := stored[index].(string)
		fc := memberFC(entry)
		member := Member{Position: index + 1, FC: fc, Text: memberText(reference, domain, fc)}
		member.Leaves = Expand(reference, fc, signals)
		result = append(result, member)
	}
	return result
}

// memberFC reads the functional constraint of a stored member text: the field after the node.
func memberFC(entry string) string {
	if at := strings.Index(entry, "/"); at >= 0 && at < strings.Index(entry, "$") {
		entry = entry[at+1:]
	}
	parts := strings.Split(entry, "$")
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

// memberText is what the owner reads: the member without its own domain, then its FC.
func memberText(reference, domain, fc string) string {
	shown := reference
	if owner, rest, found := strings.Cut(reference, "/"); found && owner == domain {
		shown = rest
	}
	if fc == "" {
		return shown
	}
	return shown + " [" + fc + "]"
}

// Expand returns the leaf signals one dataset member covers (IEC 61850-7-2 FCDA): the attribute
// it names, or every attribute below the data object it names, but only those with the member's
// functional constraint. "XCBR1.Pos" with ST covers Pos.stVal, Pos.q and Pos.t, never
// Pos.ctlModel (CF), Pos.d (DC) or Pos.subVal (SV). signals are sorted by reference per FC.
func Expand(reference, fc string, signals map[string][]Leaf) []Leaf {
	if reference == "" || fc == "" {
		return nil
	}
	candidates := signals[fc]
	start := sort.Search(len(candidates), func(index int) bool { return candidates[index].Ref >= reference })
	var leaves []Leaf
	for index := start; index < len(candidates) && strings.HasPrefix(candidates[index].Ref, reference); index++ {
		ref := candidates[index].Ref
		if ref == reference || strings.HasPrefix(ref, reference+".") || strings.HasPrefix(ref, reference+"(") {
			leaves = append(leaves, candidates[index])
		}
	}
	return leaves
}
