package project

import (
	"strconv"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// AssignEndpoints is networking/loopback.assign_endpoints: every IED without an address
// gets the first free 127.50.0.2-254 loopback address (set in place).
func AssignEndpoints(ieds []*pyjson.Object) error {
	used := map[string]bool{}
	for _, ied := range ieds {
		ip := pyval.Lookup(ied, "ip", nil)
		if !pyval.Truthy(ip) {
			continue
		}
		if err := pyval.CheckHashable(ip); err != nil {
			return err
		}
		if text, ok := ip.(string); ok {
			used[text] = true
		}
	}
	for _, ied := range ieds {
		if pyval.Truthy(pyval.Lookup(ied, "ip", nil)) {
			continue
		}
		assigned := false
		for index := 2; index < 255; index++ {
			candidate := "127.50.0." + strconv.Itoa(index)
			if !used[candidate] {
				ied.Set("ip", candidate)
				used[candidate] = true
				assigned = true
				break
			}
		}
		if !assigned {
			return apperr.New("No free loopback addresses")
		}
	}
	return nil
}
