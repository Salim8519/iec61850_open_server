package netcfg

// StatusAdapterDHCP marks an address ScadaBench will not add because the card gets its own
// address automatically (ADR-0131).
const StatusAdapterDHCP = "adapter_uses_dhcp"

// guardDHCP keeps fixed addresses off a card that gets its address from DHCP. Windows turns DHCP
// off when a fixed address is added, so the card (often Wi-Fi) loses its own address and its
// connection; the owner lost Wi-Fi that way. Every address ScadaBench would add becomes unusable,
// so Apply stays off. Addresses already on the card and rows with another problem are kept.
func (s *NetworkService) guardDHCP(rows []Row, adapter Adapter) error {
	if s.DHCP == nil {
		return nil
	}
	automatic, err := s.DHCP(adapter.GUID)
	if err != nil || !automatic {
		return err
	}
	for index, row := range rows {
		if row.Status == "ok" {
			rows[index].Status = StatusAdapterDHCP
			rows[index].Hint = "\"" + adapter.Name + "\" gets its address automatically (DHCP). A fixed " +
				"address would switch that off and \"" + adapter.Name + "\" would lose its connection. Pick " +
				"ScadaBench Lab or a card with a fixed address."
		}
	}
	return nil
}
