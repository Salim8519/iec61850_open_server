package main

import (
	"errors"
	"path/filepath"
	"time"
)

// Control IDs (also used by the end-to-end test to drive the real window).
const (
	idStatus   = 10
	idPassword = 12
	idMake     = 13
	idRemove   = 14
	idResult   = 15
	idCopy     = 16
	idFirst    = 22
	idSecond   = 24
	idCreate   = 25
	idCreated  = 26
	idUpdate   = 28
)

const (
	editStyle   = esPassword | esAutoHScroll | wsTabStop
	buttonStyle = wsTabStop
)

// helperWindow is the sealed helper: a password box, Make / Remove, and Copy.
func helperWindow(vault []byte) error {
	controls := []control{
		{id: idStatus, class: "STATIC", x: 16, y: 16, w: 488, h: 28},
		{id: 11, class: "STATIC", text: "Password:", x: 16, y: 64, w: 90, h: 24},
		{id: idPassword, class: "EDIT", style: editStyle, exStyle: wsExClientEdge, x: 110, y: 60, w: 394, h: 28},
		{id: idMake, class: "BUTTON", text: "Make license for this PC", style: buttonStyle, x: 16, y: 104, w: 240, h: 44},
		{id: idRemove, class: "BUTTON", text: "Remove license from this PC", style: buttonStyle, x: 264, y: 104, w: 240, h: 44},
		{id: idCopy, class: "BUTTON", text: "Copy license text", style: buttonStyle, x: 16, y: 156, w: 488, h: 36},
		{id: idResult, class: "STATIC", x: 16, y: 204, w: 488, h: 68},
	}
	press := func(action func([]byte, string, time.Time) (string, error)) func(*window) {
		return func(w *window) {
			setText(w.item(idResult), "Checking the password…")
			procUpdateWindow.Call(w.item(idResult))
			text, err := action(vault, getText(w.item(idPassword)), time.Now())
			setText(w.item(idResult), outcome(text, err))
			setText(w.item(idStatus), statusText())
		}
	}
	clicks := map[int]func(*window){idMake: press(makeLicenseHere), idRemove: press(removeLicenseHere),
		idCopy: func(w *window) {
			text, err := licenseText()
			if err == nil {
				err = setClipboardText(w.hwnd, text)
			}
			setText(w.item(idResult), outcome("Copied the license text. Paste it with Ctrl+V,\n"+
				"for example in ScadaBench > \"Server & offline options\".", err))
		}}
	return runWindow("ScadaBench License Helper", 520, 284, controls, clicks, func(w *window) {
		setText(w.item(idStatus), statusText())
		setText(w.item(idResult), "Type your password, then press a button.")
		procSetFocus.Call(w.item(idPassword))
	})
}

// setupWindow is the first run on the owner's PC: choose the password once.
func setupWindow(program []byte) error {
	publicFile, err := ownerPublicFile()
	if err != nil {
		return err
	}
	desktop, err := desktopFolder()
	if err != nil {
		return err
	}
	controls := []control{
		{id: 20, class: "STATIC", text: "New helper: choose a password (at least 8 characters) and press Create.\n" +
			"Your password never expires.", x: 16, y: 14, w: 488, h: 48},
		{id: 21, class: "STATIC", text: "Password:", x: 16, y: 74, w: 110, h: 24},
		{id: idFirst, class: "EDIT", style: editStyle, exStyle: wsExClientEdge, x: 130, y: 70, w: 374, h: 28},
		{id: 23, class: "STATIC", text: "Type it again:", x: 16, y: 112, w: 110, h: 24},
		{id: idSecond, class: "EDIT", style: editStyle, exStyle: wsExClientEdge, x: 130, y: 108, w: 374, h: 28},
		{id: idCreate, class: "BUTTON", text: "Create my helper", style: buttonStyle, x: 16, y: 148, w: 488, h: 40},
		{id: 27, class: "STATIC", text: "Already have a helper on the Desktop? Get this new version, same password:",
			x: 16, y: 204, w: 488, h: 24},
		{id: idUpdate, class: "BUTTON", text: "Update my helper (keeps my password)", style: buttonStyle,
			x: 16, y: 232, w: 488, h: 40},
		{id: idCreated, class: "STATIC", x: 16, y: 284, w: 488, h: 72},
	}
	working := func(w *window, text string) {
		setText(w.item(idCreated), text)
		procUpdateWindow.Call(w.item(idCreated))
	}
	clicks := map[int]func(*window){
		idCreate: func(w *window) {
			working(w, "Creating your helper…")
			text, err := createHelper(program, getText(w.item(idFirst)), getText(w.item(idSecond)), desktop, publicFile)
			setText(w.item(idCreated), outcome(text, err))
		},
		idUpdate: func(w *window) {
			working(w, "Updating your helper…")
			text, err := updateHelper(program, filepath.Join(desktop, sealedName),
				filepath.Join(filepath.Dir(publicFile), "helper-backups"), time.Now())
			setText(w.item(idCreated), outcome(text, err))
		},
	}
	return runWindow("ScadaBench License Helper - setup", 520, 368, controls, clicks, func(w *window) {
		procSetFocus.Call(w.item(idFirst))
	})
}

// outcome turns an action result into the words shown under the buttons.
func outcome(text string, err error) string {
	switch {
	case errors.Is(err, errWrongPassword):
		return "Not done: wrong password. Nothing was changed."
	case err != nil:
		return "Not done: " + err.Error() + "."
	}
	return text
}
