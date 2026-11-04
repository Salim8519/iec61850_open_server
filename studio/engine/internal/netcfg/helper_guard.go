package netcfg

// StatusHelperOff marks an address the plan cannot put on a real card because the ScadaBench
// network service cannot add it now (ADR-0136).
const StatusHelperOff = "helper_off"

// labKind is the ScadaBench Lab card's adapter kind; its addresses come from the lab page.
const labKind = "virtual_lab"

// guardHelper: on a real card only the network service adds addresses, at Start, and takes them
// off at Stop, exit and the next launch (ADR-0136). While it is missing, stopped or too old the
// plan has no way to add an address that will not be left behind, so every new address waits for
// the user to turn the helper on; an address already on the card stays usable.
func (s *NetworkService) guardHelper(rows []Row, adapter Adapter) {
	if s.RealCards == nil || adapter.Kind == labKind || s.RealCards(adapter.GUID) {
		return
	}
	for index, row := range rows {
		if row.Status == "ok" {
			rows[index].Status = StatusHelperOff
			rows[index].Hint = "The ScadaBench network helper is not running, so this address could not be " +
				"taken off again at Stop. Open Network › This PC lab network and press Repair (one Windows " +
				"prompt), or Create the lab network card if there is none yet. Then review again."
		}
	}
}
