//go:build windows

package labnet

import (
	"errors"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// TurnOn starts the installed network service and makes it start with Windows again, when another
// program set it to manual or disabled (ADR-0136). Nothing else changes: the card, the program and
// the approvals stay as they are. Administrators only (the app asks Windows first).
func TurnOn() error {
	manager, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer manager.Disconnect()
	service, err := manager.OpenService(ServiceName)
	if err != nil {
		return errors.New("the ScadaBench network service is not installed; use Create on the lab network page")
	}
	defer service.Close()
	config, err := service.Config()
	if err != nil {
		return err
	}
	if config.StartType != mgr.StartAutomatic {
		config.StartType = mgr.StartAutomatic
		if err := service.UpdateConfig(config); err != nil {
			return errors.New("the ScadaBench network service could not be set to start with Windows: " + err.Error())
		}
	}
	if err := service.Start(); err != nil && !isRunning(service) {
		return errors.New("the ScadaBench network service did not start: " + err.Error())
	}
	return waitState(service, svc.Running)
}
