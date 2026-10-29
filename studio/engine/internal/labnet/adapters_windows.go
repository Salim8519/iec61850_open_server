//go:build windows

package labnet

import (
	"net/netip"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Adapter is one Windows network adapter as the lab network needs it.
type Adapter struct {
	GUID        string
	Name        string
	Description string
	Index       uint32
	LUID        uint64
	Up          bool
	// Networks are the IPv4 address/prefix pairs, e.g. "192.168.40.111/24".
	Networks []string
}

// Addresses are the adapter's IPv4 addresses without prefixes.
func (a Adapter) Addresses() []string {
	addresses := make([]string, 0, len(a.Networks))
	for _, network := range a.Networks {
		address, _, _ := strings.Cut(network, "/")
		addresses = append(addresses, address)
	}
	return addresses
}

// gaaSkipDNSInfo is GAA_FLAG_SKIP_DNS_INFO (iptypes.h; not in x/sys): without it every listing
// asks the Windows DNS Client about every adapter, which nothing here reads (ADR-0128).
const gaaSkipDNSInfo = 0x800

// listFlags are what every listing asks for: unicast IPv4 addresses, names and states only.
const listFlags = windows.GAA_FLAG_SKIP_ANYCAST | windows.GAA_FLAG_SKIP_MULTICAST |
	windows.GAA_FLAG_SKIP_DNS_SERVER | gaaSkipDNSInfo

// ListAdapters reads every adapter with its IPv4 addresses through GetAdaptersAddresses.
func ListAdapters() ([]Adapter, error) { return listAdapters(listFlags) }

func listAdapters(flags uint32) ([]Adapter, error) {
	size := uint32(16 * 1024)
	for attempt := 0; attempt < 4; attempt++ {
		buffer := make([]byte, size)
		first := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buffer[0]))
		err := windows.GetAdaptersAddresses(windows.AF_INET, flags, 0, first, &size)
		if err == windows.ERROR_BUFFER_OVERFLOW {
			continue
		}
		if err != nil {
			return nil, err
		}
		adapters := []Adapter{}
		for item := first; item != nil; item = item.Next {
			adapters = append(adapters, adapterOf(item))
		}
		return adapters, nil
	}
	return nil, windows.ERROR_BUFFER_OVERFLOW
}

func adapterOf(item *windows.IpAdapterAddresses) Adapter {
	adapter := Adapter{GUID: strings.ToUpper(windows.BytePtrToString(item.AdapterName)),
		Name: windows.UTF16PtrToString(item.FriendlyName), Description: windows.UTF16PtrToString(item.Description),
		Index: item.IfIndex, LUID: item.Luid, Up: item.OperStatus == windows.IfOperStatusUp}
	for unicast := item.FirstUnicastAddress; unicast != nil; unicast = unicast.Next {
		raw := unicast.Address.Sockaddr
		if raw == nil || raw.Addr.Family != windows.AF_INET {
			continue
		}
		inet := (*windows.RawSockaddrInet4)(unsafe.Pointer(raw))
		address := netip.AddrFrom4(inet.Addr).String()
		adapter.Networks = append(adapter.Networks, address+"/"+strconv.Itoa(int(unicast.OnLinkPrefixLength)))
	}
	return adapter
}
