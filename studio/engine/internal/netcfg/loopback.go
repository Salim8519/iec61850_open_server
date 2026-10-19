package netcfg

import (
	"strconv"

	"studio-engine/internal/apperr"
)

// AssignLoopback is loopback.assign_endpoints: every IED without an address gets the first
// free 127.50.0.2-254. ips holds each IED's "ip" ("" when none) and is filled in place; like
// Python, IEDs before a failure keep their new address.
func AssignLoopback(ips []string) error {
	used := map[string]bool{}
	for _, ip := range ips {
		if ip != "" {
			used[ip] = true
		}
	}
	for position, ip := range ips {
		if ip != "" {
			continue
		}
		assigned := false
		for index := 2; index < 255; index++ {
			candidate := "127.50.0." + strconv.Itoa(index)
			if !used[candidate] {
				ips[position], used[candidate], assigned = candidate, true, true
				break
			}
		}
		if !assigned {
			return apperr.New("No free loopback addresses")
		}
	}
	return nil
}
