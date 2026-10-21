//go:build windows

package hostkey

import (
	"errors"
	"net"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	gaaSkipAnycast    = 0x2
	gaaSkipMulticast  = 0x4
	gaaSkipDNSServer  = 0x8
	gaaIncludeGateway = 0x80
	ifOperUp          = 1
)

// SystemAdapters lists this PC's IPv4 adapters through the IP Helper API (one call, no
// PowerShell): friendly name, description, status, addresses with masks and default gateways.
func SystemAdapters() ([]Adapter, error) {
	flags := uint32(gaaSkipAnycast | gaaSkipMulticast | gaaSkipDNSServer | gaaIncludeGateway)
	size := uint32(32 * 1024)
	var buffer []byte
	for attempt := 0; attempt < 4; attempt++ {
		buffer = make([]byte, size)
		head := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buffer[0]))
		err := windows.GetAdaptersAddresses(windows.AF_INET, flags, 0, head, &size)
		if err == nil {
			return readAdapters(head), nil
		}
		if !errors.Is(err, windows.ERROR_BUFFER_OVERFLOW) {
			return nil, err
		}
	}
	return nil, errors.New("the adapter list kept changing while it was read")
}

func readAdapters(head *windows.IpAdapterAddresses) []Adapter {
	found := []Adapter{}
	for adapter := head; adapter != nil; adapter = adapter.Next {
		item := Adapter{Name: windows.UTF16PtrToString(adapter.FriendlyName),
			Description: windows.UTF16PtrToString(adapter.Description), Up: adapter.OperStatus == ifOperUp}
		for unicast := adapter.FirstUnicastAddress; unicast != nil; unicast = unicast.Next {
			if ip := socketIP(unicast.Address); ip != nil {
				item.IPv4 = append(item.IPv4, net.IPNet{IP: ip, Mask: net.CIDRMask(int(unicast.OnLinkPrefixLength), 32)})
			}
		}
		for gateway := adapter.FirstGatewayAddress; gateway != nil; gateway = gateway.Next {
			if ip := socketIP(gateway.Address); ip != nil && !ip.IsUnspecified() {
				item.Gateways = append(item.Gateways, ip)
			}
		}
		found = append(found, item)
	}
	return found
}

// socketIP is the IPv4 address of a SOCKADDR, or nil for any other family.
func socketIP(address windows.SocketAddress) net.IP {
	if address.Sockaddr == nil || address.SockaddrLength < 8 {
		return nil
	}
	raw := (*windows.RawSockaddrAny)(unsafe.Pointer(address.Sockaddr))
	if raw.Addr.Family != windows.AF_INET {
		return nil
	}
	ipv4 := (*windows.RawSockaddrInet4)(unsafe.Pointer(raw))
	return net.IPv4(ipv4.Addr[0], ipv4.Addr[1], ipv4.Addr[2], ipv4.Addr[3]).To4()
}
