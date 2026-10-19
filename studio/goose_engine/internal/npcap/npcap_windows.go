// Package npcap binds the few wpcap.dll functions the uplink needs, without cgo.
// Packet.dll is loaded from the Npcap folder first so an old WinPcap copy in
// System32 can never be picked up by wpcap.dll's import (seen on real PCs).
package npcap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"
)

const (
	openPromiscuous    = 1
	openNoCaptureLocal = 8
	readTimeoutMs      = 1
	snapLength         = 65536
	gooseFilter        = "ether proto 0x88b8 or (vlan and ether proto 0x88b8)"
)

var (
	loadOnce  sync.Once
	loadError error
	procs     = map[string]*syscall.Proc{}
)

// ErrMissing means Npcap is not installed; GOOSE stays inside the app.
var ErrMissing = errors.New("Npcap is not installed, so GOOSE cannot reach the network")

func load() {
	folder := filepath.Join(os.Getenv("SystemRoot"), "System32", "Npcap")
	if _, err := os.Stat(filepath.Join(folder, "wpcap.dll")); err != nil {
		loadError = ErrMissing
		return
	}
	if _, err := syscall.LoadDLL(filepath.Join(folder, "Packet.dll")); err != nil {
		loadError = fmt.Errorf("Npcap is installed but Packet.dll could not be loaded: %w", err)
		return
	}
	wpcap, err := syscall.LoadDLL(filepath.Join(folder, "wpcap.dll"))
	if err != nil {
		loadError = fmt.Errorf("Npcap is installed but wpcap.dll could not be loaded: %w", err)
		return
	}
	for _, name := range []string{"pcap_open", "pcap_next_ex", "pcap_sendpacket", "pcap_compile",
		"pcap_setfilter", "pcap_freecode", "pcap_setmintocopy", "pcap_geterr", "pcap_close"} {
		proc, err := wpcap.FindProc(name)
		if err != nil {
			loadError = fmt.Errorf("Npcap is too old: %s is missing", name)
			return
		}
		procs[name] = proc
	}
}

// Available loads Npcap once and reports why it cannot be used.
func Available() error {
	loadOnce.Do(load)
	return loadError
}

// Handle is one open adapter. It must be used from a single goroutine.
type Handle struct{ p uintptr }

type bpfProgram struct {
	length uint32
	insns  uintptr
}

// Open opens the adapter by Windows GUID ({...}) for GOOSE only. The handle never
// captures its own injections, so the engine cannot hear itself.
func Open(guid string) (*Handle, error) {
	if err := Available(); err != nil {
		return nil, err
	}
	source, _ := syscall.BytePtrFromString(`rpcap://\Device\NPF_` + guid)
	errbuf := make([]byte, 512)
	p, _, _ := procs["pcap_open"].Call(uintptr(unsafe.Pointer(source)), snapLength,
		openPromiscuous|openNoCaptureLocal, readTimeoutMs, 0, uintptr(unsafe.Pointer(&errbuf[0])))
	if p == 0 {
		return nil, fmt.Errorf("could not open the adapter for GOOSE: %s", cString(errbuf))
	}
	h := &Handle{p: p}
	filter, _ := syscall.BytePtrFromString(gooseFilter)
	var program bpfProgram
	if r, _, _ := procs["pcap_compile"].Call(p, uintptr(unsafe.Pointer(&program)), uintptr(unsafe.Pointer(filter)), 1, 0xFFFFFFFF); r != 0 {
		h.Close()
		return nil, fmt.Errorf("GOOSE filter failed: %s", h.lastError())
	}
	r, _, _ := procs["pcap_setfilter"].Call(p, uintptr(unsafe.Pointer(&program)))
	procs["pcap_freecode"].Call(uintptr(unsafe.Pointer(&program)))
	if r != 0 {
		h.Close()
		return nil, errors.New("GOOSE filter could not be applied")
	}
	procs["pcap_setmintocopy"].Call(p, 0)
	return h, nil
}

// Next waits up to 1 ms for a frame. It returns nil, nil on timeout.
func (h *Handle) Next() ([]byte, error) {
	var header, data uintptr
	r, _, _ := procs["pcap_next_ex"].Call(h.p, uintptr(unsafe.Pointer(&header)), uintptr(unsafe.Pointer(&data)))
	switch int32(r) {
	case 1:
		// struct pcap_pkthdr on Windows: timeval (two 32-bit longs), caplen, len.
		caplen := *(*uint32)(foreign(header + 8))
		return append([]byte(nil), unsafe.Slice((*byte)(foreign(data)), caplen)...), nil
	case 0:
		return nil, nil
	default:
		return nil, fmt.Errorf("adapter stopped delivering frames: %s", h.lastError())
	}
}

// Send injects one frame on the adapter.
func (h *Handle) Send(b []byte) error {
	if len(b) == 0 {
		return errors.New("empty frame")
	}
	r, _, _ := procs["pcap_sendpacket"].Call(h.p, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
	if r != 0 {
		return errors.New(h.lastError())
	}
	return nil
}

// Close releases the adapter.
func (h *Handle) Close() {
	if h.p != 0 {
		procs["pcap_close"].Call(h.p)
		h.p = 0
	}
}

func (h *Handle) lastError() string {
	r, _, _ := procs["pcap_geterr"].Call(h.p)
	if r == 0 {
		return "unknown Npcap error"
	}
	return cString(unsafe.Slice((*byte)(foreign(r)), errorBufferSize))
}

// errorBufferSize is PCAP_ERRBUF_SIZE, the size of the text pcap_geterr points at.
const errorBufferSize = 256

// foreign turns an address returned by wpcap.dll into a pointer. The memory is owned
// by Npcap, not the Go heap, so the garbage collector never moves or frees it.
func foreign(address uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&address))
}

func cString(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}
