package netcfg

// StatusHelperOff marks an address the plan cannot put on a real card because the ScadaBench
// network service cannot add it now (ADR-0136).
const StatusHelperOff = "helper_off"

// labKind is the ScadaBench Lab card's adapter kind; its addresses come from the lab page.
const labKind = "virtual_lab"

// guardHelper: on a real card only the network service adds addresses, at Start, and takes them
// off at Stop, exit and the next launch (ADR-0136). A service that is missing, stopped or too old
// is set up by the plan's own prompt (service_repair.go). Only when this installation lacks the
// program for that has the plan no way to add an address that will not be left behind: every new
// address then waits, and an address already on the card stays usable.
func (s *NetworkService) guardHelper(rows []Row, adapter Adapter) {
	if s.RealCards == nil || adapter.Kind == labKind || s.RealCards(adapter.GUID) {
		return
	}
	for index, row := range rows {
		if row.Status == "ok" {
			rows[index].Status = StatusHelperOff
			rows[index].Hint = "ScadaBench needs its network service to put this address on at Start and take " +
				"it off at Stop, and the program that installs it is missing. Reinstall ScadaBench."
		}
	}
}
