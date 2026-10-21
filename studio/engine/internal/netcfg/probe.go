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

// ARPProber probes with one question per address, at most 64 at a time (16 before change
// 0150; each question still waits the full time Windows gives it). Answers defaults to
// SendARP on Windows (always false elsewhere); Report, when set, hears each answer as it
// arrives.
type ARPProber struct {
	Answers func(address string) bool
	Report  func(address string, used bool)
}

// arpParallel is how many addresses are asked at the same time.
const arpParallel = 64

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
	slots := make(chan struct{}, min(arpParallel, len(candidates)))
	var group sync.WaitGroup
	for index, address := range candidates {
		group.Add(1)
		slots <- struct{}{}
		go func(index int, address string) {
			defer group.Done()
			results[index] = answers(address)
			if p.Report != nil {
				p.Report(address, results[index])
			}
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
