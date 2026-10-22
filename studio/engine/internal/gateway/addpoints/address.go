package addpoints

import (
	"strconv"

	"studio-engine/internal/gateway"
)

// addresser hands out free addresses for new points, never one an existing or earlier new
// point uses. Commands are paired with their status point: both get the same distance from
// the start of their blocks (IEC 104: double point 20003 and its command 50003), so a SCADA
// engineer finds the pair at a glance.
type addresser struct {
	protocol string
	used     map[gateway.Cell]bool
}

func newAddresser(protocol string, current []any) *addresser {
	return &addresser{protocol: protocol, used: gateway.UsedCells(protocol, current)}
}

// free reports whether a point of kind fits at address (in range, cells not used).
func (a *addresser) free(kind string, address int64) bool {
	bounds := gateway.AddressRange[a.protocol]
	cells := gateway.Cells(a.protocol, kind, address)
	return address >= bounds[0] && cells[len(cells)-1].Address <= bounds[1] && !gateway.AnyUsed(a.used, cells)
}

func (a *addresser) take(kind string, address int64) {
	for _, cell := range gateway.Cells(a.protocol, kind, address) {
		a.used[cell] = true
	}
}

// next takes the first free address of kind's block.
func (a *addresser) next(kind string) (int64, bool) {
	at, ok := gateway.FreeAddress(a.protocol, kind, gateway.BlockStart(a.protocol, kind), a.used)
	if ok {
		a.take(kind, at)
	}
	return at, ok
}

// claim takes a typed address, or says plainly why it cannot be used.
func (a *addresser) claim(kind string, address int64) string {
	bounds := gateway.AddressRange[a.protocol]
	if address < bounds[0] || address > bounds[1] {
		return "the address must be a whole number from " + strconv.FormatInt(bounds[0], 10) + " to " +
			strconv.FormatInt(bounds[1], 10)
	}
	if !a.free(kind, address) {
		return "address " + strconv.FormatInt(address, 10) + " is already used by another point"
	}
	a.take(kind, address)
	return ""
}

// follow takes the address of kind that pairs with a point of otherKind at otherAt, or the
// next free one when that is used.
func (a *addresser) follow(kind, otherKind string, otherAt int64) (int64, bool) {
	want := gateway.BlockStart(a.protocol, kind) + otherAt - gateway.BlockStart(a.protocol, otherKind)
	if a.free(kind, want) {
		a.take(kind, want)
		return want, true
	}
	return a.next(kind)
}

// pair takes two new addresses at the same distance from their blocks' starts: the first
// distance free for both, else the next free address of each.
func (a *addresser) pair(statusKind, commandKind string) (int64, int64, bool) {
	statusStart := gateway.BlockStart(a.protocol, statusKind)
	commandStart := gateway.BlockStart(a.protocol, commandKind)
	high := gateway.AddressRange[a.protocol][1]
	for offset := int64(0); statusStart+offset <= high && commandStart+offset <= high && offset < 100000; offset++ {
		status, command := statusStart+offset, commandStart+offset
		if a.free(statusKind, status) {
			a.take(statusKind, status)
			if a.free(commandKind, command) {
				a.take(commandKind, command)
				return status, command, true
			}
			a.release(statusKind, status)
		}
	}
	status, ok := a.next(statusKind)
	if !ok {
		return 0, 0, false
	}
	command, ok := a.next(commandKind)
	return status, command, ok
}

func (a *addresser) release(kind string, address int64) {
	for _, cell := range gateway.Cells(a.protocol, kind, address) {
		delete(a.used, cell)
	}
}
