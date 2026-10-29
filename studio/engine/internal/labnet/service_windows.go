//go:build windows

package labnet

import (
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
)

// ProtocolVersion is the pipe protocol the app and the service speak (2: several addresses in one
// request, ADR-0128).
const ProtocolVersion = "2"

// RunService runs the lab network service under the Service Control Manager.
func RunService() error {
	return svc.Run(ServiceName, serviceHandler{})
}

type serviceHandler struct{}

// Execute serves the pipe until Windows stops the service.
func (serviceHandler) Execute(_ []string, requests <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending}
	stop := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- Serve(stop, func(request Request) Reply { return Handle(WindowsOps{}, request, ProtocolVersion) })
	}()
	status <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	for {
		select {
		case change := <-requests:
			switch change.Cmd {
			case svc.Interrogate:
				status <- change.CurrentStatus
			case svc.Stop, svc.Shutdown:
				status <- svc.Status{State: svc.StopPending}
				close(stop)
				<-done
				return false, 0
			}
		case err := <-done:
			if err != nil {
				return false, 1
			}
			return false, 0
		}
	}
}

// ServiceState says whether the service is installed and running; any user may ask.
func ServiceState() (installed, running bool) {
	manager, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return false, false
	}
	defer windows.CloseServiceHandle(manager)
	name, _ := windows.UTF16PtrFromString(ServiceName)
	service, err := windows.OpenService(manager, name, windows.SERVICE_QUERY_STATUS)
	if err != nil {
		return false, false
	}
	defer windows.CloseServiceHandle(service)
	var state windows.SERVICE_STATUS
	if err := windows.QueryServiceStatus(service, &state); err != nil {
		return true, false
	}
	return true, state.CurrentState == windows.SERVICE_RUNNING
}

// servicePID is the process ID Windows runs the service in (0 when it is not running).
func servicePID() (uint32, error) {
	manager, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return 0, err
	}
	defer windows.CloseServiceHandle(manager)
	name, _ := windows.UTF16PtrFromString(ServiceName)
	service, err := windows.OpenService(manager, name, windows.SERVICE_QUERY_STATUS)
	if err != nil {
		return 0, err
	}
	defer windows.CloseServiceHandle(service)
	var status windows.SERVICE_STATUS_PROCESS
	var needed uint32
	if err := windows.QueryServiceStatusEx(service, windows.SC_STATUS_PROCESS_INFO,
		(*byte)(unsafe.Pointer(&status)), uint32(unsafe.Sizeof(status)), &needed); err != nil {
		return 0, err
	}
	return status.ProcessId, nil
}
