//go:build windows

package labnet

import (
	"errors"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// settleWait bounds how long Settle waits for Windows' duplicate-address check on a connected
// card; all addresses of one request are checked together.
const settleWait = 6 * time.Second

// Cards lists every adapter as a real card, by upper-case GUID, and every adapter's addresses
// (ADR-0132). The lab card is listed too; nothing is ever approved for it.
func (WindowsOps) Cards() (map[string]RealCard, map[string][]string, error) {
	adapters, err := ListAdapters()
	if err != nil {
		return nil, nil, errors.New("Windows did not list the network addresses: " + err.Error())
	}
	cards, all := map[string]RealCard{}, map[string][]string{}
	for _, adapter := range adapters {
		cards[adapter.GUID] = RealCard{GUID: adapter.GUID, Name: adapter.Name, Index: adapter.Index,
			LUID: adapter.LUID, Up: adapter.Up, DHCP: adapter.DHCP, Own: adapter.Networks}
		all[adapter.GUID] = adapter.Addresses()
	}
	return cards, all, nil
}

// ReadApproved is the approved real-card addresses (the app reads the same key the service does;
// anyone may read it, only administrators may write it).
func ReadApproved() map[string]map[string]int { return WindowsOps{}.Approved() }

// Approved reads the service's Approved key: one DWORD value per address, named
// "{card GUID} address", holding the prefix length. Anything else there is ignored.
func (WindowsOps) Approved() map[string]map[string]int {
	approved := map[string]map[string]int{}
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, ApprovedKey, registry.QUERY_VALUE)
	if err != nil {
		return approved
	}
	defer key.Close()
	names, _ := key.ReadValueNames(0)
	for _, name := range names {
		guid, address, found := strings.Cut(name, " ")
		prefix, _, err := key.GetIntegerValue(name)
		ip, bad := ParseAddress(address)
		if !found || !isGUID(guid) || err != nil || bad != nil || ip != address || prefix < 8 || prefix > 32 {
			continue
		}
		guid = strings.ToUpper(guid)
		if approved[guid] == nil {
			approved[guid] = map[string]int{}
		}
		approved[guid][address] = int(prefix)
	}
	return approved
}

// askWait bounds Ask: a device on the same network answers ARP within milliseconds; only "nobody
// there" takes Windows seconds, and that answer is the one Ask does not wait for.
const askWait = 700 * time.Millisecond

// Ask sends one ARP question per address from the card's own address on that network, all at
// once, and waits askWait at most (ADR-0132). Windows' own duplicate check takes 3.3–3.5 s.
func (WindowsOps) Ask(card RealCard, prefixes map[string]int) map[string]bool {
	answers := map[string]bool{}
	if !card.Up {
		return answers
	}
	type answer struct {
		ip  string
		yes bool
	}
	results := make(chan answer, len(prefixes))
	for ip := range prefixes {
		if source := sourceOn(card, ip); source != "" {
			answers[ip] = false
			go func(ip, source string) { results <- answer{ip, AskOn(ip, source)} }(ip, source)
		}
	}
	deadline := time.After(askWait)
	for pending := len(answers); pending > 0; pending-- {
		select {
		case result := <-results:
			answers[result.ip] = result.yes
		case <-deadline:
			return answers // nobody answered in time: free
		}
	}
	return answers
}

// sourceOn is the card's own address on the network of ip ("" when it has none there).
func sourceOn(card RealCard, ip string) string {
	for _, network := range card.Own {
		own, prefixText, _ := strings.Cut(network, "/")
		prefix, err := strconv.Atoi(prefixText)
		if err == nil && !InNetwork(own, "169.254.0.0", 16) && InNetwork(ip, own, prefix) {
			return own
		}
	}
	return ""
}

// AddReal puts address/prefix on the card as the network helper did (manual, never chosen as a
// source address). Asked (nobody answered just now) it is ready at once; otherwise Windows checks
// the network first, as for any new address, and a real device keeps its address.
func (WindowsOps) AddReal(card RealCard, ip string, prefix int, asked bool) error {
	row, err := realRow(card, ip)
	if err != nil {
		return err
	}
	row.PrefixOrigin, row.SuffixOrigin = ipPrefixOriginManual, ipSuffixOriginManual
	row.ValidLifetime, row.PreferredLifetime = infinite, infinite
	row.OnLinkPrefixLength, row.SkipAsSource = uint8(prefix), 1
	if asked {
		row.DadState = ipDadStatePreferred
	}
	return status(procCreateUnicastIpAddressEntry.Call(uintptr(unsafe.Pointer(row))))
}

// RemoveReal takes the address off the card.
func (WindowsOps) RemoveReal(card RealCard, ip string) error {
	row, err := realRow(card, ip)
	if err != nil {
		return err
	}
	return status(procDeleteUnicastIpAddressEntry.Call(uintptr(unsafe.Pointer(row))))
}

// Settle waits while Windows checks the added addresses (settle: a duplicate, and an address
// Windows is still checking when the wait ends, are taken off again and reported). A card without
// a link gets no address at all (handleReal), so nothing waits for a cable (change 0221).
func (ops WindowsOps) Settle(card RealCard, ips []string) map[string]string {
	return settle(ops, card, ips, settleWait, time.Now, time.Sleep)
}

// Duplicate says whether Windows' duplicate check found another device at ip.
func (WindowsOps) Duplicate(card RealCard, ip string) bool {
	return dadState(card, ip) == windows.IpDadStateDuplicate
}

// Usable is usable, for settle.
func (WindowsOps) Usable(ip string) bool { return usable(ip) }

// usable says whether a program can listen on the address now, which is what the IED workers need:
// a tentative address refuses (WSAEADDRNOTAVAIL).
func usable(ip string) bool {
	listener, err := net.ListenPacket("udp4", ip+":0")
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

func dadState(card RealCard, ip string) uint32 {
	row, err := realRow(card, ip)
	if err != nil || windows.GetUnicastIpAddressEntry(row) != nil {
		return 0
	}
	return row.DadState
}

func realRow(card RealCard, ip string) (*windows.MibUnicastIpAddressRow, error) {
	address, err := netip.ParseAddr(ip)
	if err != nil || !address.Is4() {
		return nil, ErrAddress
	}
	row := &windows.MibUnicastIpAddressRow{}
	procInitializeUnicastIpAddressEntry.Call(uintptr(unsafe.Pointer(row)))
	inet := (*windows.RawSockaddrInet4)(unsafe.Pointer(&row.Address))
	inet.Family = windows.AF_INET
	inet.Addr = address.As4()
	row.InterfaceLuid, row.InterfaceIndex = card.LUID, card.Index
	return row, nil
}
