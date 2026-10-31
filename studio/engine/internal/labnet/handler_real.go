package labnet

import (
	"errors"
	"strings"
)

// RealCard is a real network card as the service changes it (ADR-0132).
type RealCard struct {
	GUID  string
	Name  string
	Index uint32
	LUID  uint64
	Up    bool
	DHCP  bool     // Windows gets the card's own address automatically: never add a fixed one
	Own   []string // the card's own IPv4 address/prefix pairs, e.g. "192.168.230.1/24"
}

// RealOps is what the service needs for approved addresses on real cards.
type RealOps interface {
	// Cards lists the real cards now, by upper-case GUID, with every adapter's addresses.
	Cards() (map[string]RealCard, map[string][]string, error)
	// Approved lists what an administrator approved: card GUID (upper case) -> address -> prefix.
	Approved() map[string]map[string]int
	// Ask asks the card's network, all addresses at once and for a moment only, whether a device
	// already answers there: true answered, false nobody answered. An address missing from the
	// answer could not be asked (no cable, or the card has no own address on that network).
	Ask(card RealCard, prefixes map[string]int) map[string]bool
	// AddReal puts address/prefix on the card: ready at once when the network was asked and
	// nobody answered, else after Windows' own duplicate check. RemoveReal takes it off.
	AddReal(card RealCard, ip string, prefix int, asked bool) error
	RemoveReal(card RealCard, ip string) error
	// Settle waits (a few seconds at most) until the addresses Windows still checks can be used,
	// and returns the ones another device already uses, with the reason.
	Settle(card RealCard, ips []string) map[string]string
}

// handleReal answers a request that names a real card: only approved addresses change, never on
// a card that gets its own address from DHCP, never an address another adapter has.
func handleReal(ops Ops, request Request, version string) Reply {
	reply := Reply{Version: version, Addresses: []string{}, Card: request.Card}
	real, ok := ops.(RealOps)
	if !ok {
		reply.Message = "This service does not change real network cards."
		return reply
	}
	cards, all, err := real.Cards()
	if err != nil {
		reply.Message = err.Error()
		return reply
	}
	guid := strings.ToUpper(request.Card)
	card, found := cards[guid]
	if !found {
		reply.Message = "That network card is not on this PC."
		return reply
	}
	own := all[guid]
	if request.Op == OpStatus {
		reply.OK = true
		return answered(reply, own)
	}
	approved := real.Approved()[guid]
	answers := map[string]bool{}
	if request.Op == OpAdd {
		answers = real.Ask(card, missing(request.Wanted(), approved, own))
	}
	failed, added := map[string]string{}, []string{}
	for _, wanted := range request.Wanted() {
		ip, err := ParseAddress(wanted)
		if err == nil {
			own, err = changeReal(real, card, request.Op, ip, approved, own, all, answers)
		}
		if err != nil {
			failed[wanted] = err.Error()
		} else if _, asked := answers[ip]; request.Op == OpAdd && !asked {
			added = append(added, ip) // Windows checks it; wait until it can be used
		}
	}
	for ip, reason := range real.Settle(card, added) {
		failed[ip], own = reason, without(own, ip)
	}
	reply.OK = len(failed) == 0
	for _, wanted := range request.Wanted() {
		if reason, bad := failed[wanted]; bad && reply.Message == "" {
			reply.Message = reason
		}
	}
	if len(failed) > 0 {
		reply.Failed = failed
	}
	return answered(reply, own)
}

// missing is the approved addresses of the request that are not on the card yet, with prefixes.
func missing(wanted []string, approved map[string]int, own []string) map[string]int {
	found := map[string]int{}
	for _, ip := range wanted {
		if prefix := approved[ip]; prefix > 0 && !contains(own, ip) {
			found[ip] = prefix
		}
	}
	return found
}

func changeReal(real RealOps, card RealCard, op, ip string, approved map[string]int, own []string,
	all map[string][]string, answers map[string]bool) ([]string, error) {
	prefix, allowed := approved[ip]
	present := contains(own, ip)
	switch {
	case !allowed:
		return own, errors.New(ip + " is not approved for “" + card.Name + "”; apply the network plan again")
	case op == OpAdd && !present:
		if card.DHCP {
			return own, errors.New("“" + card.Name + "” gets its address automatically (DHCP); nothing was changed")
		}
		if other := owner(all, card.GUID, ip); other != "" {
			return own, errors.New(ip + " already belongs to another card of this PC; nothing was changed")
		}
		answered, asked := answers[ip]
		if answered {
			return own, errors.New("A device on “" + card.Name + "” already answers at " + ip + "; nothing was changed")
		}
		if err := real.AddReal(card, ip, prefix, asked); err != nil {
			return own, errors.New("Windows did not add " + ip + ": " + err.Error())
		}
		return append(own, ip), nil
	case op == OpRemove && present:
		if err := real.RemoveReal(card, ip); err != nil {
			return own, errors.New("Windows did not remove " + ip + ": " + err.Error())
		}
		return without(own, ip), nil
	}
	return own, nil
}
