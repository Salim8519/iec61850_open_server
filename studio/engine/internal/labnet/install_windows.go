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

	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// Installed is what Install did: the card's GUID and whether this run created it.
type Installed struct {
	Card    string `json:"card"`
	Created bool   `json:"created"`
	Program string `json:"program"`
}

// Install sets up the lab network (needs administrator rights): the card (kept when the
// service's key already names an existing one, so Repair never makes a second card), its
// settings, the service program in Program Files and the service itself, started.
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
	if err := configureCard(result.Card); err != nil {
		return result, err
	}
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
			service.Close()
		}
		return result, errors.New("the service program could not be copied: " + err.Error())
	}
	if service == nil {
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

// configureCard names the card, keeps it last in route choice, turns DHCP and IPv6 off.
func configureCard(guid string) error {
	script := "$ErrorActionPreference='Stop'; $a = Get-NetAdapter -IncludeHidden | Where-Object { " +
		"$_.InterfaceGuid -eq '" + guid + "' }; if (-not $a) { throw 'card not found' }; " +
		"if ($a.Name -ne '" + CardName + "') { Rename-NetAdapter -InputObject $a -NewName '" + CardName + "' }; " +
		"Set-NetIPInterface -InterfaceIndex $a.ifIndex -AddressFamily IPv4 -InterfaceMetric 9999 -Dhcp Disabled; " +
		"Disable-NetAdapterBinding -Name '" + CardName + "' -ComponentID ms_tcpip6"
	command := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if output, err := command.CombinedOutput(); err != nil {
		return errors.New("the card's settings could not be set: " + tail(string(output)))
	}
	return nil
}

func tail(text string) string {
	lines := []rune(text)
	if len(lines) > 300 {
		lines = lines[len(lines)-300:]
	}
	return string(lines)
}

func copyProgram(source, target string) error {
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

func stopService(service *mgr.Service) {
	service.Control(svc.Stop)
	waitState(service, svc.Stopped)
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
