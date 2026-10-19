//go:build windows

package netcfg

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	"studio-engine/internal/pyjson"
)

// Read-only IP Helper calls (ip_helper.py); nothing here changes adapter settings.
var (
	iphlpapi                     = syscall.NewLazyDLL("iphlpapi.dll")
	procGetAdaptersAddresses     = iphlpapi.NewProc("GetAdaptersAddresses")
	procGetUnicastIpAddressTable = iphlpapi.NewProc("GetUnicastIpAddressTable")
	procFreeMibTable             = iphlpapi.NewProc("FreeMibTable")
)

const errorBufferOverflow = 111

// ipAdapterUnicast is IP_ADAPTER_UNICAST_ADDRESS_LH up to OnLinkPrefixLength.
type ipAdapterUnicast struct {
	Length             uint32
	Flags              uint32
	Next               *ipAdapterUnicast
	Sockaddr           *[8]byte
	SockaddrLength     int32
	PrefixOrigin       int32
	SuffixOrigin       int32
	DadState           int32
	ValidLifetime      uint32
	PreferredLifetime  uint32
	LeaseLifetime      uint32
	OnLinkPrefixLength uint8
}

// ipAdapterAddresses is IP_ADAPTER_ADDRESSES_LH up to FirstGatewayAddress (64-bit layout).
type ipAdapterAddresses struct {
	Length                uint32
	IfIndex               uint32
	Next                  *ipAdapterAddresses
	AdapterName           *byte
	FirstUnicastAddress   *ipAdapterUnicast
	FirstAnycastAddress   uintptr
	FirstMulticastAddress uintptr
	FirstDnsServerAddress uintptr
	DnsSuffix             *uint16
	Description           *uint16
	FriendlyName          *uint16
	PhysicalAddress       [8]byte
	PhysicalAddressLength uint32
	Flags                 uint32
	Mtu                   uint32
	IfType                uint32
	OperStatus            int32
	Ipv6IfIndex           uint32
	ZoneIndices           [16]uint32
	FirstPrefix           uintptr
	TransmitLinkSpeed     uint64
	ReceiveLinkSpeed      uint64
	FirstWinsServer       uintptr
	FirstGatewayAddress   uintptr
}

func adapterBuffer() (*ipAdapterAddresses, error) {
	size := uint32(64 * 1024)
	for attempt := 0; attempt < 3; attempt++ {
		buffer := make([]byte, size)
		result, _, _ := procGetAdaptersAddresses.Call(afInet, gaaFlags, 0,
			uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(&size)))
		if result == 0 {
			return (*ipAdapterAddresses)(unsafe.Pointer(&buffer[0])), nil
		}
		if result != errorBufferOverflow {
			return nil, fmt.Errorf("[WinError %d] GetAdaptersAddresses failed", result)
		}
	}
	return nil, fmt.Errorf("[WinError %d] GetAdaptersAddresses buffer kept growing", errorBufferOverflow)
}

func utf16Text(pointer *uint16) string {
	if pointer == nil {
		return ""
	}
	length := 0
	for *(*uint16)(unsafe.Add(unsafe.Pointer(pointer), length*2)) != 0 {
		length++
	}
	return syscall.UTF16ToString(unsafe.Slice(pointer, length))
}

func ansiText(pointer *byte) string {
	if pointer == nil {
		return ""
	}
	length := 0
	for *(*byte)(unsafe.Add(unsafe.Pointer(pointer), length)) != 0 {
		length++
	}
	return string(unsafe.Slice(pointer, length))
}

func sockaddrIPv4(raw *[8]byte) string {
	return formatIPv4(uint32(raw[4])<<24 | uint32(raw[5])<<16 | uint32(raw[6])<<8 | uint32(raw[7]))
}

type unicastKey struct {
	index uint32
	ip    string
}

// unicastRows is ip_helper.unicast_rows: prefix, origin and SkipAsSource per (ifIndex, ip).
func unicastRows() (map[unicastKey]*pyjson.Object, error) {
	var table unsafe.Pointer
	result, _, _ := procGetUnicastIpAddressTable.Call(afInet, uintptr(unsafe.Pointer(&table)))
	if result != 0 {
		return nil, fmt.Errorf("[WinError %d] GetUnicastIpAddressTable failed", result)
	}
	defer procFreeMibTable.Call(uintptr(table))
	count := *(*uint32)(table)
	found := map[unicastKey]*pyjson.Object{}
	for row := 0; row < int(count); row++ {
		base := unsafe.Add(table, 8+row*80) // MIB_UNICASTIPADDRESS_ROW is 80 bytes on x64
		raw := (*[8]byte)(base)
		key := unicastKey{index: *(*uint32)(unsafe.Add(base, 40)), ip: sockaddrIPv4(raw)}
		found[key] = pyjson.NewObject().Set("prefix", int(*(*uint8)(unsafe.Add(base, 60)))).
			Set("origin", prefixOrigin(*(*int32)(unsafe.Add(base, 44)))).
			Set("skip_as_source", *(*uint8)(unsafe.Add(base, 61)) != 0)
	}
	return found, nil
}

// RawAdapters is ip_helper.adapters: one record per listed adapter, in the shape the
// PowerShell listing produced (normalize them with Normalize).
func RawAdapters() ([]any, error) {
	first, err := adapterBuffer()
	if err != nil {
		return nil, err
	}
	details, err := unicastRows()
	if err != nil {
		return nil, err
	}
	result := []any{}
	for item := first; item != nil; item = item.Next {
		description := utf16Text(item.Description)
		if !IsListed(item.IfType, description) {
			continue
		}
		addresses := []any{}
		for unicast := item.FirstUnicastAddress; unicast != nil; unicast = unicast.Next {
			ip := sockaddrIPv4(unicast.Sockaddr)
			address := pyjson.NewObject().Set("ip", ip)
			extra := details[unicastKey{item.IfIndex, ip}]
			if extra == nil {
				extra = pyjson.NewObject().Set("prefix", int(unicast.OnLinkPrefixLength)).
					Set("skip_as_source", false).Set("origin", prefixOrigin(unicast.PrefixOrigin))
			}
			for _, key := range extra.Keys() {
				value, _ := extra.Get(key)
				address.Set(key, value)
			}
			addresses = append(addresses, address)
		}
		result = append(result, pyjson.NewObject().Set("guid", ansiText(item.AdapterName)).
			Set("name", utf16Text(item.FriendlyName)).Set("description", description).
			Set("status", StatusWord(item.OperStatus)).Set("index", int64(item.IfIndex)).
			Set("if_type", int64(item.IfType)).Set("mac", MACWords(item.PhysicalAddress, item.PhysicalAddressLength)).
			Set("speed", SpeedWords(item.TransmitLinkSpeed)).Set("default_route", item.FirstGatewayAddress != 0).
			Set("addresses", addresses))
	}
	return result, nil
}

// AddressesOn is ip_helper.addresses_on: the IPv4 addresses on the adapter with this GUID.
func AddressesOn(adapterGUID string) (map[string]bool, error) {
	found := map[string]bool{}
	first, err := adapterBuffer()
	if err != nil {
		return found, err
	}
	wanted := strings.ToUpper(adapterGUID)
	for item := first; item != nil; item = item.Next {
		if strings.ToUpper(ansiText(item.AdapterName)) == wanted {
			for unicast := item.FirstUnicastAddress; unicast != nil; unicast = unicast.Next {
				found[sockaddrIPv4(unicast.Sockaddr)] = true
			}
			return found, nil
		}
	}
	return found, nil
}
