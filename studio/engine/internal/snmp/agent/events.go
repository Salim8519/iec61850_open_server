package agent

import (
	"errors"
	"fmt"
	"net/netip"
	"sync"
	"syscall"
	"time"

	"studio-engine/internal/snmp/proto"
)

// Windows socket errors behind a failed bind.
const (
	wsaAccess       = 10013
	wsaAddrInUse    = 10048
	wsaAddrNotAvail = 10049
)

// BindProblem words a failed bind for the user. service names the Windows service that
// usually holds the port ("SNMP" for 161, "SNMP Trap" for 162).
func BindProblem(ip string, port int, err error) string {
	var code syscall.Errno
	if errors.As(err, &code) {
		switch int(code) {
		case wsaAddrInUse:
			return fmt.Sprintf("Port %d on %s is already in use. The Windows %s service or another SNMP program "+
				"may hold it; choose another port such as %d.", port, ip, serviceFor(port), port+1000)
		case wsaAddrNotAvail:
			return fmt.Sprintf("%s is not an address of this PC. Use a 127.x.x.x address (this PC only) "+
				"or an address of one of its network cards.", ip)
		case wsaAccess:
			return fmt.Sprintf("Windows does not allow port %d on %s (it may be reserved); choose another port "+
				"such as %d.", port, ip, port+1000)
		}
	}
	return fmt.Sprintf("Cannot open UDP port %d on %s: %v", port, ip, err)
}

func serviceFor(port int) string {
	if port == 162 {
		return "SNMP Trap"
	}
	return "SNMP"
}

// limiter lets at most perSecond request events through each second and counts the rest,
// so a manager polling hard cannot flood the page's log.
type limiter struct {
	mu      sync.Mutex
	now     func() time.Time
	window  time.Time
	used    int
	dropped int
}

const perSecond = 20

func newLimiter(now func() time.Time) *limiter { return &limiter{now: now} }

// allow returns whether to log, and how many events were held back before this one.
func (l *limiter) allow() (bool, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if now.Sub(l.window) >= time.Second {
		l.window, l.used = now, 0
	}
	if l.used >= perSecond {
		l.dropped++
		return false, 0
	}
	l.used++
	held := l.dropped
	l.dropped = 0
	return true, held
}

// request logs one request and its outcome.
func (d *Device) request(peer netip.AddrPort, version, op string, count int, oid, result string) {
	ok, held := d.log.allow()
	if !ok || d.emit == nil {
		return
	}
	d.emit(map[string]any{"event": "agent_request", "device": d.cfg.ID, "peer": peer.String(),
		"version": version, "op": op, "count": count, "oid": oid, "result": result, "held": held})
}

// wrote reports points a manager changed with SET.
func (d *Device) wrote(peer netip.AddrPort, items []*entry) {
	if len(items) == 0 || d.emit == nil {
		return
	}
	points := make([]any, len(items))
	d.mu.Lock()
	for index, item := range items {
		points[index] = map[string]any{"oid": item.point.OID, "type": item.point.Type, "value": item.point.Value,
			"hex": item.point.Hex}
	}
	d.mu.Unlock()
	d.emit(map[string]any{"event": "agent_written", "device": d.cfg.ID, "peer": peer.String(), "points": points})
}

func first(p proto.PDU) string {
	if len(p.VarBinds) == 0 {
		return ""
	}
	return p.VarBinds[0].OID.String()
}

var statusNames = map[int]string{proto.TooBig: "tooBig", proto.NoSuchName: "noSuchName", proto.BadValue: "badValue",
	proto.ReadOnly: "readOnly", proto.GenErr: "genErr", proto.NoAccess: "noAccess", proto.WrongType: "wrongType",
	proto.WrongLength: "wrongLength", proto.WrongValue: "wrongValue", proto.NoCreation: "noCreation",
	proto.NotWritable: "notWritable", proto.AuthorizationError: "authorizationError"}

// resultText is "ok", "ok, 3 values" or the error status and index.
func resultText(response proto.PDU) string {
	if response.ErrorStatus != proto.NoError {
		name := statusNames[response.ErrorStatus]
		if name == "" {
			name = fmt.Sprintf("error %d", response.ErrorStatus)
		}
		return fmt.Sprintf("%s (item %d)", name, response.ErrorIndex)
	}
	return fmt.Sprintf("ok, %d values", len(response.VarBinds))
}
