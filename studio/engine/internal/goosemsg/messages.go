package goosemsg

import (
	"sort"

	"studio-engine/internal/pyjson"
)

// Partner is one GOOSE link of the project's link map (GET /goose/links): who receives which block.
type Partner struct {
	GocbRef, Publisher, Subscriber, Block, Status string
}

// SendState says whether values of this IED can be sent now, and why not in plain words.
type SendState struct {
	Running bool
	Refusal string
}

// Messages is the answer of GET /ieds/{ied_id}/goose/messages (ADR-0112): the IED's sendable
// messages with their receivers and members (each member's leaves, their types and, while
// running, their values) and the messages it receives. value(ref) gives a running value.
func Messages(catalog *Catalog, role string, partners []Partner, value func(string) (any, bool),
	state SendState) *pyjson.Object {
	messages := make([]any, 0, len(catalog.Blocks))
	for _, block := range catalog.Blocks {
		subscribers := []any{}
		seen := map[string]bool{}
		for _, partner := range partners {
			if partner.GocbRef == block.GocbRef && partner.Publisher == catalog.Name && !seen[partner.Subscriber] {
				seen[partner.Subscriber] = true
				subscribers = append(subscribers, partner.Subscriber)
			}
		}
		members := make([]any, 0, len(block.Members))
		for _, member := range block.Members {
			refs := make([]any, 0, len(member.Leaves))
			for _, leaf := range member.Leaves {
				entry := pyjson.NewObject().Set("ref", leaf.Ref).Set("kind", leaf.Kind).Set("fc", leaf.FC).
					Set("value_type", nullable(ValueType(leaf.Kind))).Set("writable", leaf.Writable).
					Set("sendable", leaf.Sendable()).Set("description", leaf.Description).Set("value", nil)
				if state.Running && value != nil {
					if current, found := value(leaf.Ref); found {
						entry.Set("value", current)
					}
				}
				refs = append(refs, entry)
			}
			members = append(members, pyjson.NewObject().Set("position", int64(member.Position)).
				Set("text", member.Text).Set("fc", member.FC).Set("refs", refs))
		}
		messages = append(messages, pyjson.NewObject().Set("gocb_ref", block.GocbRef).Set("name", block.Name).
			Set("dataset", block.Dataset).Set("go_id", block.GoID).Set("app_id", block.AppID).
			Set("conf_rev", block.ConfRev).Set("subscribers", subscribers).
			Set("complete", len(block.Members) > 0).Set("members", members))
	}
	return pyjson.NewObject().Set("ied_id", catalog.IEDID).Set("name", catalog.Name).Set("role", role).
		Set("running", state.Running).Set("can_send", state.Refusal == "").Set("why_not", state.Refusal).
		Set("messages", messages).Set("receives", receives(catalog.Name, partners))
}

func nullable(text string) any {
	if text == "" {
		return nil
	}
	return text
}

// receives lists the messages this IED gets, one entry per sender block, sorted by sender.
func receives(name string, partners []Partner) []any {
	var chosen []Partner
	seen := map[string]bool{}
	for _, partner := range partners {
		key := partner.Publisher + "\x00" + partner.GocbRef + "\x00" + partner.Block
		if partner.Subscriber == name && !seen[key] {
			seen[key] = true
			chosen = append(chosen, partner)
		}
	}
	sort.SliceStable(chosen, func(a, b int) bool { return chosen[a].Publisher < chosen[b].Publisher })
	result := make([]any, len(chosen))
	for index, partner := range chosen {
		result[index] = pyjson.NewObject().Set("gocb_ref", nullable(partner.GocbRef)).
			Set("publisher", partner.Publisher).Set("block", partner.Block).Set("status", partner.Status)
	}
	return result
}

// LeafRefs is the answer of GET /goose/members: for every project IED, every sendable block's
// exact leaf references (the SLD uses them to tell which changed values a message carries).
func LeafRefs(ieds []*pyjson.Object) *pyjson.Object {
	result := pyjson.NewObject()
	for _, ied := range ieds {
		catalog := For(ied)
		if len(catalog.Blocks) == 0 {
			continue
		}
		blocks := pyjson.NewObject()
		for _, block := range catalog.Blocks {
			refs := []any{}
			for _, member := range block.Members {
				for _, leaf := range member.Leaves {
					refs = append(refs, leaf.Ref)
				}
			}
			blocks.Set(block.GocbRef, refs)
		}
		result.Set(catalog.IEDID, blocks)
	}
	return pyjson.NewObject().Set("ieds", result)
}
