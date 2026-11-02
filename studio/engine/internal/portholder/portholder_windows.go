//go:build windows

package portholder

import (
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	tcpTableOwnerPIDListener = 3   // TCP_TABLE_OWNER_PID_LISTENER
	errInsufficientBuffer    = 122 // ERROR_INSUFFICIENT_BUFFER
	systemPID                = 4   // the Windows kernel (http.sys and other drivers listen as it)
)

var getExtendedTCPTable = windows.NewLazySystemDLL("iphlpapi.dll").NewProc("GetExtendedTcpTable")

// Find is the program listening on address:port (on that address or on every address).
func Find(address string, port int) (Holder, bool) {
	row, found := pick(listeners(), address, port)
	if !found {
		return Holder{}, false
	}
	return Holder{PID: row.pid, Program: programOf(row.pid), Address: row.address}, true
}

// listeners reads the IPv4 TCP listener table (nil when Windows does not give it).
func listeners() []listener {
	size := uint32(16 * 1024)
	for attempt := 0; attempt < 4; attempt++ {
		table := make([]byte, size)
		code, _, _ := getExtendedTCPTable.Call(uintptr(unsafe.Pointer(&table[0])), uintptr(unsafe.Pointer(&size)),
			0, windows.AF_INET, tcpTableOwnerPIDListener, 0)
		switch code {
		case 0:
			return parseTable(table[:size])
		case errInsufficientBuffer:
			size += 4 * 1024 // the table can grow between the two calls
		default:
			return nil
		}
	}
	return nil
}

// programOf is the file name of the process (Windows itself for the kernel, "" when hidden).
func programOf(pid uint32) string {
	if pid == systemPID {
		return "Windows (System)"
	}
	process, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(process)
	buffer := make([]uint16, windows.MAX_LONG_PATH)
	size := uint32(len(buffer))
	if windows.QueryFullProcessImageName(process, 0, &buffer[0], &size) != nil {
		return ""
	}
	return filepath.Base(windows.UTF16ToString(buffer[:size]))
}
