//go:build windows

package labnet

import (
	"errors"
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

// Read takes the GUID the installer stored, lists the adapters once and checks that this KM-TEST
// card exists now.
func (WindowsOps) Read() (View, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, ParametersKey, registry.QUERY_VALUE)
	if err != nil {
		return View{}, ErrNoCard
	}
	defer key.Close()
	guid, _, err := key.GetStringValue(CardValue)
	if err != nil || guid == "" {
		return View{}, ErrNoCard
	}
	adapters, err := ListAdapters()
	if err != nil {
		return View{}, errors.New("Windows did not list the network addresses: " + err.Error())
	}
	view := View{Addresses: map[string][]string{}, Names: map[string]string{}}
	for _, adapter := range adapters {
		view.Addresses[adapter.GUID], view.Names[adapter.GUID] = adapter.Addresses(), adapter.Name
		if strings.EqualFold(adapter.GUID, guid) && IsCard(adapter.Description) {
			view.Card, view.Index, view.LUID = adapter.GUID, adapter.Index, adapter.LUID
		}
	}
	if view.Card == "" {
		return View{}, ErrNoCard
	}
	return view, nil
}

// Card is the lab card's GUID when it exists now (Install keeps an existing card).
func (ops WindowsOps) Card() (string, error) {
	view, err := ops.Read()
	return view.Card, err
}

// Add puts ip/32 on the card: preferred at once (no duplicate-address wait), usable as a
// source address so ping and clients without an explicit bind work (ADR-0120).
func (WindowsOps) Add(view View, ip string) error {
	row, err := rowFor(view, ip)
	if err != nil {
		return err
	}
	return status(procCreateUnicastIpAddressEntry.Call(uintptr(unsafe.Pointer(row))))
}

// Remove takes ip off the card.
func (WindowsOps) Remove(view View, ip string) error {
	row, err := rowFor(view, ip)
	if err != nil {
		return err
	}
	return status(procDeleteUnicastIpAddressEntry.Call(uintptr(unsafe.Pointer(row))))
}

func rowFor(view View, ip string) (*windows.MibUnicastIpAddressRow, error) {
	address, err := netip.ParseAddr(ip)
	if err != nil || !address.Is4() {
		return nil, ErrAddress
	}
	row := &windows.MibUnicastIpAddressRow{}
	procInitializeUnicastIpAddressEntry.Call(uintptr(unsafe.Pointer(row)))
	inet := (*windows.RawSockaddrInet4)(unsafe.Pointer(&row.Address))
	inet.Family = windows.AF_INET
	inet.Addr = address.As4()
	row.InterfaceLuid, row.InterfaceIndex = view.LUID, view.Index
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
