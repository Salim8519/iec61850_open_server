package signalindex

import (
	"strings"
	"sync"
)

// nodeTable is the logical node ("LD/LN") of every signal, made on the first request that
// filters or counts by node (ADR-0113), so Build and the other requests stay as they were.
type nodeTable struct {
	once  sync.Once
	ids   []int32          // node position of every signal
	names []string         // logical nodes in model order
	index map[string]int32 // position of each node in names
}

// logicalNodes returns the index's node table, building it once.
func (ix *Index) logicalNodes() *nodeTable {
	table := &ix.nodes
	table.once.Do(func() {
		table.ids = make([]int32, len(ix.signals))
		table.index = map[string]int32{}
		for position := range ix.signals {
			key := logicalNodeOf(ix.signals[position].ref)
			id, found := table.index[key]
			if !found {
				id = int32(len(table.names))
				table.names = append(table.names, key)
				table.index[key] = id
			}
			table.ids[position] = id
		}
	})
	return table
}

// logicalNodeOf is the reference up to the first "." after its "/" ("LD/LN"), or the whole
// reference when it has no such ".".
func logicalNodeOf(ref string) string {
	start := strings.IndexByte(ref, '/') + 1
	if dot := strings.IndexByte(ref[start:], '.'); dot >= 0 {
		return ref[:start+dot]
	}
	return ref
}
