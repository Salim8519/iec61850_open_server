package netcfg

import (
	"sync"

	"studio-engine/internal/apperr"
)

// Prober asks the local network which addresses another device already answers on
// (arp_probe.addresses_in_use). Tests use a fake.
type Prober interface {
	InUse(addresses []string) (map[string]bool, error)
}

// ProberFunc adapts a function to Prober.
type ProberFunc func(addresses []string) (map[string]bool, error)

// InUse calls the function.
func (f ProberFunc) InUse(addresses []string) (map[string]bool, error) { return f(addresses) }

// ARPProber probes with one question per address, at most 16 at a time. Answers defaults to
// SendARP on Windows (always false elsewhere).
type ARPProber struct {
	Answers func(address string) bool
}

// InUse returns the subset of addresses that answered.
func (p ARPProber) InUse(addresses []string) (map[string]bool, error) {
	candidates := []string{}
	for _, address := range addresses {
		if address == "" {
			continue
		}
		ip, ok := parseIPv4(address)
		if !ok {
			return nil, apperr.ValueError(stringRepr(address) + " does not appear to be an IPv4 address")
		}
		candidates = append(candidates, formatIPv4(ip))
	}
	used := map[string]bool{}
	if len(candidates) == 0 {
		return used, nil
	}
	answers := p.Answers
	if answers == nil {
		answers = sendARP
	}
	results := make([]bool, len(candidates))
	slots := make(chan struct{}, min(16, len(candidates)))
	var group sync.WaitGroup
	for index, address := range candidates {
		group.Add(1)
		slots <- struct{}{}
		go func(index int, address string) {
			defer group.Done()
			results[index] = answers(address)
			<-slots
		}(index, address)
	}
	group.Wait()
	for index, address := range candidates {
		if results[index] {
			used[address] = true
		}
	}
	return used, nil
}
