package labsetup

import (
	"sort"
	"sync"
)

// Addresses kept for the session (ADR-0141, trial N10): an IED's address that this run of
// ScadaBench put on a card stays there after Stop while the project stays open, so the next Run
// adds nothing and Windows (and its DNS Client) sees no change. It comes off when the project is
// closed or another one opened, when the network changes or the IED moves, at exit (EndSession)
// and, after a kill, at the next launch (the session is empty then and the journal says the
// address is ours).
type session struct {
	mu    sync.Mutex
	added map[string]bool // realKey(card, address)
}

func (k *session) keep(card, address string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.added == nil {
		k.added = map[string]bool{}
	}
	k.added[realKey(card, address)] = true
}

func (k *session) has(card, address string) bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.added[realKey(card, address)]
}

func (k *session) forget(card, address string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	delete(k.added, realKey(card, address))
}

// forgetAll forgets the addresses that came off (every one in off that is not in failed).
func (k *session) forgetAll(card string, off []string, failed []failure) {
	stayed := map[string]bool{}
	for _, item := range failed {
		stayed[item.address] = true
	}
	for _, address := range off {
		if !stayed[address] {
			k.forget(card, address)
		}
	}
}

func (k *session) clear() {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.added = nil
}

// CleanUp is POST /network/lab/cleanup, the lab page's "Take them off now" (ADR-0141): every
// address this copy put on a card that nothing uses comes off now, also the ones kept for the
// session. Refused while IEDs run.
func (s *Service) CleanUp() (Status, error) {
	s.change.Lock()
	defer s.change.Unlock()
	if len(s.Runtime.Running()) > 0 {
		return Status{}, plain("Stop the simulation first. The addresses of running IEDs stay on the card.")
	}
	if err := s.Runtime.Stopped(s.EndSession); err != nil {
		return Status{}, err
	}
	return s.Status(nil)
}

// keptRows is the stopped IEDs' addresses on the card that this session keeps there.
func (s *Service) keptRows(card string, rows []Row) []string {
	kept, running := []string{}, s.Runtime.Running()
	for _, row := range rows {
		if card != "" && row.OnCard && !contains(running, row.IEDID) && s.session.has(card, row.IP) {
			kept = append(kept, row.IP)
		}
	}
	sort.Strings(kept)
	return kept
}
