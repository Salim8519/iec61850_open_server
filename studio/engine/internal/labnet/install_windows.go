//go:build windows

package labnet

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// Installed is what Install did: the card's GUID and whether this run created it.
type Installed struct {
	Card    string `json:"card"`
	Created bool   `json:"created"`
	Program string `json:"program"`
	Note    string `json:"note,omitempty"` // a skipped optional setting and Windows' reason
}

// Install sets up the lab network (needs administrator rights): the card (kept when the
// service's key already names an existing one, so Repair never makes a second card), its
// settings, the service program (source, scadabench_labsvc.exe) in Program Files and the service
// itself, started. A service of an older installation is pointed at the new program.
func Install(source string) (Installed, error) {
	result := Installed{Program: InstalledProgram()}
	if card, err := (WindowsOps{}).Card(); err == nil {
		result.Card = card
	} else {
		card, err := CreateCard()
		if err != nil {
			return result, errors.New("Windows did not create the loopback card: " + err.Error())
		}
		result.Card, result.Created = card, true
	}
	note, err := configureCard(result.Card)
	if err != nil {
		return result, err
	}
	result.Note = note
	manager, err := mgr.Connect()
	if err != nil {
		return result, err
	}
	defer manager.Disconnect()
	service, err := manager.OpenService(ServiceName)
	if err == nil {
		stopService(service)
	}
	if err := copyProgram(source, result.Program); err != nil {
		if service != nil {
			service.Start() // the old program keeps working
			service.Close()
		}
		return result, errors.New("the service program could not be copied: " + err.Error())
	}
	if service != nil {
		if err := pointAt(service, result.Program); err != nil {
			service.Close()
			return result, err
		}
	} else {
		service, err = manager.CreateService(ServiceName, result.Program, mgr.Config{StartType: mgr.StartAutomatic,
			DisplayName: ServiceDisplay, Description: "Adds and removes the simulated IEDs' addresses on the " +
				"ScadaBench Lab network card. Removed with \"Remove the lab network card\" in ScadaBench."}, "service")
		if err != nil {
			return result, err
		}
	}
	defer service.Close()
	service.SetRecoveryActions([]mgr.RecoveryAction{{Type: mgr.ServiceRestart, Delay: time.Second}}, 86400)
	if err := writeCard(result.Card); err != nil {
		return result, err
	}
	if err := service.Start(); err != nil && !isRunning(service) {
		return result, errors.New("the lab network service did not start: " + err.Error())
	}
	return result, waitState(service, svc.Running)
}

// Uninstall removes the service, its program and, when removeCard is set, the card.
func Uninstall(removeCard bool) error {
	card, _ := readCard()
	if manager, err := mgr.Connect(); err == nil {
		if service, err := manager.OpenService(ServiceName); err == nil {
			stopService(service)
			service.Delete()
			service.Close()
		}
		manager.Disconnect()
	}
	os.RemoveAll(filepath.Dir(InstalledProgram()))
	if removeCard && card != "" {
		return RemoveCard(card)
	}
	return nil
}

// configureCard names the card, keeps it last in route choice, turns DHCP and IPv6 off and stops
// the DNS Client registering its addresses (it never uses DNS; ADR-0128). Only the DNS step may
// fail (ADR-0134): its reason comes back as the note and the other steps still run and must work.
func configureCard(guid string) (note string, err error) {
	command := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", cardScript(guid))
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := command.CombinedOutput()
	if err != nil {
		return "", errors.New("the card's settings could not be set: " + tail(string(output)))
	}
	return dnsNote(string(output)), nil
}

func tail(text string) string {
	lines := []rune(text)
	if len(lines) > 300 {
		lines = lines[len(lines)-300:]
	}
	return string(lines)
}

// pointAt makes an existing service run the program (a service made before ADR-0131 runs the
// installer's own copy) and removes that old copy.
func pointAt(service *mgr.Service, program string) error {
	config, err := service.Config()
	if err != nil {
		return err
	}
	quoted := syscall.EscapeArg(program) + " service" // as mgr.CreateService writes it
	if config.BinaryPathName != quoted {
		config.BinaryPathName = quoted
		if err := service.UpdateConfig(config); err != nil {
			return errors.New("the lab network service could not be pointed at its program: " + err.Error())
		}
	}
	old := filepath.Join(filepath.Dir(program), oldServiceProgram)
	if old != program {
		os.Remove(old)
	}
	return nil
}

// copyProgram copies the service program, retrying for a few seconds while Windows or an
// antivirus still holds the old file.
func copyProgram(source, target string) error {
	var err error
	for attempt := 0; attempt < 20; attempt++ {
		if err = copyOnce(source, target); err == nil {
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return err
}

func copyOnce(source, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(target)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func writeCard(guid string) error {
	key, _, err := registry.CreateKey(registry.LOCAL_MACHINE, ParametersKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	return key.SetStringValue(CardValue, guid)
}

func readCard() (string, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, ParametersKey, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer key.Close()
	value, _, err := key.GetStringValue(CardValue)
	return value, err
}

// stopService stops the service and waits until its process has exited: Windows reports
// "stopped" a moment before the program file is free to be replaced (Repair failed on that).
func stopService(service *mgr.Service) {
	var pid uint32
	if status, err := service.Query(); err == nil {
		pid = status.ProcessId
	}
	service.Control(svc.Stop)
	waitState(service, svc.Stopped)
	if pid == 0 {
		return
	}
	if process, err := windows.OpenProcess(windows.SYNCHRONIZE, false, pid); err == nil {
		windows.WaitForSingleObject(process, 10000)
		windows.CloseHandle(process)
	}
}

func isRunning(service *mgr.Service) bool {
	status, err := service.Query()
	return err == nil && status.State == svc.Running
}

func waitState(service *mgr.Service, wanted svc.State) error {
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		if status, err := service.Query(); err == nil && status.State == wanted {
			return nil
		}
	}
	return errors.New("the lab network service did not reach the expected state in 15 seconds")
}
