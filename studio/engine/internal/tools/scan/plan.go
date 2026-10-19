package scan

import (
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/tools/pytext"
)

// MaxHosts is the most addresses one scan may probe (a /24).
const MaxHosts = 256

const (
	textEmpty    = "Type a network such as 192.168.1.0/24, one address, or a range such as 192.168.1.20-40"
	textTooLarge = "Scan at most 256 addresses (a /24 network) so the scan stays quick and polite"
	textBackward = "The range must go from the lower to the higher address"
	textSpecial  = "Multicast and 0.0.0.0 addresses cannot be scanned"
)

// Plan is plan_range's result.
type Plan struct {
	Range string
	Hosts []string
	Count int
	First string
	Last  string
}

// PlanRange is domain/scan_range.plan_range(text, own_ip); ownIP "" stands for None.
func PlanRange(text any, ownIP string) (*Plan, error) {
	value := ""
	if pytext.Truthy(text) {
		value = pytext.Str(text)
	}
	value = pytext.Strip(value)
	if value == "" {
		return nil, apperr.ValueError(textEmpty)
	}
	first, last, err := bounds(value)
	if err != nil {
		return nil, err
	}
	if last-first+1 > MaxHosts {
		return nil, apperr.ValueError(textTooLarge)
	}
	plan := &Plan{Range: value, Hosts: []string{}}
	for address := first; ; address++ {
		if address>>28 == 0xe || address == 0 {
			return nil, apperr.ValueError(textSpecial)
		}
		if address == last {
			break
		}
	}
	for address := first; ; address++ {
		if text := format(address); text != ownIP {
			plan.Hosts = append(plan.Hosts, text)
		}
		if address == last {
			break
		}
	}
	plan.Count = len(plan.Hosts)
	if plan.Count > 0 {
		plan.First, plan.Last = plan.Hosts[0], plan.Hosts[plan.Count-1]
	}
	return plan, nil
}

// bounds returns the first and last host of a range or network, with plan_range's errors.
func bounds(text string) (uint32, uint32, error) {
	invalid := apperr.ValueError("'" + text + "' is not a valid address, range or network")
	if start, end, isRange := strings.Cut(text, "-"); isRange {
		first, err := parseAddress(pytext.Strip(start))
		if err != nil {
			return 0, 0, invalid
		}
		endText := pytext.Strip(end)
		var last uint32
		if strings.Contains(endText, ".") {
			last, err = parseAddress(endText)
		} else {
			number, parseErr := pytext.ParseInt(endText)
			if parseErr != nil {
				return 0, 0, invalid
			}
			prefix := format(first)
			prefix = prefix[:strings.LastIndexByte(prefix, '.')]
			last, err = parseAddress(prefix + "." + pytext.Str(number))
		}
		if err != nil {
			return 0, 0, invalid
		}
		if last < first {
			return 0, 0, apperr.ValueError(textBackward)
		}
		return first, last, nil
	}
	network, prefix, err := parseNetwork(text)
	if err != nil {
		return 0, 0, invalid
	}
	if prefix < 24 {
		return 0, 0, apperr.ValueError(textTooLarge)
	}
	broadcast := network | ^mask(prefix)
	if prefix < 31 {
		return network + 1, broadcast - 1, nil
	}
	return network, broadcast, nil
}
