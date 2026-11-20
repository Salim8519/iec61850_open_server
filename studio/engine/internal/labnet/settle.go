package labnet

import (
	"strconv"
	"time"
)

// settleProbe is what settle reads and changes on one real card: WindowsOps on Windows, a fake in
// tests.
type settleProbe interface {
	// Duplicate says whether Windows found another device using ip (its duplicate check failed).
	Duplicate(card RealCard, ip string) bool
	// Usable says whether a program can listen on ip now (Windows finished checking it).
	Usable(ip string) bool
	RemoveReal(card RealCard, ip string) error
}

// settle waits at most wait while Windows checks the added addresses, all together. An address
// another device uses is taken off again and reported. An address Windows is still checking when
// the wait ends is taken off too and reported (change 0221, trial N6): before, it was reported as
// added although no IED could listen on it, and it stayed on the card.
func settle(probe settleProbe, card RealCard, ips []string, wait time.Duration, now func() time.Time,
	sleep func(time.Duration)) map[string]string {
	failed := map[string]string{}
	waiting := append([]string{}, ips...)
	for deadline := now().Add(wait); len(waiting) > 0; {
		still := []string{}
		for _, ip := range waiting {
			switch {
			case probe.Duplicate(card, ip):
				probe.RemoveReal(card, ip)
				failed[ip] = "Another device on “" + card.Name + "” already uses " + ip + "; ScadaBench took it off again"
			case !probe.Usable(ip):
				still = append(still, ip) // Windows is still checking the network (tentative)
			}
		}
		waiting = still
		if len(waiting) == 0 || !now().Before(deadline) {
			break
		}
		sleep(100 * time.Millisecond)
	}
	seconds := strconv.Itoa(int(wait / time.Second))
	for _, ip := range waiting {
		probe.RemoveReal(card, ip)
		failed[ip] = "Windows was still checking " + ip + " on “" + card.Name + "” after " + seconds +
			" seconds, so ScadaBench took it off again. Check the cable and the network switch, then press Start again"
	}
	return failed
}

// disconnected is the reason an address is not added to a card that has no link (change 0221,
// trial N6): Windows cannot check the network there, and no device could reach the IED.
func disconnected(card RealCard, ip string) string {
	return "“" + card.Name + "” is disconnected (no cable, or its network is switched off), so ScadaBench did not add " +
		ip + ". Plug the cable in, then press Start again"
}
