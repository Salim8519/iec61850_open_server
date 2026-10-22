package gateway

// Exported address helpers for the Go-only "Add points" planner (internal/gateway/addpoints,
// ADR-0105). They expose the existing block, cell and free-address rules unchanged, so new
// points are numbered exactly the way Propose and Add point number them.

// BlockStart is the first address of the block a kind's points start from.
func BlockStart(protocol, kind string) int64 {
	block := kind
	if IsCommandKind(kind) {
		block = "command"
	}
	return startAddress[protocol][block]
}

// Cells are the (space, address) pairs a point of kind at address occupies.
func Cells(protocol, kind string, address int64) []Cell { return cells(protocol, kind, address) }

// UsedCells are the cells the well-formed points of a list occupy.
func UsedCells(protocol string, list []any) map[Cell]bool { return usedCells(protocol, list) }

// FreeAddress is the first address from "from" whose cells are free and in range.
func FreeAddress(protocol, kind string, from int64, used map[Cell]bool) (int64, bool) {
	return freeAddress(protocol, kind, from, used)
}

// AnyUsed reports whether one of the cells is taken.
func AnyUsed(used map[Cell]bool, list []Cell) bool { return anyUsed(used, list) }
