package acquisition

import (
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/apperr"
)

// libiec61850 IedClientError codes that mean the device sent nothing back for a point.
const (
	clientNotConnected   = 1  // IED_ERROR_NOT_CONNECTED
	clientConnectionLost = 3  // IED_ERROR_CONNECTION_LOST
	clientTimeout        = 20 // IED_ERROR_TIMEOUT
)

// errSilent is a read the device answered for no point at all. The connection can stay open
// for minutes after a device froze or its cable was pulled, and reports simply stop; without
// this the session kept saying "Receiving live values" with frozen values (fix plan A18).
var errSilent = apperr.RuntimeError("The device did not answer a read (no reply in time)")

// silentAnswer reports whether a studio_live read answer has at least one point and every
// point failed without a reply. A point the device refuses (no such object, access denied)
// or any value is an answer: the device is alive.
func silentAnswer(answer any) bool {
	entries := pyv.Objects(pyv.GetList(pyv.Obj(answer), "values"))
	if len(entries) == 0 {
		return false
	}
	for _, entry := range entries {
		code, isCode := pyv.Int64(pyv.Get(entry, "error"))
		if !isCode || (code != clientNotConnected && code != clientConnectionLost && code != clientTimeout) {
			return false
		}
	}
	return true
}
