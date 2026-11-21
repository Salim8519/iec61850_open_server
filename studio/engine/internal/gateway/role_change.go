package gateway

// Role switching (ADR-0143). The station gateway (studio_gateway.exe) and the SCADA master
// (studio_master.exe) are separate programs, so both may run at the same time: "Act as" picks
// which one the screen shows. The protocol converter uses both programs together (the master
// reads the device, the gateway serves its points), so it stays on its own.

// Messages of a refused role change; the screen greys out the same buttons first.
const (
	RefuseLeaveConverter = "Stop the protocol converter (gateway and master link) before switching roles"
	RefuseEnterConverter = "Stop the gateway and disconnect the master before switching to Protocol converter"
)

// RoleChangeRefusal is why the role cannot change from current to next right now, or "".
// gatewayRunning and masterBusy say whether each program runs.
func RoleChangeRefusal(current, next string, gatewayRunning, masterBusy bool) string {
	if current == next || (!gatewayRunning && !masterBusy) {
		return ""
	}
	if current == "converter" {
		return RefuseLeaveConverter
	}
	if next == "converter" {
		return RefuseEnterConverter
	}
	return ""
}

// RoleLocks maps each role that cannot be picked right now to the reason; empty when every
// role can be picked (the Go-only "role_locks" of /activity is left out then).
func RoleLocks(current string, gatewayRunning, masterBusy bool) map[string]string {
	locks := map[string]string{}
	for _, role := range Roles {
		if reason := RoleChangeRefusal(current, role, gatewayRunning, masterBusy); reason != "" {
			locks[role] = reason
		}
	}
	return locks
}
