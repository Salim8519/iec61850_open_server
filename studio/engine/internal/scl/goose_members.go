package scl

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// SubscriptionMembers is acquisition/goose_inputs._with_members (it implements
// acquisition.SubscriptionMembers): a stored subscription without dataset positions
// (projects imported before change 0094) is completed from the IED's stored SCL; one that
// has them, or that the SCL no longer declares, is returned unchanged. Errors are the
// SCL's parse errors and Python's KeyError texts for a missing "source", "name" or
// "gocb_ref".
func SubscriptionMembers(ied, subscription *pyjson.Object) (*pyjson.Object, error) {
	if complete(subscription) {
		return subscription, nil
	}
	source, ok := get(ied, "source").(string)
	if !ok {
		return nil, apperr.New("'source'")
	}
	parsed, err := ParseDocument([]byte(source), false)
	if err != nil {
		return nil, err
	}
	name, hasName := ied.Get("name")
	if !hasName {
		return nil, apperr.New("'name'")
	}
	var found []any
	for _, element := range parsed.Root.FindAll("IED") {
		if value, present := element.Get("name"); present && value == name {
			found, _ = gooseSubscriptions(parsed.Root, element, publisherCatalogue(parsed.Root))
			break
		}
	}
	reference, hasReference := subscription.Get("gocb_ref")
	for _, item := range found {
		if !hasReference {
			return nil, apperr.New("'gocb_ref'")
		}
		if entry := item.(*pyjson.Object); get(entry, "gocb_ref") == reference {
			return entry, nil
		}
	}
	return subscription, nil
}

// complete is all("member" in item for item in inputs) and "member_count" in subscription.
func complete(subscription *pyjson.Object) bool {
	inputs, _ := get(subscription, "inputs").([]any)
	for _, item := range inputs {
		entry, ok := item.(*pyjson.Object)
		if !ok {
			return false
		}
		if _, has := entry.Get("member"); !has {
			return false
		}
	}
	_, has := subscription.Get("member_count")
	return has
}
