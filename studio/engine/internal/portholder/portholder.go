// Package portholder finds which program listens on a TCP port, so a failed listen check can
// name it (ADR-0134). It only reads the Windows TCP table; it never changes anything.
package portholder

import (
	"encoding/binary"
	"fmt"
	"net/netip"
)

// Any is the address of a listener on every address of the PC.
const Any = "0.0.0.0"

// Holder is a program listening on a port.
type Holder struct {
	PID     uint32
	Program string // the program file name ("" when Windows does not tell)
	Address string // the address it listens on: one address or Any
}

// Label names the holder for a message: "s7oiehsx64.exe (process 4120)".
func (h Holder) Label() string {
	if h.Program == "" {
		return fmt.Sprintf("the program with process id %d", h.PID)
	}
	return fmt.Sprintf("%s (process %d)", h.Program, h.PID)
}

// listener is one row of the IPv4 listener table.
type listener struct {
	address string
	port    int
	pid     uint32
}

// parseTable reads a MIB_TCPTABLE_OWNER_PID: a row count, then rows of six 32-bit fields (state,
// local address, local port, remote address, remote port, process id). Addresses and ports are
// in network byte order.
func parseTable(table []byte) []listener {
	if len(table) < 4 {
		return nil
	}
	count := int(binary.LittleEndian.Uint32(table))
	rows := make([]listener, 0, count)
	for index := 0; index < count; index++ {
		at := 4 + index*24
		if at+24 > len(table) {
			break
		}
		row := table[at : at+24]
		address := netip.AddrFrom4([4]byte{row[4], row[5], row[6], row[7]})
		rows = append(rows, listener{address: address.String(), port: int(row[8])<<8 | int(row[9]),
			pid: binary.LittleEndian.Uint32(row[20:])})
	}
	return rows
}

// pick is the listener that holds address:port: one on that exact address first, else one on
// every address.
func pick(rows []listener, address string, port int) (listener, bool) {
	var wildcard *listener
	for index, row := range rows {
		if row.port != port {
			continue
		}
		if row.address == address {
			return row, true
		}
		if row.address == Any && wildcard == nil {
			wildcard = &rows[index]
		}
	}
	if wildcard != nil {
		return *wildcard, true
	}
	return listener{}, false
}
