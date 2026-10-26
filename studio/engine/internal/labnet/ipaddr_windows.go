//go:build windows

package labnet

import (
	"net/netip"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	iphlpapi                            = windows.NewLazySystemDLL("iphlpapi.dll")
	procInitializeUnicastIpAddressEntry = iphlpapi.NewProc("InitializeUnicastIpAddressEntry")
	procCreateUnicastIpAddressEntry     = iphlpapi.NewProc("CreateUnicastIpAddressEntry")
	procDeleteUnicastIpAddressEntry     = iphlpapi.NewProc("DeleteUnicastIpAddressEntry")
)

const (
	ipPrefixOriginManual = 1
	ipSuffixOriginManual = 1
	ipDadStatePreferred  = 4
	infinite             = 0xFFFFFFFF
)

// WindowsOps is the service's real Ops: the card from its own registry key, addresses through
// the IP Helper API. CreateUnicastIpAddressEntry keeps an address until Windows restarts.
type WindowsOps struct{}

// Card reads the GUID the installer stored and checks that this KM-TEST card exists now.
func (WindowsOps) Card() (string, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, ParametersKey, registry.QUERY_VALUE)
	if err != nil {
		return "", ErrNoCard
	}
	defer key.Close()
	guid, _, err := key.GetStringValue(CardValue)
	if err != nil || guid == "" {
		return "", ErrNoCard
	}
	adapter, err := find(guid)
	if err != nil {
		return "", err
	}
	if !IsCard(adapter.Description) {
		return "", ErrNoCard
	}
	return adapter.GUID, nil
}

// Addresses lists IPv4 addresses by adapter GUID.
func (WindowsOps) Addresses() (map[string][]string, error) {
	adapters, err := ListAdapters()
	if err != nil {
		return nil, err
	}
	all := map[string][]string{}
	for _, adapter := range adapters {
		all[adapter.GUID] = adapter.Addresses()
	}
	return all, nil
}

// Names maps GUIDs to Windows adapter names.
func (WindowsOps) Names() map[string]string {
	names := map[string]string{}
	adapters, _ := ListAdapters()
	for _, adapter := range adapters {
		names[adapter.GUID] = adapter.Name
	}
	return names
}

// Add puts ip/32 on the card: preferred at once (no duplicate-address wait), usable as a
// source address so ping and clients without an explicit bind work (ADR-0120).
func (WindowsOps) Add(card, ip string) error {
	row, err := rowFor(card, ip)
	if err != nil {
		return err
	}
	return status(procCreateUnicastIpAddressEntry.Call(uintptr(unsafe.Pointer(row))))
}

// Remove takes ip off the card.
func (WindowsOps) Remove(card, ip string) error {
	row, err := rowFor(card, ip)
	if err != nil {
		return err
	}
	return status(procDeleteUnicastIpAddressEntry.Call(uintptr(unsafe.Pointer(row))))
}

func rowFor(card, ip string) (*windows.MibUnicastIpAddressRow, error) {
	adapter, err := find(card)
	if err != nil {
		return nil, err
	}
	address, err := netip.ParseAddr(ip)
	if err != nil || !address.Is4() {
		return nil, ErrAddress
	}
	row := &windows.MibUnicastIpAddressRow{}
	procInitializeUnicastIpAddressEntry.Call(uintptr(unsafe.Pointer(row)))
	inet := (*windows.RawSockaddrInet4)(unsafe.Pointer(&row.Address))
	inet.Family = windows.AF_INET
	inet.Addr = address.As4()
	row.InterfaceLuid, row.InterfaceIndex = adapter.LUID, adapter.Index
	row.PrefixOrigin, row.SuffixOrigin = ipPrefixOriginManual, ipSuffixOriginManual
	row.ValidLifetime, row.PreferredLifetime = infinite, infinite
	row.OnLinkPrefixLength, row.SkipAsSource, row.DadState = 32, 0, ipDadStatePreferred
	return row, nil
}

func status(code, _ uintptr, _ error) error {
	if code == 0 {
		return nil
	}
	return windows.Errno(code)
}

// find returns the adapter with this GUID.
func find(guid string) (Adapter, error) {
	adapters, err := ListAdapters()
	if err != nil {
		return Adapter{}, err
	}
	for _, adapter := range adapters {
		if strings.EqualFold(adapter.GUID, guid) {
			return adapter, nil
		}
	}
	return Adapter{}, ErrNoCard
}
