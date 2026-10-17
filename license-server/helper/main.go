// Command license-helper is the owner's portable ScadaBench license tool.
//
// Unsealed (as built) it opens first-time setup: choose a password, and a sealed copy
// is written to the Desktop. Sealed, it shows one password box and two buttons, fully
// offline: make a license for this PC, or remove the license from this PC.
// See docs/specs/license-helper.md.
package main

func main() {
	program, vault, err := ownVault()
	if err != nil {
		showError("Cannot read this program file: " + err.Error())
		return
	}
	if vault == nil {
		err = setupWindow(program)
	} else {
		err = helperWindow(vault)
	}
	if err != nil {
		showError(err.Error())
	}
}
