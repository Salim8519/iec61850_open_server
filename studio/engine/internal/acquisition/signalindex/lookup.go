package signalindex

import "studio-engine/internal/pyjson"

// Signal returns the signal object with reference ref, like Python's
// {signal["ref"]: signal for signal in ied["signals"]}[ref] (the last duplicate wins).
func (ix *Index) Signal(ref string) *pyjson.Object { return ix.last[ref] }
