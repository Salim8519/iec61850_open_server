package labsetup

import (
	"strings"
)

// Every address that could not be put on or taken off a card is reported with its own reason
// (change 0221, trial N1): before, Run went on after a refused lab address and showed only the
// first refused real-card address, and Stop said nothing at all.

// failure is one address that did not change, whose it is and why (the service's or Windows'
// words).
type failure struct {
	address, name, card, reason string
}

// sentence is a reason as the end of a sentence: trimmed, without its own full stop.
func sentence(reason string) string {
	return strings.TrimRight(strings.TrimSpace(reason), ".")
}

// runRefused is the Run's message for the addresses that could not go on: every one in project
// order with its reason, then what to do. nil when nothing failed.
func runRefused(failures []failure) error {
	if len(failures) == 0 {
		return nil
	}
	parts, duplicate, ownAdvice := make([]string, len(failures)), false, true
	for index, item := range failures {
		parts[index] = item.name + " cannot start on " + item.address
		if reason := sentence(item.reason); reason != "" {
			parts[index] += ": " + reason
		}
		duplicate = duplicate || strings.Contains(item.reason, "already")
		ownAdvice = ownAdvice && strings.Contains(item.reason, "Start again")
	}
	text := strings.Join(parts, ". ") + "."
	switch {
	case duplicate && len(failures) == 1:
		text += " Stop the other device that uses this address, or give " + failures[0].name +
			" another address in Network > Set up lab network."
	case duplicate:
		text += " Stop the other device that uses an address, or give that IED another address in " +
			"Network > Set up lab network."
	case !ownAdvice:
		text += " Press Start again. If it keeps failing, press Repair on Network > This PC lab network."
	}
	return plain(text)
}

// serviceDownForRun is the Run's message when the lab network service is off or does not answer
// and some starting IEDs' addresses are not on the card.
func serviceDownForRun(missing []wanted, why string) error {
	if len(missing) == 0 {
		return nil
	}
	return plain("The lab network service " + why + ", so ScadaBench could not put " + listed(missing) +
		" on the ScadaBench Lab card. Press Repair on Network > This PC lab network (Windows asks once), " +
		"then press Start again.")
}

// listed names addresses with their owners: "192.168.216.1 (P545) and 10.20.0.5 (D60)".
func listed(items []wanted) string {
	parts := make([]string, len(items))
	for index, item := range items {
		parts[index] = item.address
		if item.name != "" {
			parts[index] += " (" + item.name + ")"
		}
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
}

// releaseLeft is the message after Stop (and at launch and exit) for addresses that stayed on a
// card. nil when everything that should go went.
func releaseLeft(failures []failure) error {
	if len(failures) == 0 {
		return nil
	}
	parts := make([]string, len(failures))
	for index, item := range failures {
		parts[index] = item.address + " on " + item.card
		if reason := sentence(item.reason); reason != "" {
			parts[index] += " (" + reason + ")"
		}
	}
	return plain("ScadaBench could not take these addresses off their network card: " + strings.Join(parts, "; ") +
		". It tries again at the next Stop and when ScadaBench closes; a Windows restart also removes them. " +
		"If this keeps happening, press Repair on Network > This PC lab network.")
}
