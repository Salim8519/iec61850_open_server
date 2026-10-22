package netcfg

import "strings"

// Typed (manual) addresses in Network setup. This is a Go-only fix that intentionally differs
// from Python (owner decision, 2026-10-22; Python is frozen): Python forced every valid typed
// address to "ok", so an address another IED, this PC's other adapter or a second typed row
// already used still showed "Free" and the apply button stayed on. Here a typed address gets the
// same checks as a planned one, in memory only: no ping, ARP or Windows call is added.

// typedAddress is the validated typed value of one row; err is why it was refused.
type typedAddress struct {
	ip     string
	prefix int
	err    error
}

// trimTyped removes the spaces around a typed text ("172.16.1.20 " is the address 172.16.1.20);
// a value that is not text stays exactly as it was.
func trimTyped(value any) any {
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return value
}

// typedAddresses reads and validates the typed address of every row that has one.
func typedAddresses(ieds []PlanIED, manual any) (map[string]typedAddress, error) {
	typed := map[string]typedAddress{}
	for _, ied := range ieds {
		override, err := manualOverride(manual, ied.ID)
		if err != nil {
			return nil, err
		}
		if override == nil {
			continue
		}
		ip, prefix, err := ValidateManual(trimTyped(get(override, "ip")), trimTyped(get(override, "prefix")))
		typed[ied.ID] = typedAddress{ip: ip, prefix: prefix, err: err}
	}
	return typed, nil
}

// avoidTyped adds the valid typed addresses to avoid, so a planned row never picks one of them.
func avoidTyped(avoid map[string]bool, typed map[string]typedAddress) map[string]bool {
	if len(typed) == 0 {
		return avoid
	}
	extended := copySet(avoid)
	for _, item := range typed {
		if item.err == nil {
			extended[item.ip] = true
		}
	}
	return extended
}

// claim records the first row that holds an address (the planner's "chosen").
func (s planState) claim(row Row) {
	if row.Proposed == "" {
		return
	}
	if _, found := s.chosen[row.Proposed]; !found {
		s.chosen[row.Proposed] = row.Name
	}
}

// checkTyped is check for a typed address: an address another row already holds is a duplicate
// even when it is the adapter's own, then the planner's own checks (this PC, another adapter).
func (s planState) checkTyped(row *Row) {
	if row.Proposed == "" || row.Status != "ok" {
		return
	}
	if name, found := s.chosen[row.Proposed]; found {
		noun := "IED"
		if isSNMPRow(*row) {
			noun = "device"
		}
		row.Status = "duplicate"
		row.Hint = name + " already uses " + row.Proposed + ". Each " + noun + " needs its own address."
		return
	}
	s.check(row)
}

// applyTyped puts the typed addresses on the rows and checks them. Rows nobody typed for keep
// the planner's answer, except that a "duplicate" caused by a pick the typed address replaced is
// cleared. Planned rows hold their address first, then typed rows in list order.
func (s planState) applyTyped(rows []Row, typed map[string]typedAddress) {
	if len(typed) == 0 {
		return
	}
	for index := range rows {
		item, found := typed[rows[index].IEDID]
		switch {
		case !found:
		case item.err != nil:
			rows[index].Status, rows[index].Hint = "invalid_manual", item.err.Error()
		default:
			rows[index].Proposed, rows[index].Prefix = item.ip, intPointer(item.prefix)
			rows[index].Status, rows[index].Hint = "ok", "Address typed by you."
		}
	}
	for index := range rows {
		if _, found := typed[rows[index].IEDID]; found {
			continue
		}
		if rows[index].Status == "duplicate" {
			rows[index].Status, rows[index].Hint = "ok", ""
		}
		s.check(&rows[index])
		s.claim(rows[index])
	}
	for index := range rows {
		if item, found := typed[rows[index].IEDID]; found && item.err == nil {
			s.checkTyped(&rows[index])
			s.claim(rows[index])
		}
	}
}
