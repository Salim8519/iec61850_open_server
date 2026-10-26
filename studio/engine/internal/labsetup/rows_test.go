package labsetup

import (
	"testing"

	"studio-engine/internal/netcfg"
)

const card = "{CARD}"

type fixedAnswers map[[2]string]bool

func (f fixedAnswers) Answer(address, source string) (bool, bool) {
	used, known := f[[2]string{address, source}]
	return used, known
}

func pcAdapters() []netcfg.Adapter {
	return []netcfg.Adapter{
		{GUID: card, Name: "ScadaBench Lab", Kind: "virtual_lab", Status: "Up",
			Addresses: []netcfg.Address{{IP: "192.168.216.1", Prefix: 32}, {IP: "169.254.9.9", Prefix: 16}}},
		{GUID: "{ETH}", Name: "Ethernet", Kind: "ethernet", Status: "Disconnected",
			Addresses: []netcfg.Address{{IP: "192.168.40.111", Prefix: 24}}},
		{GUID: "{WIFI}", Name: "Wi-Fi", Kind: "wifi", Status: "Up",
			Addresses: []netcfg.Address{{IP: "192.168.118.112", Prefix: 20}}},
	}
}

func byID(rows []Row) map[string]Row {
	found := map[string]Row{}
	for _, row := range rows {
		found[row.IEDID] = row
	}
	return found
}

func TestEveryIEDKeepsItsOwnAddress(t *testing.T) {
	ieds := []IED{{ID: "a", Name: "P545", SCLIP: "192.168.216.1"}, {ID: "b", Name: "D60", SCLIP: "10.20.0.5"}}
	rows := byID(Rows(ieds, nil, false, pcAdapters(), card, fixedAnswers{}))
	if rows["a"].Status != "ok" || !rows["a"].OnCard || rows["b"].Status != "ok" || rows["b"].OnCard {
		t.Fatalf("rows %+v", rows)
	}
	if rows["a"].Yours || rows["b"].IP != "10.20.0.5" {
		t.Fatalf("file addresses were not kept: %+v", rows)
	}
}

func TestTypedAndStoredAddressesWin(t *testing.T) {
	ieds := []IED{{ID: "a", Name: "P545", SCLIP: "192.168.216.1", LabIP: "192.168.216.9"}}
	if row := Rows(ieds, nil, true, pcAdapters(), card, fixedAnswers{})[0]; row.IP != "192.168.216.9" || !row.Yours {
		t.Fatalf("stored lab_ip not used while active: %+v", row)
	}
	if row := Rows(ieds, nil, false, pcAdapters(), card, fixedAnswers{})[0]; row.IP != "192.168.216.1" {
		t.Fatalf("lab_ip of another card was used: %+v", row)
	}
	typed := map[string]string{"a": " 10.1.1.1 "}
	if row := Rows(ieds, typed, true, pcAdapters(), card, fixedAnswers{})[0]; row.IP != "10.1.1.1" || row.Status != "ok" {
		t.Fatalf("typed address: %+v", row)
	}
}

func TestBlockingClashes(t *testing.T) {
	ieds := []IED{{ID: "a", Name: "A", SCLIP: "10.0.0.1"}, {ID: "b", Name: "B", SCLIP: "10.0.0.1"},
		{ID: "c", Name: "C", SCLIP: "192.168.118.112"}, {ID: "d", Name: "D", SCLIP: ""},
		{ID: "e", Name: "E", SCLIP: "127.0.0.9"}}
	rows := byID(Rows(ieds, nil, false, pcAdapters(), card, fixedAnswers{}))
	want := map[string]string{"a": "duplicate", "b": "duplicate", "c": "other_card", "d": "missing", "e": "invalid"}
	for id, status := range want {
		if rows[id].Status != status || !rows[id].Blocking() || rows[id].Hint == "" {
			t.Fatalf("%s: %+v, want %s", id, rows[id], status)
		}
	}
}

func TestRealNetworkRanges(t *testing.T) {
	ieds := []IED{{ID: "eth", Name: "E", SCLIP: "192.168.40.5"}, {ID: "free", Name: "F", SCLIP: "192.168.118.200"},
		{ID: "real", Name: "R", SCLIP: "192.168.112.1"}, {ID: "wait", Name: "W", SCLIP: "192.168.118.7"}}
	answers := fixedAnswers{{"192.168.118.200", "192.168.118.112"}: false, {"192.168.112.1", "192.168.118.112"}: true,
		{"192.168.112.130", "192.168.118.112"}: true, {"192.168.112.131", "192.168.118.112"}: false}
	rows := byID(Rows(ieds, nil, false, pcAdapters(), card, answers))
	if rows["eth"].Status != "same_range" || rows["eth"].Card != "Ethernet" {
		t.Fatalf("unplugged Ethernet range: %+v", rows["eth"])
	}
	if rows["free"].Status != "same_range" || rows["free"].Blocking() {
		t.Fatalf("silent Wi-Fi range: %+v", rows["free"])
	}
	if rows["real"].Status != "real_device" || rows["real"].Blocking() || rows["real"].Alternative != "192.168.112.131" {
		t.Fatalf("real device: %+v", rows["real"])
	}
	if rows["wait"].Status != "checking" {
		t.Fatalf("unknown answer: %+v", rows["wait"])
	}
	questions := Questions(Rows(ieds, nil, false, pcAdapters(), card, answers), RealCards(pcAdapters(), card), answers)
	for _, pair := range questions {
		if pair[1] != "192.168.118.112" || pair[0] == "192.168.40.5" {
			t.Fatalf("asked %v: only the connected Wi-Fi may be asked, from its own address", pair)
		}
	}
}

func TestLinkLocalIsNotARealNetwork(t *testing.T) {
	reals := RealCards(pcAdapters(), card)
	if len(reals) != 2 || reals[1].Source != "192.168.118.112" || len(reals[0].Networks) != 1 {
		t.Fatalf("reals %+v", reals)
	}
}
